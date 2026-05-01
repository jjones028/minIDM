package app

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"
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

	handler := NewHandler(queries)

	fmt.Println("🚀 API server listening on :8080")
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down server...")
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
