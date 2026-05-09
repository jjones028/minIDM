-- +goose Up
ALTER TABLE oauth2_authorization_codes ADD COLUMN nonce TEXT;

-- +goose Down
ALTER TABLE oauth2_authorization_codes DROP COLUMN nonce;
