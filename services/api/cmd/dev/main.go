package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"minIDM/internal/app"

	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

func main() {
	ctx := context.Background()

	// Podman Desktop / macOS settings
	// Ryuk often fails on Podman due to network 'bridge' expectations.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	fmt.Println("🚀 Starting development stack via Testcontainers Compose...")

	// Path to compose.yaml in the root
	composeFile, err := filepath.Abs(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		log.Fatalf("failed to get absolute path: %v", err)
	}

	stack, err := compose.NewDockerComposeWith(
		compose.WithStackFiles(composeFile),
		compose.StackIdentifier("minidm"),
	)
	if err != nil {
		log.Fatalf("failed to initialize compose stack: %v", err)
	}

	// Ensure the stack is up
	// Using ForExposedPort is more reliable for reused containers than ForLog
	err = stack.WaitForService("db", wait.ForExposedPort().WithStartupTimeout(60*time.Second)).
		Up(ctx, compose.Wait(true))

	if err != nil {
		// If it's already running, Up might still be successful and we can continue
		fmt.Printf("ℹ️  Note: Compose Up returned: %v (continuing...)\n", err)
	}

	// Get the DB container to extract the mapped port
	container, err := stack.ServiceContainer(ctx, "db")
	if err != nil {
		log.Fatalf("failed to get db container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("failed to get container port: %v", err)
	}

	connStr := fmt.Sprintf("postgres://user:pass@%s:%s/idm?sslmode=disable", host, port.Port())
	fmt.Printf("✅ Postgres is ready at %s\n", connStr)

	// Run migrations
	fmt.Println("🔄 Running migrations...")
	migrationsDir := filepath.Join("db", "migrations")
	cmd := exec.Command("goose", "-dir", migrationsDir, "postgres", connStr, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Migration failed (ensure goose is installed and in PATH): %v\n", err)
	}

	// Start the API server directly in the same process for debugging
	fmt.Println("📡 Starting API server...")
	if err := app.Run(ctx, connStr); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
