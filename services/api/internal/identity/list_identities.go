package identity

import (
	"context"
	db "my-idm/db/sqlc"
)

type ListIdentitiesHandler struct {
	q *db.Queries
}

func NewListIdentitiesHandler(q *db.Queries) *ListIdentitiesHandler {
	return &ListIdentitiesHandler{q: q}
}

func (h *ListIdentitiesHandler) Handle(ctx context.Context) ([]db.Identity, error) {
	return h.q.ListIdentities(ctx)
}
