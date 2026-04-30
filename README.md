# my-idm

A simple Identity Management (IDM) system, organized by features.

## Project Structure

- `services/api`: Go-based backend API.
  - `internal/app`: Core server infrastructure and router assembly.
  - `internal/identity`: Identity feature containing API handlers, CQRS commands, and business logic.
- `web`: Frontend application (currently empty).

## Features

- Identity registration with secure password hashing using Argon2id.
- PostgreSQL database integration using `sqlc` for type-safe queries.
- Database migrations managed via `goose`.

## Tech Stack

- **Language:** [Go](https://go.dev/) 1.26.1
- **Database:** [PostgreSQL](https://www.postgresql.org/)
- **Database Tools:**
  - [sqlc](https://sqlc.dev/) - Type-safe Go from SQL.
  - [goose](https://github.com/pressly/goose) - Database migrations.
- **Crypto:** [Argon2id](https://en.wikipedia.org/wiki/Argon2) for password hashing.
- **Architecture:** CQRS (Command Query Responsibility Segregation) with feature-based organization.

## Getting Started

### Prerequisites

- Go 1.26.1+
- PostgreSQL
- [sqlc](https://sqlc.dev/docs/install)
- [goose](https://github.com/pressly/goose#install)

### Backend Setup

1.  **Navigate to the API directory:**
    ```bash
    cd services/api
    ```
2.  **Install dependencies:**
    ```bash
    go mod download
    ```
### Development Mode (Recommended)

The project includes a "Dev Mode" that manages your database using Testcontainers and runs the API server directly in your terminal for easy debugging.

1.  **Ensure Docker or Podman Desktop is running.**
2.  **Run the dev command:**
    ```bash
    cd services/api
    go run cmd/dev/main.go
    ```

This will:
- Start PostgreSQL via Compose (using Testcontainers).
- Automatically run migrations via `goose`.
- Start the API server locally on `:8080` with the correct `DATABASE_URL` configured.

### Manual Database Setup (Optional)

For manual control, you can start a PostgreSQL instance using Compose from the root directory:
```bash
docker compose up -d
```
The database will be available at `localhost:5432` with a persistent volume `./postgres_data`.

### Podman Note

If you are using **Podman Desktop**, the project automatically sets `TESTCONTAINERS_RYUK_DISABLED=true` to ensure compatibility with Podman's default networking. If you encounter issues, ensure your Podman machine is started and the `DOCKER_HOST` environment variable is correctly set (Podman Desktop usually handles this).

4.  **Run migrations:**
    ```bash
    goose -dir db/migrations postgres "postgres://user:pass@localhost:5432/idm" up
    ```
5.  **Start the server:**
    ```bash
    go run cmd/server/main.go
    ```


The API will be available at `http://localhost:8080`.

## API Endpoints

### Register Identity

- **Endpoint:** `POST /api/register`
- **Description:** Registers a new user identity.
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword"
  }
  ```
- **Response:**
  - `201 Created` on success.
  - `400 Bad Request` for invalid JSON.
  - `422 Unprocessable Entity` if the password is too short (min 8 characters).
  - `500 Internal Server Error` for database or hashing failures.
