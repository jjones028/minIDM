package main

import (
	"context"
	"encoding/json"
	db "my-idm/db/sqlc"
	"my-idm/internal/identity"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()

	// Connect to Postgres using pgx
	conn, _ := pgx.Connect(ctx, "postgres://user:pass@localhost:5432/idm")
	defer conn.Close(ctx)

	// Instantiate the generated SQL queries
	queries := db.New(conn)

	// Instantiate our Business Service
	identitySvc := identity.NewService(queries)

	// Setup Chi Router (Functional style)
	// r := chi.NewRouter()
	// r.Post("/api/register", func(w http.ResponseWriter, r *http.Request) {
	//     // Decode request and call identitySvc.Register...
	// })

	// http.ListenAndServe(":8080", r)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		// Decode request and call identitySvc.Register...
		// 1. Decode
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid_request", http.StatusBadRequest)
			return
		}

		// 2. Validate basic constraints before hitting DB
		if len(req.Password) < 8 {
			http.Error(w, "password_too_short", http.StatusUnprocessableEntity)
			return
		}
		registerCmd := identity.RegisterCommand{
			Email:    req.Email,
			Password: req.Password,
		}

		_, err := identitySvc.Register(ctx, registerCmd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)

	})

	http.ListenAndServe(":8080", mux)
}
