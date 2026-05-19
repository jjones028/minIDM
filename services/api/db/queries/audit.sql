-- name: CreateAuditLog :exec
INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5);

-- name: ListAuditLogsFiltered :many
SELECT
    al.id,
    al.actor_id,
    al.action,
    al.resource_type,
    al.resource_id,
    al.details,
    al.created_at,
    i.email AS actor_email
FROM audit_logs al
LEFT JOIN identities i ON i.id = al.actor_id
WHERE
    ($1::text = '' OR al.resource_type = $1)
    AND ($2::text = '' OR al.action LIKE $2)
    AND ($3::uuid IS NULL OR al.actor_id = $3)
    AND ($4::timestamptz IS NULL OR al.created_at >= $4)
    AND ($5::timestamptz IS NULL OR al.created_at <= $5)
ORDER BY al.created_at DESC
LIMIT $6 OFFSET $7;

-- name: CountAuditLogsFiltered :one
SELECT COUNT(*) FROM audit_logs al
WHERE
    ($1::text = '' OR al.resource_type = $1)
    AND ($2::text = '' OR al.action LIKE $2)
    AND ($3::uuid IS NULL OR al.actor_id = $3)
    AND ($4::timestamptz IS NULL OR al.created_at >= $4)
    AND ($5::timestamptz IS NULL OR al.created_at <= $5);

-- name: ListDistinctAuditResourceTypes :many
SELECT DISTINCT resource_type FROM audit_logs ORDER BY resource_type;
