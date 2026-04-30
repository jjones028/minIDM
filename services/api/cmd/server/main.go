package main

import (
	"context"
	"log"
	"my-idm/internal/app"
	"os"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:pass@localhost:5432/idm"
	}

	if err := app.Run(ctx, dbURL); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
