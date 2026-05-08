package identity

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type ListIdentitySessionsHandler struct {
	q *db.Queries
}

func NewListIdentitySessionsHandler(q *db.Queries) *ListIdentitySessionsHandler {
	return &ListIdentitySessionsHandler{q: q}
}

func (h *ListIdentitySessionsHandler) Handle(ctx context.Context, id pgtype.UUID) ([]db.Session, error) {
	return h.q.ListActiveSessionsByIdentityID(ctx, id)
}
