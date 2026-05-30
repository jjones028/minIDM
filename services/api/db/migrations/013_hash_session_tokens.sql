-- +goose Up
-- Store SHA-256(token) instead of the raw token.
-- All existing sessions are invalidated — logged-in users must re-authenticate.
ALTER TABLE sessions RENAME COLUMN token TO token_hash;

-- +goose Down
ALTER TABLE sessions RENAME COLUMN token_hash TO token;
