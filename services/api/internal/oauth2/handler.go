package oauth2

import (
	"crypto/rsa"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
)

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
func Register(
	mux *http.ServeMux,
	q *db.Queries,
	key *rsa.PrivateKey,
	issuer string,
	protectClientRead func(http.Handler) http.Handler,
	protectClientWrite func(http.Handler) http.Handler,
	auditor *audit.Auditor,
	authenticate func(http.Handler) http.Handler,
) {
	api := &API{
		clients:      NewClientService(q),
		tokens:       NewTokenService(q),
		roles:        &RoleService{q: q},
		groups:       &GroupService{q: q},
		inspectToken: NewInspectTokenHandler(q, key, issuer),
		authorize:    NewAuthorizeHandler(q, key),
		consent:      NewConsentHandler(q, key),
		clientInfo:   NewClientInfoHandler(q),
		token:        NewTokenHandler(q, key, issuer),
		introspect:   NewIntrospectHandler(q, key, issuer),
		revoke:       NewRevokeHandler(q, key, issuer),
		userinfo:     NewUserinfoHandler(q, key, issuer),
		discovery:    NewDiscoveryHandler(issuer),
		jwks:         NewJWKSHandler(key),
		auditor:      auditor,
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
	mux.Handle("POST /api/oauth2/consent", authenticate(api.consent))

	// Admin API — protected by RBAC
	mux.Handle("GET /api/oauth2/clients", protectClientRead(http.HandlerFunc(api.ListClients)))
	mux.Handle("POST /api/oauth2/clients", protectClientWrite(http.HandlerFunc(api.CreateClient)))
	mux.Handle("GET /api/oauth2/clients/{id}", protectClientRead(http.HandlerFunc(api.GetClient)))
	mux.Handle("PATCH /api/oauth2/clients/{id}", protectClientWrite(http.HandlerFunc(api.UpdateClient)))
	mux.Handle("DELETE /api/oauth2/clients/{id}", protectClientWrite(http.HandlerFunc(api.DeleteClient)))
	mux.Handle("POST /api/oauth2/clients/{id}/rotate-secret", protectClientWrite(http.HandlerFunc(api.RotateSecret)))
	mux.Handle("GET /api/oauth2/tokens", protectClientRead(http.HandlerFunc(api.ListTokens)))
	mux.Handle("DELETE /api/oauth2/tokens/{id}", protectClientWrite(http.HandlerFunc(api.RevokeToken)))
	mux.Handle("POST /api/oauth2/tokens/inspect", protectClientRead(http.HandlerFunc(api.InspectToken)))

	// Client roles
	mux.Handle("GET /api/oauth2/clients/{id}/roles", protectClientRead(http.HandlerFunc(api.ListClientRoles)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles", protectClientWrite(http.HandlerFunc(api.CreateClientRole)))
	mux.Handle("PATCH /api/oauth2/clients/{id}/roles/{roleId}", protectClientWrite(http.HandlerFunc(api.UpdateClientRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}", protectClientWrite(http.HandlerFunc(api.DeleteClientRole)))
	mux.Handle("GET /api/oauth2/clients/{id}/roles/{roleId}/identities", protectClientRead(http.HandlerFunc(api.ListIdentitiesWithRole)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles/{roleId}/identities", protectClientWrite(http.HandlerFunc(api.AssignIdentityToRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}/identities/{identityId}", protectClientWrite(http.HandlerFunc(api.RemoveIdentityFromRole)))
	mux.Handle("GET /api/oauth2/clients/{id}/roles/{roleId}/groups", protectClientRead(http.HandlerFunc(api.ListGroupsForRole)))
	mux.Handle("POST /api/oauth2/clients/{id}/roles/{roleId}/groups", protectClientWrite(http.HandlerFunc(api.AssignGroupToRole)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/roles/{roleId}/groups/{groupId}", protectClientWrite(http.HandlerFunc(api.RemoveGroupFromRole)))

	// Client groups
	mux.Handle("GET /api/oauth2/clients/{id}/groups", protectClientRead(http.HandlerFunc(api.ListClientGroups)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups", protectClientWrite(http.HandlerFunc(api.CreateClientGroup)))
	mux.Handle("PATCH /api/oauth2/clients/{id}/groups/{groupId}", protectClientWrite(http.HandlerFunc(api.UpdateClientGroup)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}", protectClientWrite(http.HandlerFunc(api.DeleteClientGroup)))
	mux.Handle("GET /api/oauth2/clients/{id}/groups/{groupId}/members", protectClientRead(http.HandlerFunc(api.ListGroupMembers)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups/{groupId}/members", protectClientWrite(http.HandlerFunc(api.AddGroupMember)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}/members/{identityId}", protectClientWrite(http.HandlerFunc(api.RemoveGroupMember)))
	mux.Handle("GET /api/oauth2/clients/{id}/groups/{groupId}/roles", protectClientRead(http.HandlerFunc(api.ListGroupRoles)))
	mux.Handle("POST /api/oauth2/clients/{id}/groups/{groupId}/roles", protectClientWrite(http.HandlerFunc(api.AddRoleToGroup)))
	mux.Handle("DELETE /api/oauth2/clients/{id}/groups/{groupId}/roles/{roleId}", protectClientWrite(http.HandlerFunc(api.RemoveRoleFromGroup)))
}
