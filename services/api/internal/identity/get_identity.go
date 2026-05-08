package identity

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type GetIdentityHandler struct {
	q *db.Queries
}

func NewGetIdentityHandler(q *db.Queries) *GetIdentityHandler {
	return &GetIdentityHandler{q: q}
}

func (h *GetIdentityHandler) Handle(ctx context.Context, id pgtype.UUID) (db.Identity, error) {
	return h.q.GetIdentityByID(ctx, id)
}
