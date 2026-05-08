-- name: CreateSession :one
INSERT INTO sessions (token, identity_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();

-- name: ListActiveSessionsByIdentityID :many
SELECT * FROM sessions
WHERE identity_id = $1 AND expires_at > NOW()
ORDER BY created_at DESC;
