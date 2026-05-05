package rbac

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// Queries

type ListIdentityRolesHandler struct {
	q *db.Queries
}

func NewListIdentityRolesHandler(q *db.Queries) *ListIdentityRolesHandler {
	return &ListIdentityRolesHandler{q: q}
}

func (h *ListIdentityRolesHandler) Handle(ctx context.Context, identityID pgtype.UUID) ([]db.Role, error) {
	return h.q.ListRolesForIdentity(ctx, identityID)
}

// Commands

type AssignRoleCommand struct {
	IdentityID pgtype.UUID
	RoleID     pgtype.UUID
}

type AssignRoleHandler struct {
	q *db.Queries
}

func NewAssignRoleHandler(q *db.Queries) *AssignRoleHandler {
	return &AssignRoleHandler{q: q}
}

func (h *AssignRoleHandler) Handle(ctx context.Context, cmd AssignRoleCommand) error {
	return h.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: cmd.IdentityID,
		RoleID:     cmd.RoleID,
	})
}

type RemoveRoleCommand struct {
	IdentityID pgtype.UUID
	RoleID     pgtype.UUID
}

type RemoveRoleHandler struct {
	q *db.Queries
}

func NewRemoveRoleHandler(q *db.Queries) *RemoveRoleHandler {
	return &RemoveRoleHandler{q: q}
}

func (h *RemoveRoleHandler) Handle(ctx context.Context, cmd RemoveRoleCommand) error {
	return h.q.RemoveRoleFromIdentity(ctx, db.RemoveRoleFromIdentityParams{
		IdentityID: cmd.IdentityID,
		RoleID:     cmd.RoleID,
	})
}
