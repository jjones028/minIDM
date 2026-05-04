-- +goose Up
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE actions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    action_id   UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (role_id, resource_id, action_id)
);

CREATE TABLE identity_roles (
    identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (identity_id, role_id)
);

-- Seed bootstrap data
INSERT INTO resources (name, description) VALUES
    ('identity', 'User identity records'),
    ('role',     'RBAC roles'),
    ('session',  'User sessions');

INSERT INTO actions (name, description) VALUES
    ('read',   'Read or list a resource'),
    ('write',  'Create or update a resource'),
    ('delete', 'Delete a resource');

INSERT INTO roles (name, description) VALUES
    ('admin',  'Full access to all resources'),
    ('viewer', 'Read-only access to all resources');

-- admin: all permissions
INSERT INTO permissions (role_id, resource_id, action_id)
SELECT r.id, res.id, a.id
FROM   roles r, resources res, actions a
WHERE  r.name = 'admin';

-- viewer: read-only permissions
INSERT INTO permissions (role_id, resource_id, action_id)
SELECT r.id, res.id, a.id
FROM   roles r, resources res, actions a
WHERE  r.name = 'viewer' AND a.name = 'read';

-- +goose Down
DROP TABLE identity_roles;
DROP TABLE permissions;
DROP TABLE actions;
DROP TABLE resources;
DROP TABLE roles;
