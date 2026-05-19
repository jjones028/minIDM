-- +goose Up
ALTER TABLE oauth2_clients ALTER COLUMN client_secret_hash DROP NOT NULL;

-- +goose Down
UPDATE oauth2_clients SET client_secret_hash = '' WHERE client_secret_hash IS NULL;
ALTER TABLE oauth2_clients ALTER COLUMN client_secret_hash SET NOT NULL;
