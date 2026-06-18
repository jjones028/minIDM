package identity

import (
	"context"
	"errors"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrSessionNotFound = errors.New("session not found")

type RevokeIdentitySessionHandler struct {
	q *db.Queries
}

func NewRevokeIdentitySessionHandler(q *db.Queries) *RevokeIdentitySessionHandler {
	return &RevokeIdentitySessionHandler{q: q}
}

type RevokeIdentitySessionCommand struct {
	IdentityID pgtype.UUID
	Handle     string
}

// sessionHandle returns the first 8 hex chars of the stored token_hash.
// The DB stores hex(SHA-256(token)), so this is prefix-safe and collision-resistant
// for any realistic number of concurrent sessions per identity.
func sessionHandle(tokenHash string) string {
	return tokenHash[:8]
}

func (h *RevokeIdentitySessionHandler) Handle(ctx context.Context, cmd RevokeIdentitySessionCommand) error {
	sessions, err := h.q.ListActiveSessionsByIdentityID(ctx, cmd.IdentityID)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if sessionHandle(s.TokenHash) == cmd.Handle {
			return h.q.DeleteSession(ctx, s.TokenHash)
		}
	}
	return ErrSessionNotFound
}
