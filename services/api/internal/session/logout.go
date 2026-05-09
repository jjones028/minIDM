package session

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type LogoutCommand struct {
	Token string
}

type LogoutResult struct {
	IdentityID pgtype.UUID
}

type LogoutHandler struct {
	q *db.Queries
}

func NewLogoutHandler(q *db.Queries) *LogoutHandler {
	return &LogoutHandler{q: q}
}

func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) (*LogoutResult, error) {
	sess, err := h.q.GetSessionByToken(ctx, cmd.Token)
	if err != nil {
		// Session may be expired or already gone — still attempt delete.
		_ = h.q.DeleteSession(ctx, cmd.Token)
		return nil, nil
	}
	return &LogoutResult{IdentityID: sess.IdentityID}, h.q.DeleteSession(ctx, cmd.Token)
}
