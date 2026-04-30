package app

import (
	db "my-idm/db/sqlc"
	"my-idm/internal/identity"
	"net/http"
)

// NewHandler assembles the application's routes and middleware.
func NewHandler(queries *db.Queries) http.Handler {
	mux := http.NewServeMux()

	// Register Features
	identity.Register(mux, queries)

	// Here you can wrap the mux with global middleware (logging, auth, etc.)
	return mux
}
