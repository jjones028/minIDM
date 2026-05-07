-- +goose Up
CREATE TABLE oauth2_authorization_codes (
    code                  TEXT    PRIMARY KEY,
    client_id             TEXT    NOT NULL REFERENCES oauth2_clients(client_id) ON DELETE CASCADE,
    identity_id           UUID    NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    redirect_uri          TEXT    NOT NULL,
    scopes                TEXT[]  NOT NULL,
    code_challenge        TEXT    NOT NULL,
    code_challenge_method TEXT    NOT NULL DEFAULT 'S256',
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used                  BOOLEAN NOT NULL DEFAULT FALSE
);

-- +goose Down
DROP TABLE oauth2_authorization_codes;
