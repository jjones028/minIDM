# my-idm

A simple Identity Management (IDM) system.

## Project Structure

- `services/api`: Go-based backend API.
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

## Getting Started

### Prerequisites

- Go 1.26.1+
- PostgreSQL
- [sqlc](https://sqlc.dev/docs/install) (optional, for code generation)
- [goose](https://github.com/pressly/goose#install) (optional, for migrations)

### Backend Setup

1.  **Navigate to the API directory:**
    ```bash
    cd services/api
    ```
2.  **Install dependencies:**
    ```bash
    go mod download
    ```
3.  **Database Configuration:**
    The database connection string is currently configured in `main.go`. Ensure you have a running PostgreSQL instance and update the connection string if necessary:
    ```go
    conn, _ := pgx.Connect(ctx, "postgres://user:pass@localhost:5432/idm")
    ```
4.  **Run migrations:**
    ```bash
    goose -dir db/migrations postgres "postgres://user:pass@localhost:5432/idm" up
    ```
5.  **Generate SQL code (optional):**
    If you modify the SQL queries in `db/queries/`, regenerate the Go code using:
    ```bash
    sqlc generate
    ```
6.  **Start the server:**
    ```bash
    go run main.go
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
