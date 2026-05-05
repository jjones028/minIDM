package rbac

import (
	"context"
	"errors"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrBuiltinRoleProtected = errors.New("builtin_role_protected")
var ErrNotFound = errors.New("not_found")

// Queries

type ListRolesHandler struct {
	q *db.Queries
}

func NewListRolesHandler(q *db.Queries) *ListRolesHandler {
	return &ListRolesHandler{q: q}
}

func (h *ListRolesHandler) Handle(ctx context.Context) ([]db.Role, error) {
	return h.q.ListRoles(ctx)
}

// Commands

type CreateRoleCommand struct {
	Name        string
	Description string
}

type CreateRoleHandler struct {
	q *db.Queries
}

func NewCreateRoleHandler(q *db.Queries) *CreateRoleHandler {
	return &CreateRoleHandler{q: q}
}

func (h *CreateRoleHandler) Handle(ctx context.Context, cmd CreateRoleCommand) (db.Role, error) {
	return h.q.CreateRole(ctx, db.CreateRoleParams{
		Name:        cmd.Name,
		Description: pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
	})
}

type UpdateRoleCommand struct {
	ID          pgtype.UUID
	Name        string
	Description string
}

type UpdateRoleHandler struct {
	q *db.Queries
}

func NewUpdateRoleHandler(q *db.Queries) *UpdateRoleHandler {
	return &UpdateRoleHandler{q: q}
}

func (h *UpdateRoleHandler) Handle(ctx context.Context, cmd UpdateRoleCommand) (db.Role, error) {
	return h.q.UpdateRole(ctx, db.UpdateRoleParams{
		ID:          cmd.ID,
		Name:        cmd.Name,
		Description: pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
	})
}

type DeleteRoleCommand struct {
	ID pgtype.UUID
}

type DeleteRoleHandler struct {
	q *db.Queries
}

func NewDeleteRoleHandler(q *db.Queries) *DeleteRoleHandler {
	return &DeleteRoleHandler{q: q}
}

func (h *DeleteRoleHandler) Handle(ctx context.Context, cmd DeleteRoleCommand) error {
	role, err := h.q.GetRoleByID(ctx, cmd.ID)
	if err != nil {
		return ErrNotFound
	}
	if role.IsBuiltin {
		return ErrBuiltinRoleProtected
	}
	return h.q.DeleteRole(ctx, cmd.ID)
}
