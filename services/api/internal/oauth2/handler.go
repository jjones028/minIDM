package oauth2

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/rbac"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// clientResponse is the safe representation of an OAuth2 client (no secret hash).
type clientResponse struct {
	ID           pgtype.UUID        `json:"id"`
	ClientID     string             `json:"client_id"`
	Name         string             `json:"name"`
	Description  pgtype.Text        `json:"description"`
	RedirectURIs []string           `json:"redirect_uris"`
	Scopes       []string           `json:"scopes"`
	IsEnabled    bool               `json:"is_enabled"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"updated_at"`
}

func toClientResponse(c db.Oauth2Client) clientResponse {
	return clientResponse{
		ID:           c.ID,
		ClientID:     c.ClientID,
		Name:         c.Name,
		Description:  c.Description,
		RedirectURIs: c.RedirectUris,
		Scopes:       c.Scopes,
		IsEnabled:    c.IsEnabled,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// API holds all OAuth2 handlers.
type API struct {
	createClient *CreateClientHandler
	listClients  *ListClientsHandler
	getClient    *GetClientHandler
	updateClient *UpdateClientHandler
	deleteClient *DeleteClientHandler
	authorize    *AuthorizeHandler
	token        *TokenHandler
	userinfo     *UserinfoHandler
	discovery    *DiscoveryHandler
	jwks         *JWKSHandler
	auditor      *audit.Auditor
}

// Register wires up all OAuth2 routes into mux.
//
//   - Public (no auth): /.well-known/openid-configuration, /oauth2/jwks.json,
//     POST /oauth2/token, GET /oauth2/userinfo
//   - Session-gated: GET /oauth2/authorize (handler checks session itself)
//   - Admin API (RBAC): /api/oauth2/clients
func Register(
	mux *http.ServeMux,
	q *db.Queries,
	key *rsa.PrivateKey,
	issuer string,
	protectClientRead func(http.Handler) http.Handler,
	protectClientWrite func(http.Handler) http.Handler,
	auditor *audit.Auditor,
) {
	api := &API{
		createClient: NewCreateClientHandler(q),
		listClients:  NewListClientsHandler(q),
		getClient:    NewGetClientHandler(q),
		updateClient: NewUpdateClientHandler(q),
		deleteClient: NewDeleteClientHandler(q),
		authorize:    NewAuthorizeHandler(q, key),
		token:        NewTokenHandler(q, key, issuer),
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
	mux.Handle("POST /oauth2/token", http.HandlerFunc(api.token.ServeHTTP))
	mux.Handle("GET /oauth2/userinfo", api.userinfo)

	// Admin API — protected by RBAC
	mux.Handle("GET /api/oauth2/clients", protectClientRead(http.HandlerFunc(api.ListClients)))
	mux.Handle("POST /api/oauth2/clients", protectClientWrite(http.HandlerFunc(api.CreateClient)))
	mux.Handle("GET /api/oauth2/clients/{id}", protectClientRead(http.HandlerFunc(api.GetClient)))
	mux.Handle("PATCH /api/oauth2/clients/{id}", protectClientWrite(http.HandlerFunc(api.UpdateClient)))
	mux.Handle("DELETE /api/oauth2/clients/{id}", protectClientWrite(http.HandlerFunc(api.DeleteClient)))
}

// --- HTTP handlers ---

func (a *API) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := a.listClients.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]clientResponse, len(clients))
	for i, c := range clients {
		out[i] = toClientResponse(c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (a *API) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name_required", http.StatusUnprocessableEntity)
		return
	}
	if len(req.RedirectURIs) == 0 {
		http.Error(w, "redirect_uris_required", http.StatusUnprocessableEntity)
		return
	}

	result, err := a.createClient.Handle(r.Context(), CreateClientCommand{
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_client.create", "oauth2_client", audit.UUIDStr(result.Client.ID), map[string]any{
		"name":      result.Client.Name,
		"client_id": result.Client.ClientID,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"client":        toClientResponse(result.Client),
		"client_secret": result.ClientSecret, // shown once
	})
}

func (a *API) GetClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	client, err := a.getClient.Handle(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toClientResponse(client))
}

func (a *API) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
		IsEnabled    bool     `json:"is_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name_required", http.StatusUnprocessableEntity)
		return
	}

	client, err := a.updateClient.Handle(r.Context(), UpdateClientCommand{
		ID:           id,
		Name:         req.Name,
		Description:  req.Description,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		IsEnabled:    req.IsEnabled,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_client.update", "oauth2_client", audit.UUIDStr(id), map[string]any{
		"name": req.Name,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toClientResponse(client))
}

func (a *API) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	if err := a.deleteClient.Handle(r.Context(), DeleteClientCommand{ID: id}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_client.delete", "oauth2_client", audit.UUIDStr(id), nil)
	w.WriteHeader(http.StatusNoContent)
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}
