-- +goose Up
CREATE TABLE oauth2_tokens (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id          TEXT NOT NULL REFERENCES oauth2_clients(client_id) ON DELETE CASCADE,
    identity_id        UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    jti                TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT,
    scopes             TEXT[]  NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked            BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE oauth2_tokens;
