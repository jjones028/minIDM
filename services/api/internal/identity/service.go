package identity

import (
	"context"
	"errors"
	"fmt"

	db "minIDM/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrSessionNotFound      = errors.New("session not found")
)

type AddRegistrationResult struct {
	Identity *db.Identity
	Pending  bool
}

type Service struct {
	q                   *db.Queries
	registrationEnabled bool
}

func NewService(q *db.Queries, registrationEnabled bool) *Service {
	return &Service{q: q, registrationEnabled: registrationEnabled}
}

// Register handles self-registration. The first identity is auto-enabled as admin;
// subsequent registrations are pending unless registrationEnabled is true.
func (s *Service) Register(ctx context.Context, email, password string) (*AddRegistrationResult, error) {
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	count, err := s.q.CountIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting identities: %w", err)
	}

	isFirst := count == 0
	if !isFirst && !s.registrationEnabled {
		return nil, ErrRegistrationDisabled
	}

	enabled := isFirst

	user, err := s.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     email,
		PwHash:    string(hashed),
		IsEnabled: enabled,
	})
	if err != nil {
		return nil, err
	}

	roleName := "viewer"
	if isFirst {
		roleName = "admin"
	}
	role, err := s.q.GetRoleByName(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("fetching %s role: %w", roleName, err)
	}
	if err := s.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: user.ID,
		RoleID:     role.ID,
	}); err != nil {
		return nil, fmt.Errorf("assigning %s role: %w", roleName, err)
	}

	return &AddRegistrationResult{Identity: &user, Pending: !enabled}, nil
}

// Create creates an immediately-enabled identity. Intended for admin use only.
func (s *Service) Create(ctx context.Context, email, password string) (*db.Identity, error) {
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}
	user, err := s.q.CreateIdentity(ctx, db.CreateIdentityParams{
		SubjectID: uuid.New().String(),
		Email:     email,
		PwHash:    string(hashed),
		IsEnabled: true,
	})
	if err != nil {
		return nil, err
	}
	role, err := s.q.GetRoleByName(ctx, "viewer")
	if err != nil {
		return nil, fmt.Errorf("fetching viewer role: %w", err)
	}
	if err := s.q.AssignRoleToIdentity(ctx, db.AssignRoleToIdentityParams{
		IdentityID: user.ID,
		RoleID:     role.ID,
	}); err != nil {
		return nil, fmt.Errorf("assigning viewer role: %w", err)
	}
	return &user, nil
}

func (s *Service) List(ctx context.Context) ([]db.Identity, error) {
	return s.q.ListIdentities(ctx)
}

func (s *Service) Get(ctx context.Context, id pgtype.UUID) (db.Identity, error) {
	return s.q.GetIdentityByID(ctx, id)
}

func (s *Service) ListSessions(ctx context.Context, id pgtype.UUID) ([]db.Session, error) {
	return s.q.ListActiveSessionsByIdentityID(ctx, id)
}

// sessionHandle returns the first 8 hex chars of the stored token_hash as an opaque handle.
func sessionHandle(tokenHash string) string {
	return tokenHash[:8]
}

func (s *Service) RevokeSession(ctx context.Context, identityID pgtype.UUID, handle string) error {
	sessions, err := s.q.ListActiveSessionsByIdentityID(ctx, identityID)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sessionHandle(sess.TokenHash) == handle {
			return s.q.DeleteSession(ctx, sess.TokenHash)
		}
	}
	return ErrSessionNotFound
}

func (s *Service) ResetPassword(ctx context.Context, id pgtype.UUID, password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.q.UpdateIdentityPassword(ctx, db.UpdateIdentityPasswordParams{
		ID:     id,
		PwHash: hash,
	}); err != nil {
		return err
	}
	// Invalidate all active sessions — the identity must re-authenticate.
	return s.q.DeleteSessionsByIdentityID(ctx, id)
}

func (s *Service) SetEnabled(ctx context.Context, id pgtype.UUID, enabled bool) (db.UpdateIdentityEnabledRow, error) {
	return s.q.UpdateIdentityEnabled(ctx, db.UpdateIdentityEnabledParams{
		ID:        id,
		IsEnabled: enabled,
	})
}

func (s *Service) ListClientRoles(ctx context.Context, id pgtype.UUID) ([]db.ListDirectClientRolesForIdentityRow, error) {
	return s.q.ListDirectClientRolesForIdentity(ctx, id)
}

func (s *Service) RemoveClientRole(ctx context.Context, identityID, roleID pgtype.UUID) error {
	return s.q.RemoveIdentityFromClientRole(ctx, db.RemoveIdentityFromClientRoleParams{
		IdentityID: identityID,
		RoleID:     roleID,
	})
}

func (s *Service) ListClientGroups(ctx context.Context, id pgtype.UUID) ([]db.ListClientGroupsForIdentityRow, error) {
	return s.q.ListClientGroupsForIdentity(ctx, id)
}

func (s *Service) RemoveClientGroup(ctx context.Context, identityID, groupID pgtype.UUID) error {
	return s.q.RemoveIdentityFromClientGroup(ctx, db.RemoveIdentityFromClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	})
}
