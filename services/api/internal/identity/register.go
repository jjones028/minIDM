package identity

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"

	"github.com/google/uuid"
)

type AddRegistrationCommand struct {
	Email    string
	Password string
}

type AddRegistrationHandler struct {
	q *db.Queries
}

func NewAddRegistrationHandler(q *db.Queries) *AddRegistrationHandler {
	return &AddRegistrationHandler{q: q}
}

func (h *AddRegistrationHandler) Handle(ctx context.Context, cmd AddRegistrationCommand) (*db.Identity, error) {
	hashed, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	user, err := h.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     cmd.Email,
		PwHash:    string(hashed),
	})
	if err != nil {
		return nil, err
	}

	// First identity in the system becomes admin; all subsequent ones get viewer.
	count, err := h.q.CountIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting identities: %w", err)
	}
	roleName := "viewer"
	if count == 1 {
		roleName = "admin"
	}

	role, err := h.q.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("fetching %s role: %w", roleName, err)
	}
	if err := h.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: user.ID,
		RoleID:     role.ID,
	}); err != nil {
		return nil, fmt.Errorf("assigning %s role: %w", roleName, err)
	}

	return &user, nil
}
