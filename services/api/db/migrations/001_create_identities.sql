-- +goose Up
CREATE TABLE identities (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL UNIQUE,
    pw_hash    TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE identities;