package oauth2

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
)

// JWKSHandler handles GET /oauth2/jwks.json.
// It returns the RSA public key as a JSON Web Key Set.
type JWKSHandler struct {
	key *rsa.PrivateKey
	kid string
}

func NewJWKSHandler(key *rsa.PrivateKey) *JWKSHandler {
	return &JWKSHandler{key: key, kid: KeyID(key)}
}

func (h *JWKSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jwks := map[string]any{
		"keys": []any{rsaPublicKeyToJWK(&h.key.PublicKey, h.kid)},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

// rsaPublicKeyToJWK converts an RSA public key to a JWK map.
func rsaPublicKeyToJWK(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   encodeExponent(pub.E),
	}
}

// encodeExponent encodes an RSA public exponent as a base64url big-endian integer
// with leading zero bytes stripped, as required by RFC 7518 §6.3.1.2.
func encodeExponent(e int) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(e))
	// Strip leading zero bytes
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return base64.RawURLEncoding.EncodeToString(b[i:])
}
