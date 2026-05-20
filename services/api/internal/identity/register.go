package identity

import (
	"context"
	"errors"
	"fmt"
	db "minIDM/db/sqlc"

	"github.com/google/uuid"
)

var ErrRegistrationDisabled = errors.New("registration is disabled")

type AddRegistrationCommand struct {
	Email    string
	Password string
}

type AddRegistrationResult struct {
	Identity *db.Identity
	Pending  bool // true when the account awaits admin approval
}

type AddRegistrationHandler struct {
	q                   *db.Queries
	registrationEnabled bool
}

func NewAddRegistrationHandler(q *db.Queries, registrationEnabled bool) *AddRegistrationHandler {
	return &AddRegistrationHandler{q: q, registrationEnabled: registrationEnabled}
}

func (h *AddRegistrationHandler) Handle(ctx context.Context, cmd AddRegistrationCommand) (*AddRegistrationResult, error) {
	hashed, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	count, err := h.q.CountIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting identities: %w", err)
	}

	isFirst := count == 0
	if !isFirst && !h.registrationEnabled {
		return nil, ErrRegistrationDisabled
	}

	// First user bootstraps the system and is auto-approved as admin.
	// All subsequent self-registrations are pending admin approval.
	enabled := isFirst

	user, err := h.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     cmd.Email,
		PwHash:    string(hashed),
		IsEnabled: enabled,
	})
	if err != nil {
		return nil, err
	}

	roleName := "viewer"
	if isFirst {
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

	return &AddRegistrationResult{Identity: &user, Pending: !enabled}, nil
}
