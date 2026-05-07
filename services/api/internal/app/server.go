package app

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"
	"minIDM/internal/identity"
	"minIDM/internal/oauth2"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, dbURL string) error {
	conn, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

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

	// Load or generate the RSA signing key for OAuth2/OIDC.
	keyPath := os.Getenv("OAUTH2_KEY_PATH")
	if keyPath == "" {
		keyPath = "oauth2_signing.key"
	}
	signingKey, err := oauth2.LoadOrGenerateRSAKey(keyPath)
	if err != nil {
		return fmt.Errorf("oauth2 signing key: %w", err)
	}

	issuer := os.Getenv("OAUTH2_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}

	handler := NewHandler(queries, signingKey, issuer)

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
