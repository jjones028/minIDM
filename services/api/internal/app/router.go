package app

import (
	db "my-idm/db/sqlc"
	"my-idm/internal/identity"
	"net/http"
)

// NewHandler assembles the application's routes and middleware.
func NewHandler(queries *db.Queries) http.Handler {
	mux := http.NewServeMux()

	// Register API Features
	identity.Register(mux, queries)

	// Catch-all for Frontend
	mux.Handle("/", StaticHandler())

	// Simple CORS Middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		mux.ServeHTTP(w, r)
	})
}
