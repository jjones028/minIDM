-- +goose Up
CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,
    identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_identity_id ON sessions(identity_id);

-- +goose Down
DROP TABLE sessions;
