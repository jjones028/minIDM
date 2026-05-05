-- +goose Up
ALTER TABLE roles ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE roles SET is_builtin = TRUE WHERE name IN ('admin', 'viewer');

-- +goose Down
ALTER TABLE roles DROP COLUMN is_builtin;
