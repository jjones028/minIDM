package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	db "minIDM/db/sqlc"
	"minIDM/internal/identity"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountNotActive   = errors.New("account not active")
)

const sessionDuration = 24 * time.Hour

type LoginResult struct {
	Token      string
	IdentityID pgtype.UUID
	ExpiresAt  time.Time
}

type LogoutResult struct {
	IdentityID pgtype.UUID
}

type Service struct {
	q *db.Queries
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	ident, err := s.q.GetIdentityByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !ident.IsEnabled {
		return nil, ErrAccountNotActive
	}

	ok, err := identity.VerifyPassword(password, ident.PwHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	expiresAt := time.Now().Add(sessionDuration)
	_, err = s.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash:  hashSessionToken(token),
		IdentityID: ident.ID,
		ExpiresAt:  pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return &LoginResult{Token: token, IdentityID: ident.ID, ExpiresAt: expiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, token string) (*LogoutResult, error) {
	hash := hashSessionToken(token)
	sess, err := s.q.GetSessionByToken(ctx, hash)
	if err != nil {
		_ = s.q.DeleteSession(ctx, hash)
		return nil, nil
	}
	return &LogoutResult{IdentityID: sess.IdentityID}, s.q.DeleteSession(ctx, hash)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashSessionToken returns the hex SHA-256 of a session token for safe DB storage.
// The plaintext token stays in the cookie; only the hash is persisted.
func hashSessionToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h[:])
}
