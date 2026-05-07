package oauth2

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// AuthorizeHandler handles GET /oauth2/authorize.
// It validates the request, checks the user's session, and — on auto-approval —
// issues an authorization code and redirects to the client's redirect_uri.
type AuthorizeHandler struct {
	q   *db.Queries
	key *rsa.PrivateKey // kept for future use (e.g., request objects)
}

func NewAuthorizeHandler(q *db.Queries, key *rsa.PrivateKey) *AuthorizeHandler {
	return &AuthorizeHandler{q: q, key: key}
}

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()

	responseType := qp.Get("response_type")
	clientID := qp.Get("client_id")
	redirectURI := qp.Get("redirect_uri")
	scope := qp.Get("scope")
	state := qp.Get("state")
	codeChallenge := qp.Get("code_challenge")
	codeChallengeMethod := qp.Get("code_challenge_method")

	// --- Validate params that CANNOT redirect on error ---

	if responseType != "code" {
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}
	if clientID == "" {
		http.Error(w, "invalid_request: missing client_id", http.StatusBadRequest)
		return
	}

	client, err := h.q.GetOAuth2ClientByClientID(r.Context(), clientID)
	if err != nil {
		http.Error(w, "invalid_client", http.StatusBadRequest)
		return
	}
	if !client.IsEnabled {
		http.Error(w, "invalid_client: client is disabled", http.StatusBadRequest)
		return
	}
	if redirectURI == "" {
		http.Error(w, "invalid_request: missing redirect_uri", http.StatusBadRequest)
		return
	}
	if !slices.Contains(client.RedirectUris, redirectURI) {
		http.Error(w, "invalid_request: redirect_uri not registered for this client", http.StatusBadRequest)
		return
	}

	// --- From here, errors redirect to redirect_uri ---

	if codeChallenge == "" {
		redirectWithError(w, redirectURI, state, "invalid_request", "PKCE code_challenge required")
		return
	}
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}
	if codeChallengeMethod != "S256" {
		redirectWithError(w, redirectURI, state, "invalid_request", "only S256 code_challenge_method is supported")
		return
	}

	scopes := strings.Fields(scope)
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}
	for _, s := range scopes {
		if !slices.Contains(client.Scopes, s) {
			redirectWithError(w, redirectURI, state, "invalid_scope",
				fmt.Sprintf("scope %q is not authorised for this client", s))
			return
		}
	}

	// --- Authenticate user via session cookie ---

	cookie, err := r.Cookie("session")
	if err != nil {
		loginRedirect(w, r)
		return
	}
	session, err := h.q.GetSessionByToken(r.Context(), cookie.Value)
	if err != nil {
		loginRedirect(w, r)
		return
	}

	// --- Issue authorization code ---

	code, err := GenerateAuthCode()
	if err != nil {
		redirectWithError(w, redirectURI, state, "server_error", "failed to generate authorization code")
		return
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	if _, err := h.q.CreateAuthorizationCode(r.Context(), db.CreateAuthorizationCodeParams{
		Code:                code,
		ClientID:            clientID,
		IdentityID:          session.IdentityID,
		RedirectUri:         redirectURI,
		Scopes:              scopes,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		redirectWithError(w, redirectURI, state, "server_error", "failed to save authorization code")
		return
	}

	// --- Redirect with code ---

	dest, _ := url.Parse(redirectURI)
	dq := dest.Query()
	dq.Set("code", code)
	if state != "" {
		dq.Set("state", state)
	}
	dest.RawQuery = dq.Encode()
	http.Redirect(w, r, dest.String(), http.StatusFound)
}

// loginRedirect sends the browser to the login page, preserving the original
// authorize URL so the SPA can redirect back after successful authentication.
func loginRedirect(w http.ResponseWriter, r *http.Request) {
	next := "/oauth2/authorize?" + r.URL.RawQuery
	http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
}

// redirectWithError redirects to redirect_uri with OAuth2 error params.
func redirectWithError(w http.ResponseWriter, redirectURI, state, errCode, description string) {
	dest, _ := url.Parse(redirectURI)
	dq := dest.Query()
	dq.Set("error", errCode)
	if description != "" {
		dq.Set("error_description", description)
	}
	if state != "" {
		dq.Set("state", state)
	}
	dest.RawQuery = dq.Encode()
	w.Header().Set("Location", dest.String())
	w.WriteHeader(http.StatusFound)
}
