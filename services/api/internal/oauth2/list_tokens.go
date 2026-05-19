package oauth2

import (
	"context"

	db "minIDM/db/sqlc"
)

// ListTokensHandler queries active (non-revoked, non-expired) OAuth2 tokens
// with the associated identity email for admin display.
type ListTokensHandler struct {
	q *db.Queries
}

func NewListTokensHandler(q *db.Queries) *ListTokensHandler {
	return &ListTokensHandler{q: q}
}

func (h *ListTokensHandler) Handle(ctx context.Context) ([]db.ListActiveOAuth2TokensRow, error) {
	return h.q.ListActiveOAuth2Tokens(ctx)
}
