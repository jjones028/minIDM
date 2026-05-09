package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	db "minIDM/db/sqlc"
	"minIDM/internal/identity"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

const sessionDuration = 24 * time.Hour

type LoginCommand struct {
	Email    string
	Password string
}

type LoginResult struct {
	Token      string
	IdentityID pgtype.UUID
	ExpiresAt  time.Time
}

type LoginHandler struct {
	q *db.Queries
}

func NewLoginHandler(q *db.Queries) *LoginHandler {
	return &LoginHandler{q: q}
}

func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	ident, err := h.q.GetIdentityByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := identity.VerifyPassword(cmd.Password, ident.PwHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	expiresAt := time.Now().Add(sessionDuration)
	_, err = h.q.CreateSession(ctx, db.CreateSessionParams{
		Token:      token,
		IdentityID: ident.ID,
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return &LoginResult{Token: token, IdentityID: ident.ID, ExpiresAt: expiresAt}, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
