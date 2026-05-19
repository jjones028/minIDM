package identity

import (
	"context"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type SetEnabledCommand struct {
	IdentityID pgtype.UUID
	Enabled    bool
}

type SetEnabledHandler struct{ q *db.Queries }

func NewSetEnabledHandler(q *db.Queries) *SetEnabledHandler {
	return &SetEnabledHandler{q: q}
}

func (h *SetEnabledHandler) Handle(ctx context.Context, cmd SetEnabledCommand) (db.UpdateIdentityEnabledRow, error) {
	return h.q.UpdateIdentityEnabled(ctx, db.UpdateIdentityEnabledParams{
		ID:        cmd.IdentityID,
		IsEnabled: cmd.Enabled,
	})
}
