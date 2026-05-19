package oauth2

import (
	"crypto/rsa"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RevokeHandler handles POST /oauth2/revoke (RFC 7009).
// Callers must authenticate with client_id + client_secret in the form body.
// Per the RFC the server always responds 200 — never leak whether a token existed.
type RevokeHandler struct {
	q      *db.Queries
	key    *rsa.PrivateKey
	issuer string
}

func NewRevokeHandler(q *db.Queries, key *rsa.PrivateKey, issuer string) *RevokeHandler {
	return &RevokeHandler{q: q, key: key, issuer: issuer}
}

func (h *RevokeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	hint := r.FormValue("token_type_hint")

	if rawToken == "" || clientID == "" || clientSecret == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// Authenticate the requesting client — 401 on failure.
	client, err := h.q.GetOAuth2ClientByClientID(r.Context(), clientID)
	if err != nil || !client.IsEnabled {
		http.Error(w, "invalid_client", http.StatusUnauthorized)
		return
	}
	ok, err := VerifyClientSecret(clientSecret, client.ClientSecretHash)
	if err != nil || !ok {
		http.Error(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	// Attempt revocation. Use the hint to try the most-likely type first, then
	// fall back. Either way, always return 200 — the RFC forbids leaking whether
	// the token was known or not.
	switch hint {
	case "refresh_token":
		if !h.revokeRefreshToken(r, rawToken, clientID) {
			h.revokeAccessToken(r, rawToken, clientID)
		}
	default: // "access_token" or no hint
		if !h.revokeAccessToken(r, rawToken, clientID) {
			h.revokeRefreshToken(r, rawToken, clientID)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// revokeAccessToken parses the JWT (accepting expired tokens so clients can clean
// up after expiry), looks up the JTI, and revokes the row. Returns true if the
// row was found and revoked.
func (h *RevokeHandler) revokeAccessToken(r *http.Request, rawToken, clientID string) bool {
	var claims tokenClaims
	_, err := jwt.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &h.key.PublicKey, nil
	}, jwt.WithIssuer(h.issuer))

	// Accept expired tokens; any other error means this isn't one of our JWTs.
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return false
	}
	if claims.ID == "" {
		return false
	}

	tokenRow, err := h.q.GetOAuth2TokenByJTI(r.Context(), claims.ID)
	if err != nil || tokenRow.ClientID != clientID {
		return false
	}
	return h.q.RevokeOAuth2Token(r.Context(), tokenRow.ID) == nil
}

// revokeRefreshToken hashes the opaque token, looks it up, and revokes the row.
// Returns true if the row was found and revoked.
func (h *RevokeHandler) revokeRefreshToken(r *http.Request, rawToken, clientID string) bool {
	hash := HashRefreshToken(rawToken)
	tokenRow, err := h.q.GetOAuth2TokenByRefreshHash(r.Context(), pgtype.Text{String: hash, Valid: true})
	if err != nil || tokenRow.ClientID != clientID {
		return false
	}
	return h.q.RevokeOAuth2Token(r.Context(), tokenRow.ID) == nil
}
