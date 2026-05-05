-- name: ListRoles :many
SELECT * FROM roles ORDER BY is_builtin DESC, name;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = $1 LIMIT 1;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1 LIMIT 1;

-- name: CreateRole :one
INSERT INTO roles (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateRole :one
UPDATE roles
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: AssignRoleToIdentity :exec
INSERT INTO identity_roles (identity_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromIdentity :exec
DELETE FROM identity_roles
WHERE identity_id = $1 AND role_id = $2;

-- name: ListRolesForIdentity :many
SELECT r.*
FROM roles r
JOIN identity_roles ir ON ir.role_id = r.id
WHERE ir.identity_id = $1
ORDER BY r.name;

-- name: IdentityHasPermission :one
SELECT EXISTS (
    SELECT 1
    FROM identity_roles ir
    JOIN permissions p   ON p.role_id     = ir.role_id
    JOIN resources   res ON res.id         = p.resource_id
    JOIN actions     a   ON a.id           = p.action_id
    WHERE ir.identity_id = $1
      AND res.name       = $2
      AND a.name         = $3
) AS has_permission;

-- name: ListPermissionsForIdentity :many
SELECT DISTINCT res.name AS resource, a.name AS action
FROM identity_roles ir
JOIN permissions p   ON p.role_id     = ir.role_id
JOIN resources   res ON res.id         = p.resource_id
JOIN actions     a   ON a.id           = p.action_id
WHERE ir.identity_id = $1
ORDER BY res.name, a.name;

-- name: ListResources :many
SELECT * FROM resources ORDER BY name;

-- name: ListActions :many
SELECT * FROM actions ORDER BY name;

-- name: ListPermissionsForRole :many
SELECT p.id, res.name AS resource, a.name AS action
FROM permissions p
JOIN resources res ON res.id = p.resource_id
JOIN actions   a   ON a.id   = p.action_id
WHERE p.role_id = $1
ORDER BY res.name, a.name;

-- name: AddPermissionToRole :one
INSERT INTO permissions (role_id, resource_id, action_id)
VALUES ($1, $2, $3)
ON CONFLICT (role_id, resource_id, action_id) DO UPDATE SET created_at = permissions.created_at
RETURNING *;

-- name: RemovePermissionFromRole :exec
DELETE FROM permissions WHERE id = $1 AND role_id = $2;
