# Build Stage: Frontend
FROM node:23-slim AS frontend-build
WORKDIR /app
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build Stage: Backend
FROM golang:1.26-alpine AS backend-build
WORKDIR /app
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api/ .
# Copy the built frontend into the Go package for embedding
COPY --from=frontend-build /app/dist ./internal/app/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /minIDM ./cmd/server/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /migrate ./cmd/migrate/main.go

# Runtime Stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=backend-build /minIDM .
COPY --from=backend-build /migrate .
EXPOSE 8080
ENTRYPOINT ["/app/minIDM"]

# Migrator Stage — includes goose CLI + SQL migration files.
# Built and pushed separately as minidm-migrate:latest.
# Used by k8s/migrate-job.yaml; never deployed as the app.
FROM golang:1.26-alpine AS migrator
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
COPY services/api/db/migrations /migrations
CMD ["sh", "-c", "goose -dir /migrations postgres \"$DATABASE_URL\" up"]
