package app

import (
	"crypto/rsa"
	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/httputil"
	"minIDM/internal/identity"
	"minIDM/internal/oauth2"
	"minIDM/internal/rbac"
	"minIDM/internal/session"
	"net/http"
	"os"
)

// NewHandler assembles the application's routes and middleware.
func NewHandler(queries *db.Queries, signingKey *rsa.PrivateKey, issuer string) http.Handler {
	mux := http.NewServeMux()

	secureCookies := os.Getenv("SECURE_COOKIES") == "true"
	registrationEnabled := os.Getenv("REGISTRATION_ENABLED") == "true"

	// Audit log route (requires audit_log:read) + returns the shared Auditor.
	protectAuditRead := chain(rbac.Authenticate(queries), rbac.Require("audit_log", "read", queries))
	auditor := audit.Register(mux, audit.Config{
		Queries:     queries,
		ProtectRead: protectAuditRead,
	})

	// Public: session management
	session.Register(mux, session.Config{
		Queries:       queries,
		SecureCookies: secureCookies,
		Auditor:       auditor,
	})

	// Auth-only: /api/me lets the frontend check whether the cookie is still valid.
	authenticate := rbac.Authenticate(queries)
	mux.Handle("GET /api/me", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := rbac.IdentityFromContext(r.Context())
		httputil.WriteJSON(w, map[string]any{"id": id})
	})))

	// Public: app configuration for frontend feature flags.
	// ?client_id=xxx narrows registration_enabled to the per-client setting.
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		effective := registrationEnabled
		if effective {
			if clientID := r.URL.Query().Get("client_id"); clientID != "" {
				client, err := queries.GetOAuth2ClientByClientID(r.Context(), clientID)
				if err != nil || !client.IsEnabled {
					effective = false
				} else {
					effective = client.AllowRegistration
				}
			}
		}
		httputil.WriteJSON(w, map[string]bool{
			"registration_enabled": effective,
		})
	})

	// Protected: identity routes
	protectIdentityRead := chain(rbac.Authenticate(queries), rbac.Require("identity", "read", queries))
	protectIdentityWrite := chain(rbac.Authenticate(queries), rbac.Require("identity", "write", queries))
	identity.Register(mux, identity.Config{
		Queries:             queries,
		ProtectRead:         protectIdentityRead,
		ProtectWrite:        protectIdentityWrite,
		Auditor:             auditor,
		RegistrationEnabled: registrationEnabled,
	})

	// Protected: role management routes
	protectRoleRead := chain(rbac.Authenticate(queries), rbac.Require("role", "read", queries))
	protectRoleWrite := chain(rbac.Authenticate(queries), rbac.Require("role", "write", queries))
	rbac.RegisterRoleRoutes(mux, rbac.Config{
		Queries:      queries,
		ProtectRead:  protectRoleRead,
		ProtectWrite: protectRoleWrite,
		Auditor:      auditor,
	})

	// OAuth2/OIDC: discovery + protocol endpoints (public) + admin client CRUD (RBAC)
	protectClientRead := chain(rbac.Authenticate(queries), rbac.Require("oauth2_client", "read", queries))
	protectClientWrite := chain(rbac.Authenticate(queries), rbac.Require("oauth2_client", "write", queries))
	oauth2.Register(mux, oauth2.Config{
		Queries:      queries,
		Key:          signingKey,
		Issuer:       issuer,
		ProtectRead:  protectClientRead,
		ProtectWrite: protectClientWrite,
		Auditor:      auditor,
		Authenticate: rbac.Authenticate(queries),
	})

	// Catch-all for Frontend (must be last)
	mux.Handle("/", StaticHandler())

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// chain composes middlewares left-to-right: the first middleware is outermost.
func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}
