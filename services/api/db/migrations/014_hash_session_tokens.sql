-- +goose Up
-- Clear all sessions: switching to hashed token storage.
-- Existing plaintext tokens cannot be validated after this change; all users must re-authenticate.
TRUNCATE TABLE sessions;

-- +goose Down
-- Nothing to restore; sessions were intentionally cleared.
