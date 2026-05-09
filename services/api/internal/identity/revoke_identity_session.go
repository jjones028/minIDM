package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

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

// sessionHandle returns the first 8 hex chars of SHA-256(token) — safe to expose as an opaque ID.
func sessionHandle(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:8]
}

func (h *RevokeIdentitySessionHandler) Handle(ctx context.Context, cmd RevokeIdentitySessionCommand) error {
	sessions, err := h.q.ListActiveSessionsByIdentityID(ctx, cmd.IdentityID)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if sessionHandle(s.Token) == cmd.Handle {
			return h.q.DeleteSession(ctx, s.Token)
		}
	}
	return fmt.Errorf("session not found")
}
