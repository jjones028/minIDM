package oauth2

import (
	"context"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/httputil"
	"minIDM/internal/rbac"

	"github.com/jackc/pgx/v5/pgtype"
)

// TokenService manages the admin token lifecycle (list, revoke).
// Token issuance is handled by TokenHandler (OAuth2 protocol).
type TokenService struct {
	q *db.Queries
}

func NewTokenService(q *db.Queries) *TokenService {
	return &TokenService{q: q}
}

func (s *TokenService) List(ctx context.Context) ([]db.ListActiveOAuth2TokensRow, error) {
	return s.q.ListActiveOAuth2Tokens(ctx)
}

func (s *TokenService) Revoke(ctx context.Context, id pgtype.UUID) error {
	return s.q.RevokeOAuth2Token(ctx, id)
}

// --- HTTP handlers ---

func (a *API) ListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := a.tokens.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tokens == nil {
		tokens = []db.ListActiveOAuth2TokensRow{}
	}
	httputil.WriteJSON(w, tokens)
}

func (a *API) RevokeToken(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	if err := a.tokens.Revoke(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_token.revoke", "oauth2_token", audit.UUIDStr(id), nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) InspectToken(w http.ResponseWriter, r *http.Request) {
	a.inspectToken.ServeHTTP(w, r)
}
