package app

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"
	"minIDM/internal/identity"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

func Run(ctx context.Context, dbURL string) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	queries := db.New(conn)

	if email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL"); email != "" {
		password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
		if password == "" {
			return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD is required when BOOTSTRAP_ADMIN_EMAIL is set")
		}
		if err := identity.Bootstrap(ctx, queries, email, password); err != nil {
			return fmt.Errorf("bootstrap: %w", err)
		}
	}

	handler := NewHandler(queries)

	fmt.Println("API server listening on :8080")
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
