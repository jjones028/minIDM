-- name: CreateIdentity :one
INSERT INTO identities (subject_id, email, pw_hash)
VALUES ($1, $2, $3)
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