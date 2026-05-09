-- +goose Up
ALTER TABLE oauth2_clients ADD COLUMN auto_consent BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE oauth2_clients DROP COLUMN auto_consent;
