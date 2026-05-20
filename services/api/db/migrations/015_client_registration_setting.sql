-- +goose Up
ALTER TABLE oauth2_clients ADD COLUMN allow_registration BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE oauth2_clients DROP COLUMN allow_registration;
