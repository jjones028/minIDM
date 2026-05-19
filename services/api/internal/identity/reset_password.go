package identity

import (
	"context"
	"errors"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrPasswordTooShort = errors.New("password must be at least 8 characters")

type ResetPasswordCommand struct {
	IdentityID  pgtype.UUID
	NewPassword string
}

type ResetPasswordHandler struct{ q *db.Queries }

func NewResetPasswordHandler(q *db.Queries) *ResetPasswordHandler {
	return &ResetPasswordHandler{q: q}
}

func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd ResetPasswordCommand) error {
	if len(cmd.NewPassword) < 8 {
		return ErrPasswordTooShort
	}
	hash, err := HashPassword(cmd.NewPassword)
	if err != nil {
		return err
	}
	if err := h.q.UpdateIdentityPassword(ctx, db.UpdateIdentityPasswordParams{
		ID:     cmd.IdentityID,
		PwHash: hash,
	}); err != nil {
		return err
	}
	// Invalidate all active sessions — the identity must re-authenticate.
	return h.q.DeleteSessionsByIdentityID(ctx, cmd.IdentityID)
}
