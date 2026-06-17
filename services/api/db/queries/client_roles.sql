-- ---- Client Roles ----

-- name: CreateClientRole :one
INSERT INTO oauth2_client_roles (client_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetClientRole :one
SELECT * FROM oauth2_client_roles WHERE id = $1;

-- name: ListClientRoles :many
SELECT * FROM oauth2_client_roles WHERE client_id = $1 ORDER BY name;

-- name: UpdateClientRole :one
UPDATE oauth2_client_roles SET name = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: DeleteClientRole :exec
DELETE FROM oauth2_client_roles WHERE id = $1;

-- ---- Identity ↔ Role (direct assignment) ----

-- name: AssignIdentityToClientRole :exec
INSERT INTO identity_client_roles (identity_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveIdentityFromClientRole :exec
DELETE FROM identity_client_roles WHERE identity_id = $1 AND role_id = $2;

-- name: ListIdentitiesWithClientRole :many
SELECT i.id, i.email
FROM identities i
JOIN identity_client_roles icr ON icr.identity_id = i.id
WHERE icr.role_id = $1
ORDER BY i.email;

-- name: ListDirectClientRolesForIdentity :many
SELECT r.id, r.name, r.description, r.client_id, c.name AS client_name, c.client_id AS app_client_id
FROM oauth2_client_roles r
JOIN identity_client_roles icr ON icr.role_id = r.id
JOIN oauth2_clients c ON c.id = r.client_id
WHERE icr.identity_id = $1
ORDER BY c.name, r.name;

-- ---- Client Groups ----

-- name: CreateClientGroup :one
INSERT INTO oauth2_client_groups (client_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetClientGroup :one
SELECT * FROM oauth2_client_groups WHERE id = $1;

-- name: ListClientGroups :many
SELECT * FROM oauth2_client_groups WHERE client_id = $1 ORDER BY name;

-- name: UpdateClientGroup :one
UPDATE oauth2_client_groups SET name = $2, description = $3 WHERE id = $1 RETURNING *;

-- name: DeleteClientGroup :exec
DELETE FROM oauth2_client_groups WHERE id = $1;

-- ---- Identity ↔ Group ----

-- name: AddIdentityToClientGroup :exec
INSERT INTO identity_client_groups (identity_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveIdentityFromClientGroup :exec
DELETE FROM identity_client_groups WHERE identity_id = $1 AND group_id = $2;

-- name: ListIdentitiesInClientGroup :many
SELECT i.id, i.email
FROM identities i
JOIN identity_client_groups icg ON icg.identity_id = i.id
WHERE icg.group_id = $1
ORDER BY i.email;

-- name: ListClientGroupsForIdentity :many
SELECT g.id, g.name, g.description, g.client_id, c.name AS client_name, c.client_id AS app_client_id
FROM oauth2_client_groups g
JOIN identity_client_groups icg ON icg.group_id = g.id
JOIN oauth2_clients c ON c.id = g.client_id
WHERE icg.identity_id = $1
ORDER BY c.name, g.name;

-- ---- Group ↔ Role ----

-- name: AssignRoleToClientGroup :exec
INSERT INTO client_group_roles (group_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromClientGroup :exec
DELETE FROM client_group_roles WHERE group_id = $1 AND role_id = $2;

-- name: ListRolesForClientGroup :many
SELECT r.*
FROM oauth2_client_roles r
JOIN client_group_roles cgr ON cgr.role_id = r.id
WHERE cgr.group_id = $1
ORDER BY r.name;

-- name: ListGroupsForClientRole :many
SELECT g.id, g.name
FROM oauth2_client_groups g
JOIN client_group_roles cgr ON cgr.group_id = g.id
WHERE cgr.role_id = $1
ORDER BY g.name;

-- ---- Token issuance: effective roles ----

-- name: GetEffectiveClientRolesForIdentity :many
SELECT DISTINCT r.name
FROM oauth2_client_roles r
WHERE r.client_id = $1::uuid
  AND r.id IN (
    SELECT role_id FROM identity_client_roles WHERE identity_id = $2::uuid
    UNION
    SELECT cgr.role_id FROM client_group_roles cgr
    JOIN identity_client_groups icg ON icg.group_id = cgr.group_id
    WHERE icg.identity_id = $2::uuid
  )
ORDER BY r.name;
