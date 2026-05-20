-- name: CreateIdentity :one
INSERT INTO identities (subject_id, email, pw_hash, is_enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetIdentityByEmail :one
SELECT * FROM identities
WHERE email = $1 LIMIT 1;

-- name: GetIdentityBySub :one
SELECT * FROM identities
WHERE subject_id = $1 LIMIT 1;

-- name: ListIdentities :many
SELECT * FROM identities
ORDER BY created_at DESC;

-- name: CountIdentities :one
SELECT COUNT(*) FROM identities;

-- name: GetIdentityByID :one
SELECT * FROM identities WHERE id = $1 LIMIT 1;

-- name: UpdateIdentityPassword :exec
UPDATE identities SET pw_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateIdentityEnabled :one
UPDATE identities SET is_enabled = $2, updated_at = NOW() WHERE id = $1
RETURNING id, subject_id, email, is_enabled, created_at, updated_at;