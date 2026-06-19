package rbac

import (
	"context"
	"errors"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrBuiltinRoleProtected = errors.New("builtin_role_protected")
	ErrNotFound             = errors.New("not_found")
)

type Service struct {
	q *db.Queries
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) ListRoles(ctx context.Context) ([]db.Role, error) {
	return s.q.ListRoles(ctx)
}

func (s *Service) CreateRole(ctx context.Context, name, description string) (db.Role, error) {
	return s.q.CreateRole(ctx, db.CreateRoleParams{
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (s *Service) UpdateRole(ctx context.Context, id pgtype.UUID, name, description string) (db.Role, error) {
	return s.q.UpdateRole(ctx, db.UpdateRoleParams{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (s *Service) DeleteRole(ctx context.Context, id pgtype.UUID) error {
	role, err := s.q.GetRoleByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if role.IsBuiltin {
		return ErrBuiltinRoleProtected
	}
	return s.q.DeleteRole(ctx, id)
}

func (s *Service) ListRolePermissions(ctx context.Context, roleID pgtype.UUID) ([]db.ListPermissionsForRoleRow, error) {
	return s.q.ListPermissionsForRole(ctx, roleID)
}

func (s *Service) AddRolePermission(ctx context.Context, roleID, resourceID, actionID pgtype.UUID) (db.Permission, error) {
	role, err := s.q.GetRoleByID(ctx, roleID)
	if err != nil {
		return db.Permission{}, ErrNotFound
	}
	if role.IsBuiltin {
		return db.Permission{}, ErrBuiltinRoleProtected
	}
	return s.q.AddPermissionToRole(ctx, db.AddPermissionToRoleParams{
		RoleID:     roleID,
		ResourceID: resourceID,
		ActionID:   actionID,
	})
}

func (s *Service) RemoveRolePermission(ctx context.Context, permID, roleID pgtype.UUID) error {
	role, err := s.q.GetRoleByID(ctx, roleID)
	if err != nil {
		return ErrNotFound
	}
	if role.IsBuiltin {
		return ErrBuiltinRoleProtected
	}
	return s.q.RemovePermissionFromRole(ctx, db.RemovePermissionFromRoleParams{
		ID:     permID,
		RoleID: roleID,
	})
}

func (s *Service) ListResources(ctx context.Context) ([]db.Resource, error) {
	return s.q.ListResources(ctx)
}

func (s *Service) ListActions(ctx context.Context) ([]db.Action, error) {
	return s.q.ListActions(ctx)
}

func (s *Service) ListIdentityRoles(ctx context.Context, identityID pgtype.UUID) ([]db.Role, error) {
	return s.q.ListRolesForIdentity(ctx, identityID)
}

func (s *Service) AssignRole(ctx context.Context, identityID, roleID pgtype.UUID) error {
	return s.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: identityID,
		RoleID:     roleID,
	})
}

func (s *Service) RemoveRole(ctx context.Context, identityID, roleID pgtype.UUID) error {
	return s.q.RemoveRoleFromIdentity(ctx, db.RemoveRoleFromIdentityParams{
		IdentityID: identityID,
		RoleID:     roleID,
	})
}
