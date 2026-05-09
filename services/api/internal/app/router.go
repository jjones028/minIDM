package app

import (
	"crypto/rsa"
	"encoding/json"
	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
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

	// Audit log route (requires audit_log:read) + returns the shared Auditor.
	protectAuditRead := chain(rbac.Authenticate(queries), rbac.Require("audit_log", "read", queries))
	auditor := audit.Register(mux, queries, protectAuditRead)

	// Public: session management
	session.Register(mux, queries, secureCookies, auditor)

	// Auth-only: /api/me lets the frontend check whether the cookie is still valid.
	authenticate := rbac.Authenticate(queries)
	mux.Handle("GET /api/me", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := rbac.IdentityFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id})
	})))

	// Protected: identity routes
	protectIdentityRead := chain(rbac.Authenticate(queries), rbac.Require("identity", "read", queries))
	protectIdentityWrite := chain(rbac.Authenticate(queries), rbac.Require("identity", "write", queries))
	identity.Register(mux, queries, protectIdentityRead, protectIdentityWrite, auditor)

	// Protected: role management routes
	protectRoleRead := chain(rbac.Authenticate(queries), rbac.Require("role", "read", queries))
	protectRoleWrite := chain(rbac.Authenticate(queries), rbac.Require("role", "write", queries))
	rbac.RegisterRoleRoutes(mux, queries, protectRoleRead, protectRoleWrite, auditor)

	// OAuth2/OIDC: discovery + protocol endpoints (public) + admin client CRUD (RBAC)
	protectClientRead := chain(rbac.Authenticate(queries), rbac.Require("oauth2_client", "read", queries))
	protectClientWrite := chain(rbac.Authenticate(queries), rbac.Require("oauth2_client", "write", queries))
	oauth2.Register(mux, queries, signingKey, issuer, protectClientRead, protectClientWrite, auditor, rbac.Authenticate(queries))

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
