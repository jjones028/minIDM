package oauth2

import (
	"crypto/rsa"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
)

type Config struct {
	Queries      *db.Queries
	Key          *rsa.PrivateKey
	Issuer       string
	ProtectRead  func(http.Handler) http.Handler
	ProtectWrite func(http.Handler) http.Handler
	Auditor      *audit.Auditor
	Authenticate func(http.Handler) http.Handler
}

// API holds all OAuth2 services and protocol handlers.
type API struct {
	clients      *ClientService
	tokens       *TokenService
	roles        *RoleService
	groups       *GroupService
	inspectToken *InspectTokenHandler
	authorize    *AuthorizeHandler
	consent      *ConsentHandler
	clientInfo   *ClientInfoHandler
	token        *TokenHandler
	introspect   *IntrospectHandler
	revoke       *RevokeHandler
	userinfo     *UserinfoHandler
	discovery    *DiscoveryHandler
	jwks         *JWKSHandler
	auditor      *audit.Auditor
}

// Register wires up all OAuth2 routes into mux.
func Register(mux *http.ServeMux, cfg Config) {
	api := &API{
		clients:      NewClientService(cfg.Queries),
		tokens:       NewTokenService(cfg.Queries),
		roles:        &RoleService{q: cfg.Queries},
		groups:       &GroupService{q: cfg.Queries},
		inspectToken: NewInspectTokenHandler(cfg.Queries, cfg.Key, cfg.Issuer),
		authorize:    NewAuthorizeHandler(cfg.Queries, cfg.Key),
		consent:      NewConsentHandler(cfg.Queries, cfg.Key),
		clientInfo:   NewClientInfoHandler(cfg.Queries),
		token:        NewTokenHandler(cfg.Queries, cfg.Key, cfg.Issuer),
		introspect:   NewIntrospectHandler(cfg.Queries, cfg.Key, cfg.Issuer),
		revoke:       NewRevokeHandler(cfg.Queries, cfg.Key, cfg.Issuer),
		userinfo:     NewUserinfoHandler(cfg.Queries, cfg.Key, cfg.Issuer),
		discovery:    NewDiscoveryHandler(cfg.Issuer),
		jwks:         NewJWKSHandler(cfg.Key),
		auditor:      cfg.Auditor,
	}

	// OIDC discovery + JWKS (fully public)
	mux.Handle("GET /.well-known/openid-configuration", api.discovery)
	mux.Handle("GET /oauth2/jwks.json", api.jwks)

	// OAuth2 protocol endpoints (public — own auth logic)
	mux.Handle("GET /oauth2/authorize", api.authorize)
	mux.Handle("POST /oauth2/token", api.token)
	mux.Handle("POST /oauth2/introspect", api.introspect)
	mux.Handle("POST /oauth2/revoke", api.revoke)
	mux.Handle("GET /oauth2/userinfo", api.userinfo)

	// Consent: public client info + session-gated approval
	mux.Handle("GET /api/oauth2/client-info", api.clientInfo)
	mux.Handle("POST /api/oauth2/consent", cfg.Authenticate(api.consent))

	// Admin API — protected by RBAC
	mux.Handle("GET /api/oauth2/clients", cfg.ProtectRead(http.HandlerFunc(api.ListClients)))
	mux.Handle("POST /api/oauth2/clients", cfg.ProtectWrite(http.HandlerFunc(api.CreateClient)))
	mux.Handle("GET /api/oauth2/clients/{id}", cfg.ProtectRead(http.HandlerFunc(api.GetClient)))
	mux.Handle("PATCH /api/oauth2/clients/{id}", cfg.ProtectWrite(http.HandlerFunc(api.UpdateClient)))
	mux.Handle("DELETE /api/oauth2/clients/{id}", cfg.ProtectWrite(http.HandlerFunc(api.DeleteClient)))
	mux.Handle("POST /api/oauth2/clients/{id}/rotate-secret", cfg.ProtectWrite(http.HandlerFunc(api.RotateSecret)))
	mux.Handle("GET /api/oauth2/tokens", cfg.ProtectRead(http.HandlerFunc(api.ListTokens)))
	mux.Handle("DELETE /api/oauth2/tokens/{id}", cfg.ProtectWrite(http.HandlerFunc(api.RevokeToken)))
	mux.Handle("POST /api/oauth2/tokens/inspect", cfg.ProtectRead(http.HandlerFunc(api.InspectToken)))

	// Client roles
	mux.Handle("GET /api/oauth2/clients/{id}/roles", cfg.ProtectRead(http.HandlerFunc(api.ListClientRoles)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles", cfg.ProtectWrite(http.HandlerFunc(api.CreateClientRole)))
	mux.Handle("PATCH /api/oauth2/clients/{id}/roles/{roleId}", cfg.ProtectWrite(http.HandlerFunc(api.UpdateClientRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}", cfg.ProtectWrite(http.HandlerFunc(api.DeleteClientRole)))
	mux.Handle("GET /api/oauth2/clients/{id}/roles/{roleId}/identities", cfg.ProtectRead(http.HandlerFunc(api.ListIdentitiesWithRole)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles/{roleId}/identities", cfg.ProtectWrite(http.HandlerFunc(api.AssignIdentityToRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}/identities/{identityId}", cfg.ProtectWrite(http.HandlerFunc(api.RemoveIdentityFromRole)))
	mux.Handle("GET /api/oauth2/clients/{id}/roles/{roleId}/groups", cfg.ProtectRead(http.HandlerFunc(api.ListGroupsForRole)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles/{roleId}/groups", cfg.ProtectWrite(http.HandlerFunc(api.AssignGroupToRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}/groups/{groupId}", cfg.ProtectWrite(http.HandlerFunc(api.RemoveGroupFromRole)))

	// Client groups
	mux.Handle("GET /api/oauth2/clients/{id}/groups", cfg.ProtectRead(http.HandlerFunc(api.ListClientGroups)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups", cfg.ProtectWrite(http.HandlerFunc(api.CreateClientGroup)))
	mux.Handle("PATCH /api/oauth2/clients/{id}/groups/{groupId}", cfg.ProtectWrite(http.HandlerFunc(api.UpdateClientGroup)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}", cfg.ProtectWrite(http.HandlerFunc(api.DeleteClientGroup)))
	mux.Handle("GET /api/oauth2/clients/{id}/groups/{groupId}/members", cfg.ProtectRead(http.HandlerFunc(api.ListGroupMembers)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups/{groupId}/members", cfg.ProtectWrite(http.HandlerFunc(api.AddGroupMember)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}/members/{identityId}", cfg.ProtectWrite(http.HandlerFunc(api.RemoveGroupMember)))
	mux.Handle("GET /api/oauth2/clients/{id}/groups/{groupId}/roles", cfg.ProtectRead(http.HandlerFunc(api.ListGroupRoles)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups/{groupId}/roles", cfg.ProtectWrite(http.HandlerFunc(api.AddRoleToGroup)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}/roles/{roleId}", cfg.ProtectWrite(http.HandlerFunc(api.RemoveRoleFromGroup)))
}
