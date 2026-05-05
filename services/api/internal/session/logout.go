package session

import (
	"context"
	db "minIDM/db/sqlc"
)

type LogoutCommand struct {
	Token string
}

type LogoutHandler struct {
	q *db.Queries
}

func NewLogoutHandler(q *db.Queries) *LogoutHandler {
	return &LogoutHandler{q: q}
}

func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	return h.q.DeleteSession(ctx, cmd.Token)
}
