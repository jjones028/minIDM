# minidm

A simple Identity Management (IDM) system, organized by features.

## Project Structure

- `services/api`: Go-based backend API.
  - `internal/app`: Core server infrastructure and router assembly.
  - `internal/identity`: Identity feature containing API handlers, CQRS commands, and business logic.
- `web`: React-based frontend application (Vite, Tailwind CSS, Shadcn/ui).

## Build System

This project uses [go-task](https://taskfile.dev/) for orchestration. The `Taskfile.yml` in the root directory manages building, development, and code generation for both backend and frontend.

- **Run all:** `task`
- **Build frontend:** `task web:build`
- **Build backend:** `task api:build`

## Features

- Identity registration with secure password hashing using Argon2id.
- PostgreSQL database integration using `sqlc` for type-safe queries.
- Database migrations managed via `goose`.

## Tech Stack

- **Backend:** Go 1.26.1, `pgx/v5`
- **Frontend:** React 19, Vite, Tailwind CSS v4, Radix UI.
- **Database:** PostgreSQL
- **Database Tools:**
  - [sqlc](https://sqlc.dev/) - Type-safe Go from SQL.
  - [goose](https://github.com/pressly/goose) - Database migrations.
- **Crypto:** [Argon2id](https://en.wikipedia.org/wiki/Argon2) for password hashing.
- **Architecture:** CQRS (Command Query Responsibility Segregation) with feature-based organization.

## Getting Started

### Prerequisites

- Go 1.26.1+
- Node.js (for frontend)
- PostgreSQL
- [Task](https://taskfile.dev/)
- [sqlc](https://sqlc.dev/docs/install)
- [goose](https://github.com/pressly/goose#install)

### Development Mode (Recommended)

The project includes a "Dev Mode" that manages your database using Testcontainers and runs the API server directly.

1.  **Ensure Docker or Podman Desktop is running.**
2.  **Start development:**
    ```bash
    task dev
    ```

This will manage database lifecycle, generate necessary code, and run the backend server.

### Manual Setup

1.  **Install backend dependencies:**
    ```bash
    cd services/api
    go mod download
    ```
2.  **Install frontend dependencies:**
    ```bash
    cd web
    npm install
    ```
3.  **Run backend server:**
    ```bash
    go run services/api/cmd/server/main.go
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
