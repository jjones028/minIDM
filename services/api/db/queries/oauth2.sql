-- name: CreateOAuth2Client :one
INSERT INTO oauth2_clients (client_id, client_secret_hash, name, description, redirect_uris, scopes, auto_consent)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetOAuth2ClientByClientID :one
SELECT * FROM oauth2_clients WHERE client_id = $1 LIMIT 1;

-- name: GetOAuth2ClientPublicInfo :one
SELECT name, description, scopes, auto_consent FROM oauth2_clients
WHERE client_id = $1 AND is_enabled = TRUE LIMIT 1;

-- name: GetOAuth2ClientByID :one
SELECT * FROM oauth2_clients WHERE id = $1 LIMIT 1;

-- name: ListOAuth2Clients :many
SELECT * FROM oauth2_clients ORDER BY created_at DESC;

-- name: UpdateOAuth2Client :one
UPDATE oauth2_clients
SET name = $2, description = $3, redirect_uris = $4, scopes = $5, is_enabled = $6, auto_consent = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOAuth2Client :exec
DELETE FROM oauth2_clients WHERE id = $1;

-- name: CreateAuthorizationCode :one
INSERT INTO oauth2_authorization_codes
    (code, client_id, identity_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, nonce)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetAuthorizationCode :one
SELECT * FROM oauth2_authorization_codes
WHERE code = $1 AND used = FALSE AND expires_at > NOW()
LIMIT 1;

-- name: MarkAuthorizationCodeUsed :exec
UPDATE oauth2_authorization_codes SET used = TRUE WHERE code = $1;

-- name: CreateOAuth2Token :one
INSERT INTO oauth2_tokens (client_id, identity_id, jti, refresh_token_hash, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetOAuth2TokenByJTI :one
SELECT * FROM oauth2_tokens
WHERE jti = $1 AND revoked = FALSE
LIMIT 1;

-- name: GetOAuth2TokenByRefreshHash :one
SELECT * FROM oauth2_tokens
WHERE refresh_token_hash = $1 AND revoked = FALSE AND expires_at > NOW()
LIMIT 1;

-- name: RevokeOAuth2Token :exec
UPDATE oauth2_tokens SET revoked = TRUE WHERE id = $1;
