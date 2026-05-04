-- name: ListRoles :many
SELECT * FROM roles ORDER BY name;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = $1 LIMIT 1;

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
