-- +goose Up
INSERT INTO resources (name, description)
VALUES ('oauth2_client', 'OAuth2 Client Applications');

-- Grant admin full permissions on oauth2_client
INSERT INTO permissions (role_id, resource_id, action_id)
SELECT r.id, res.id, a.id
FROM roles r, resources res, actions a
WHERE r.name = 'admin' AND res.name = 'oauth2_client';

-- +goose Down
DELETE FROM permissions
WHERE resource_id = (SELECT id FROM resources WHERE name = 'oauth2_client');
DELETE FROM resources WHERE name = 'oauth2_client';
