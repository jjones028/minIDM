package identity

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"

	"github.com/google/uuid"
)

type CreateIdentityCommand struct {
	Email    string
	Password string
}

type CreateIdentityHandler struct {
	q *db.Queries
}

func NewCreateIdentityHandler(q *db.Queries) *CreateIdentityHandler {
	return &CreateIdentityHandler{q: q}
}

// Handle creates an identity that is immediately enabled. Intended for admin use only.
func (h *CreateIdentityHandler) Handle(ctx context.Context, cmd CreateIdentityCommand) (*db.Identity, error) {
	if len(cmd.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	hashed, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}
	user, err := h.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     cmd.Email,
		PwHash:    string(hashed),
		IsEnabled: true,
	})
	if err != nil {
		return nil, err
	}
	role, err := h.q.GetRoleByName(ctx, "viewer")
	if err != nil {
		return nil, fmt.Errorf("fetching viewer role: %w", err)
	}
	if err := h.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: user.ID,
		RoleID:     role.ID,
	}); err != nil {
		return nil, fmt.Errorf("assigning viewer role: %w", err)
	}
	return &user, nil
}
