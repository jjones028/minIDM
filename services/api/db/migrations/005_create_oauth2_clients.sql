-- +goose Up
CREATE TABLE oauth2_clients (
    id                 UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id          TEXT    NOT NULL UNIQUE,
    client_secret_hash TEXT    NOT NULL,
    name               TEXT    NOT NULL,
    description        TEXT,
    redirect_uris      TEXT[]  NOT NULL DEFAULT '{}',
    scopes             TEXT[]  NOT NULL DEFAULT '{openid,profile,email}',
    is_enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE oauth2_clients;
