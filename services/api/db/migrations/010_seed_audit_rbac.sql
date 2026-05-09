-- +goose Up
INSERT INTO resources (name, description)
VALUES ('audit_log', 'Audit log entries');

-- Grant admin read permission on audit_log
INSERT INTO permissions (role_id, resource_id, action_id)
SELECT r.id, res.id, a.id
FROM roles r, resources res, actions a
WHERE r.name = 'admin' AND res.name = 'audit_log' AND a.name = 'read';

-- +goose Down
DELETE FROM permissions
WHERE resource_id = (SELECT id FROM resources WHERE name = 'audit_log');
DELETE FROM resources WHERE name = 'audit_log';
