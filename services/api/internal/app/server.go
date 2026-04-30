package app

import (
	"context"
	"encoding/json"
	"fmt"
	db "my-idm/db/sqlc"
	"my-idm/internal/identity"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func Run(ctx context.Context, dbURL string) error {
	// Connect to Postgres using pgx
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Instantiate the generated SQL queries
	queries := db.New(conn)

	// Instantiate our Business Service
	identitySvc := identity.NewService(queries)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid_request", http.StatusBadRequest)
			return
		}

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

	fmt.Println("🚀 API server listening on :8080")
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down server...")
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
