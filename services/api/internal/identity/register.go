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
	// 1. Secure Hashing
	hashed, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	// 2. Data: Call the generated SQL function
	user, err := h.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     cmd.Email,
		PwHash:    string(hashed),
	})

	return &user, err
}
