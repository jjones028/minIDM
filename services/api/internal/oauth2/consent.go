package oauth2

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ConsentHandler handles POST /api/oauth2/consent.
// The React consent page POSTs the validated authorize params here after the
// user clicks Approve. We re-validate everything and issue the authorization code.
type ConsentHandler struct {
	q   *db.Queries
	key *rsa.PrivateKey
}

func NewConsentHandler(q *db.Queries, key *rsa.PrivateKey) *ConsentHandler {
	return &ConsentHandler{q: q, key: key}
}

func (h *ConsentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID            string `json:"client_id"`
		RedirectURI         string `json:"redirect_uri"`
		Scope               string `json:"scope"`
		State               string `json:"state"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// Re-validate client and redirect URI (cannot redirect on these errors).
	if body.ClientID == "" || body.RedirectURI == "" || body.CodeChallenge == "" {
		http.Error(w, "invalid_request: missing required params", http.StatusBadRequest)
		return
	}

	client, err := h.q.GetOAuth2ClientByClientID(r.Context(), body.ClientID)
	if err != nil || !client.IsEnabled {
		http.Error(w, "invalid_client", http.StatusBadRequest)
		return
	}
	if !slices.Contains(client.RedirectUris, body.RedirectURI) {
		http.Error(w, "invalid_request: redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE method.
	method := body.CodeChallengeMethod
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		http.Error(w, "invalid_request: only S256 supported", http.StatusBadRequest)
		return
	}

	// Validate scopes.
	scopes := strings.Fields(body.Scope)
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}
	for _, s := range scopes {
		if !slices.Contains(client.Scopes, s) {
			http.Error(w, fmt.Sprintf("invalid_scope: %q not allowed for client", s), http.StatusBadRequest)
			return
		}
	}

	// Re-validate session cookie — user must still be logged in.
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	session, err := h.q.GetSessionByToken(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Issue authorization code.
	code, err := GenerateAuthCode()
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(5 * time.Minute)
	if _, err := h.q.CreateAuthorizationCode(r.Context(), db.CreateAuthorizationCodeParams{
		Code:                code,
		ClientID:            body.ClientID,
		IdentityID:          session.IdentityID,
		RedirectUri:         body.RedirectURI,
		Scopes:              scopes,
		CodeChallenge:       body.CodeChallenge,
		CodeChallengeMethod: method,
		ExpiresAt:           pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	// Return the redirect URL for the SPA to follow.
	dest, _ := url.Parse(body.RedirectURI)
	dq := dest.Query()
	dq.Set("code", code)
	if body.State != "" {
		dq.Set("state", body.State)
	}
	dest.RawQuery = dq.Encode()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"redirect_url": dest.String()})
}
