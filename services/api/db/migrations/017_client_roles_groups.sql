-- +goose Up

CREATE TABLE oauth2_client_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, name)
);

CREATE TABLE identity_client_roles (
    identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES oauth2_client_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (identity_id, role_id)
);

CREATE TABLE oauth2_client_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, name)
);

CREATE TABLE identity_client_groups (
    identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES oauth2_client_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (identity_id, group_id)
);

CREATE TABLE client_group_roles (
    group_id UUID NOT NULL REFERENCES oauth2_client_groups(id) ON DELETE CASCADE,
    role_id  UUID NOT NULL REFERENCES oauth2_client_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, role_id)
);

-- +goose Down
DROP TABLE IF EXISTS client_group_roles;
DROP TABLE IF EXISTS identity_client_groups;
DROP TABLE IF EXISTS oauth2_client_groups;
DROP TABLE IF EXISTS identity_client_roles;
DROP TABLE IF EXISTS oauth2_client_roles;
