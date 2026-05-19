package oauth2

import (
	"encoding/json"
	"net/http"
)

// DiscoveryHandler handles GET /.well-known/openid-configuration.
type DiscoveryHandler struct {
	issuer string
}

func NewDiscoveryHandler(issuer string) *DiscoveryHandler {
	return &DiscoveryHandler{issuer: issuer}
}

func (h *DiscoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth2/authorize",
		"token_endpoint":                        h.issuer + "/oauth2/token",
		"userinfo_endpoint":                     h.issuer + "/oauth2/userinfo",
		"jwks_uri":                              h.issuer + "/oauth2/jwks.json",
		"introspection_endpoint":                h.issuer + "/oauth2/introspect",
		"revocation_endpoint":                   h.issuer + "/oauth2/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "email"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}
