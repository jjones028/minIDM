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

// RevokeTokenHandler admin-revokes an OAuth2 token by its DB UUID.
type RevokeTokenHandler struct {
	q *db.Queries
}

func NewRevokeTokenHandler(q *db.Queries) *RevokeTokenHandler {
	return &RevokeTokenHandler{q: q}
}

func (h *RevokeTokenHandler) Handle(ctx context.Context, id pgtype.UUID) error {
	return h.q.RevokeOAuth2Token(ctx, id)
}

// --- HTTP handlers ---

func (a *API) ListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := a.listTokens.Handle(r.Context())
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
	if err := a.revokeToken.Handle(r.Context(), id); err != nil {
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
