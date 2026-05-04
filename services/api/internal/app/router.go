package app

import (
	db "minIDM/db/sqlc"
	"minIDM/internal/identity"
	"minIDM/internal/rbac"
	"minIDM/internal/session"
	"net/http"
)

// NewHandler assembles the application's routes and middleware.
func NewHandler(queries *db.Queries) http.Handler {
	mux := http.NewServeMux()

	// Public: session management
	session.Register(mux, queries)

	// Protected: identity routes require authentication + identity:read permission
	protect := chain(
		rbac.Authenticate(queries),
		rbac.Require("identity", "read", queries),
	)
	identity.Register(mux, queries, protect)

	// Catch-all for Frontend
	mux.Handle("/", StaticHandler())

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
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
