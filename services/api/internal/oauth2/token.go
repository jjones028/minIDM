package oauth2

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	db "minIDM/db/sqlc"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

// tokenClaims is the JWT payload for both access tokens and id_tokens.
type tokenClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
	Scope string `json:"scope,omitempty"`
	Nonce string `json:"nonce,omitempty"`
}

// TokenHandler handles POST /oauth2/token.
type TokenHandler struct {
	q      *db.Queries
	key    *rsa.PrivateKey
	kid    string
	issuer string
}

func NewTokenHandler(q *db.Queries, key *rsa.PrivateKey, issuer string) *TokenHandler {
	return &TokenHandler{q: q, key: key, kid: KeyID(key), issuer: issuer}
}

func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		tokenError(w, "invalid_request", "failed to parse form body", http.StatusBadRequest)
		return
	}

	switch r.FormValue("grant_type") {
	case "authorization_code":
		h.handleAuthorizationCode(w, r)
	case "refresh_token":
		h.handleRefreshToken(w, r)
	default:
		tokenError(w, "unsupported_grant_type", "supported: authorization_code, refresh_token", http.StatusBadRequest)
	}
}

// handleAuthorizationCode exchanges an authorization code for tokens.
func (h *TokenHandler) handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" || redirectURI == "" || clientID == "" || clientSecret == "" || codeVerifier == "" {
		tokenError(w, "invalid_request", "missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate client
	client, err := h.q.GetOAuth2ClientByClientID(r.Context(), clientID)
	if err != nil {
		tokenError(w, "invalid_client", "unknown client", http.StatusUnauthorized)
		return
	}
	ok, err := VerifyClientSecret(clientSecret, client.ClientSecretHash)
	if err != nil || !ok {
		tokenError(w, "invalid_client", "invalid client_secret", http.StatusUnauthorized)
		return
	}
	if !client.IsEnabled {
		tokenError(w, "invalid_client", "client is disabled", http.StatusUnauthorized)
		return
	}

	// Fetch & validate authorization code
	authCode, err := h.q.GetAuthorizationCode(r.Context(), code)
	if err != nil {
		tokenError(w, "invalid_grant", "authorization code not found or expired", http.StatusBadRequest)
		return
	}
	if authCode.ClientID != clientID {
		tokenError(w, "invalid_grant", "code was not issued to this client", http.StatusBadRequest)
		return
	}
	if authCode.RedirectUri != redirectURI {
		tokenError(w, "invalid_grant", "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE
	computed := ComputeCodeChallenge(codeVerifier)
	if computed != authCode.CodeChallenge {
		tokenError(w, "invalid_grant", "code_verifier does not match code_challenge", http.StatusBadRequest)
		return
	}

	// Mark code used (single-use)
	if err := h.q.MarkAuthorizationCodeUsed(r.Context(), code); err != nil {
		tokenError(w, "server_error", "failed to consume authorization code", http.StatusInternalServerError)
		return
	}

	// Look up identity
	identity, err := h.q.GetIdentityByID(r.Context(), authCode.IdentityID)
	if err != nil {
		tokenError(w, "server_error", "identity not found", http.StatusInternalServerError)
		return
	}

	h.issueAndRespond(w, r, client, identity, authCode.Scopes, authCode.Nonce.String)
}

// handleRefreshToken exchanges a refresh token for a new access token.
func (h *TokenHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if refreshToken == "" || clientID == "" || clientSecret == "" {
		tokenError(w, "invalid_request", "missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate client
	client, err := h.q.GetOAuth2ClientByClientID(r.Context(), clientID)
	if err != nil {
		tokenError(w, "invalid_client", "unknown client", http.StatusUnauthorized)
		return
	}
	ok, err := VerifyClientSecret(clientSecret, client.ClientSecretHash)
	if err != nil || !ok {
		tokenError(w, "invalid_client", "invalid client_secret", http.StatusUnauthorized)
		return
	}

	// Look up refresh token
	hash := HashRefreshToken(refreshToken)
	stored, err := h.q.GetOAuth2TokenByRefreshHash(r.Context(), pgtype.Text{String: hash, Valid: true})
	if err != nil {
		tokenError(w, "invalid_grant", "refresh token not found or expired", http.StatusBadRequest)
		return
	}
	if stored.ClientID != clientID {
		tokenError(w, "invalid_grant", "refresh token was not issued to this client", http.StatusBadRequest)
		return
	}

	// Revoke old token record
	if err := h.q.RevokeOAuth2Token(r.Context(), stored.ID); err != nil {
		tokenError(w, "server_error", "failed to revoke old token", http.StatusInternalServerError)
		return
	}

	// Look up identity
	identity, err := h.q.GetIdentityByID(r.Context(), stored.IdentityID)
	if err != nil {
		tokenError(w, "server_error", "identity not found", http.StatusInternalServerError)
		return
	}

	h.issueAndRespond(w, r, client, identity, stored.Scopes, "")
}

// issueAndRespond mints tokens and writes the token response JSON.
// nonce is included in the id_token claim when non-empty (OIDC nonce binding).
func (h *TokenHandler) issueAndRespond(w http.ResponseWriter, r *http.Request, client db.Oauth2Client, identity db.Identity, scopes []string, nonce string) {
	scopeStr := strings.Join(scopes, " ")
	hasOpenID := containsScope(scopes, "openid")
	hasEmail := containsScope(scopes, "email")

	now := time.Now()
	accessExpiry := now.Add(accessTokenTTL)
	jti := uuid.New().String()

	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   identity.SubjectID,
			Audience:  jwt.ClaimStrings{client.ClientID},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			ID:        jti,
		},
		Scope: scopeStr,
	}
	if hasEmail {
		claims.Email = identity.Email
	}
	if hasOpenID && nonce != "" {
		claims.Nonce = nonce
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t.Header["kid"] = h.kid
	accessToken, err := t.SignedString(h.key)
	if err != nil {
		tokenError(w, "server_error", "failed to sign access token", http.StatusInternalServerError)
		return
	}

	// Refresh token (opaque)
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		tokenError(w, "server_error", "failed to generate refresh token", http.StatusInternalServerError)
		return
	}
	refreshHash := HashRefreshToken(refreshToken)
	refreshExpiry := now.Add(refreshTokenTTL)

	if _, err := h.q.CreateOAuth2Token(r.Context(), db.CreateOAuth2TokenParams{
		ClientID:         client.ClientID,
		IdentityID:       identity.ID,
		Jti:              jti,
		RefreshTokenHash: pgtype.Text{String: refreshHash, Valid: true},
		Scopes:           scopes,
		ExpiresAt:        pgtype.Timestamptz{Time: refreshExpiry, Valid: true},
	}); err != nil {
		tokenError(w, "server_error", "failed to persist token", http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         scopeStr,
	}
	// Include id_token for OpenID Connect clients
	if hasOpenID {
		resp["id_token"] = accessToken // same JWT satisfies OIDC id_token requirements
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(resp)
}

func containsScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

func tokenError(w http.ResponseWriter, errCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
