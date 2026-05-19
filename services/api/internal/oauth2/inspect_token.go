package oauth2

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"

	"github.com/golang-jwt/jwt/v5"
)

// InspectTokenHandler handles POST /api/oauth2/tokens/inspect.
// It is admin-only (RBAC oauth2_client:read) and returns decoded JWT claims
// plus live DB status so admins can diagnose token state without client credentials.
type InspectTokenHandler struct {
	q      *db.Queries
	key    *rsa.PrivateKey
	issuer string
}

func NewInspectTokenHandler(q *db.Queries, key *rsa.PrivateKey, issuer string) *InspectTokenHandler {
	return &InspectTokenHandler{q: q, key: key, issuer: issuer}
}

type inspectResponse struct {
	Header map[string]any  `json:"header,omitempty"`
	Claims map[string]any  `json:"claims,omitempty"`
	Status inspectStatus   `json:"status"`
}

type inspectStatus struct {
	SignatureValid bool   `json:"signature_valid"`
	Expired        bool   `json:"expired"`
	DBStatus       string `json:"db_status"` // "active" | "revoked" | "not_found" | "unknown"
	Active         bool   `json:"active"`
	Error          string `json:"error,omitempty"`
}

func (h *InspectTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var claims tokenClaims
	parsed, parseErr := jwt.ParseWithClaims(body.Token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &h.key.PublicKey, nil
	}, jwt.WithIssuer(h.issuer))

	// Malformed tokens can't be decoded at all.
	if errors.Is(parseErr, jwt.ErrTokenMalformed) {
		json.NewEncoder(w).Encode(inspectResponse{
			Status: inspectStatus{Error: "token is malformed — not a valid JWT"},
		})
		return
	}

	// Signature is valid if there's no error, or the only error is expiry.
	signatureValid := parseErr == nil || errors.Is(parseErr, jwt.ErrTokenExpired)
	expired := errors.Is(parseErr, jwt.ErrTokenExpired)

	// Collect the header from the parsed token.
	var header map[string]any
	if parsed != nil {
		header = parsed.Header
	}

	// Build the claims map from what was decoded (populated even on sig failure).
	claimsMap := map[string]any{}
	if claims.Issuer != "" {
		claimsMap["iss"] = claims.Issuer
	}
	if claims.Subject != "" {
		claimsMap["sub"] = claims.Subject
	}
	if len(claims.Audience) > 0 {
		claimsMap["aud"] = []string(claims.Audience)
	}
	if claims.ExpiresAt != nil {
		claimsMap["exp"] = claims.ExpiresAt.Unix()
	}
	if claims.IssuedAt != nil {
		claimsMap["iat"] = claims.IssuedAt.Unix()
	}
	if claims.ID != "" {
		claimsMap["jti"] = claims.ID
	}
	if claims.Email != "" {
		claimsMap["email"] = claims.Email
	}
	if claims.Scope != "" {
		claimsMap["scope"] = claims.Scope
	}
	if claims.Nonce != "" {
		claimsMap["nonce"] = claims.Nonce
	}

	// Check DB state. Uses a query that ignores revocation so we can distinguish
	// "revoked" from "never existed".
	dbStatus := "unknown"
	if claims.ID != "" {
		row, err := h.q.GetOAuth2TokenByJTIAny(r.Context(), claims.ID)
		if err != nil {
			dbStatus = "not_found"
		} else if row.Revoked {
			dbStatus = "revoked"
		} else {
			dbStatus = "active"
		}
	}

	active := signatureValid && !expired && dbStatus == "active"

	json.NewEncoder(w).Encode(inspectResponse{
		Header: header,
		Claims: claimsMap,
		Status: inspectStatus{
			SignatureValid: signatureValid,
			Expired:        expired,
			DBStatus:       dbStatus,
			Active:         active,
		},
	})
}
