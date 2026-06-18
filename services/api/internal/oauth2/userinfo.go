package oauth2

import (
	"crypto/rsa"
	"net/http"
	"strings"

	db "minIDM/db/sqlc"
	"minIDM/internal/httputil"

	"github.com/golang-jwt/jwt/v5"
)

// UserinfoHandler handles GET /oauth2/userinfo.
// It accepts a Bearer access token (RS256 JWT), validates it, and returns OIDC claims.
type UserinfoHandler struct {
	q      *db.Queries
	key    *rsa.PrivateKey
	issuer string
}

func NewUserinfoHandler(q *db.Queries, key *rsa.PrivateKey, issuer string) *UserinfoHandler {
	return &UserinfoHandler{q: q, key: key, issuer: issuer}
}

func (h *UserinfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract Bearer token
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
		return
	}
	rawToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and verify JWT signature + expiry
	var claims tokenClaims
	token, err := jwt.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &h.key.PublicKey, nil
	}, jwt.WithIssuer(h.issuer), jwt.WithExpirationRequired())

	if err != nil || !token.Valid {
		http.Error(w, `{"error":"invalid_token","error_description":"token validation failed"}`, http.StatusUnauthorized)
		return
	}

	// Check revocation via jti
	jti := claims.ID
	if jti == "" {
		http.Error(w, `{"error":"invalid_token","error_description":"missing jti claim"}`, http.StatusUnauthorized)
		return
	}
	if _, err := h.q.GetOAuth2TokenByJTI(r.Context(), jti); err != nil {
		http.Error(w, `{"error":"invalid_token","error_description":"token has been revoked"}`, http.StatusUnauthorized)
		return
	}

	// Look up identity by subject_id
	identity, err := h.q.GetIdentityBySub(r.Context(), claims.Subject)
	if err != nil {
		http.Error(w, `{"error":"invalid_token","error_description":"identity not found"}`, http.StatusUnauthorized)
		return
	}
	if !identity.IsEnabled {
		http.Error(w, `{"error":"invalid_token","error_description":"identity is disabled"}`, http.StatusUnauthorized)
		return
	}

	resp := map[string]any{
		"sub": identity.SubjectID,
	}
	if claims.Email != "" {
		resp["email"] = identity.Email
	}

	httputil.WriteJSON(w, resp)
}
