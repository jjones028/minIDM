package oauth2

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"strings"

	db "minIDM/db/sqlc"

	"github.com/golang-jwt/jwt/v5"
)

// IntrospectHandler handles POST /oauth2/introspect (RFC 7662).
// Callers must authenticate with client_id + client_secret in the form body.
// Client auth failures return 401; token validation failures return {"active":false}.
type IntrospectHandler struct {
	q      *db.Queries
	key    *rsa.PrivateKey
	issuer string
}

func NewIntrospectHandler(q *db.Queries, key *rsa.PrivateKey, issuer string) *IntrospectHandler {
	return &IntrospectHandler{q: q, key: key, issuer: issuer}
}

func (h *IntrospectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if rawToken == "" || clientID == "" || clientSecret == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// Authenticate the requesting client — 401 on failure (not active:false).
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

	// Parse and verify JWT signature + expiry.
	var claims tokenClaims
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &h.key.PublicKey, nil
	}, jwt.WithIssuer(h.issuer), jwt.WithExpirationRequired())

	if err != nil || !parsed.Valid || claims.ID == "" {
		writeInactive(w)
		return
	}

	// Check revocation via JTI.
	tokenRow, err := h.q.GetOAuth2TokenByJTI(r.Context(), claims.ID)
	if err != nil {
		writeInactive(w)
		return
	}

	resp := map[string]any{
		"active":     true,
		"jti":        claims.ID,
		"iss":        claims.Issuer,
		"sub":        claims.Subject,
		"client_id":  tokenRow.ClientID,
		"scope":      claims.Scope,
		"token_type": "Bearer",
		"exp":        claims.ExpiresAt.Unix(),
		"iat":        claims.IssuedAt.Unix(),
	}
	if len(claims.Audience) > 0 {
		resp["aud"] = strings.Join([]string(claims.Audience), " ")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(resp)
}

func writeInactive(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"active": false})
}
