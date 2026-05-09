-- name: CreateAuditLog :exec
INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
