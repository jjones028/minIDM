package rbac

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// Queries

type ListRolePermissionsHandler struct {
	q *db.Queries
}

func NewListRolePermissionsHandler(q *db.Queries) *ListRolePermissionsHandler {
	return &ListRolePermissionsHandler{q: q}
}

func (h *ListRolePermissionsHandler) Handle(ctx context.Context, roleID pgtype.UUID) ([]db.ListPermissionsForRoleRow, error) {
	return h.q.ListPermissionsForRole(ctx, roleID)
}

// Commands

type AddRolePermissionCommand struct {
	RoleID     pgtype.UUID
	ResourceID pgtype.UUID
	ActionID   pgtype.UUID
}

type AddRolePermissionHandler struct {
	q *db.Queries
}

func NewAddRolePermissionHandler(q *db.Queries) *AddRolePermissionHandler {
	return &AddRolePermissionHandler{q: q}
}

func (h *AddRolePermissionHandler) Handle(ctx context.Context, cmd AddRolePermissionCommand) (db.Permission, error) {
	role, err := h.q.GetRoleByID(ctx, cmd.RoleID)
	if err != nil {
		return db.Permission{}, ErrNotFound
	}
	if role.IsBuiltin {
		return db.Permission{}, ErrBuiltinRoleProtected
	}

	return h.q.AddPermissionToRole(ctx, db.AddPermissionToRoleParams{
		RoleID:     cmd.RoleID,
		ResourceID: cmd.ResourceID,
		ActionID:   cmd.ActionID,
	})
}

type RemoveRolePermissionCommand struct {
	ID     pgtype.UUID
	RoleID pgtype.UUID
}

type RemoveRolePermissionHandler struct {
	q *db.Queries
}

func NewRemoveRolePermissionHandler(q *db.Queries) *RemoveRolePermissionHandler {
	return &RemoveRolePermissionHandler{q: q}
}

func (h *RemoveRolePermissionHandler) Handle(ctx context.Context, cmd RemoveRolePermissionCommand) error {
	role, err := h.q.GetRoleByID(ctx, cmd.RoleID)
	if err != nil {
		return ErrNotFound
	}
	if role.IsBuiltin {
		return ErrBuiltinRoleProtected
	}

	return h.q.RemovePermissionFromRole(ctx, db.RemovePermissionFromRoleParams{
		ID:     cmd.ID,
		RoleID: cmd.RoleID,
	})
}
