package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/httputil"
	"minIDM/internal/rbac"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrPublicClient = errors.New("operation not supported for public clients")

// clientResponse is the safe representation of an OAuth2 client (no secret hash).
type clientResponse struct {
	ID                pgtype.UUID        `json:"id"`
	ClientID          string             `json:"client_id"`
	Name              string             `json:"name"`
	Description       pgtype.Text        `json:"description"`
	RedirectURIs      []string           `json:"redirect_uris"`
	Scopes            []string           `json:"scopes"`
	IsEnabled         bool               `json:"is_enabled"`
	IsPublic          bool               `json:"is_public"`
	AutoConsent       bool               `json:"auto_consent"`
	AllowRegistration bool               `json:"allow_registration"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	UpdatedAt         pgtype.Timestamptz `json:"updated_at"`
}

func toClientResponse(c db.Oauth2Client) clientResponse {
	return clientResponse{
		ID:                c.ID,
		ClientID:          c.ClientID,
		Name:              c.Name,
		Description:       c.Description,
		RedirectURIs:      c.RedirectUris,
		Scopes:            c.Scopes,
		IsEnabled:         c.IsEnabled,
		IsPublic:          !c.ClientSecretHash.Valid,
		AutoConsent:       c.AutoConsent,
		AllowRegistration: c.AllowRegistration,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

// ---- Create ----

type CreateClientCommand struct {
	Name              string
	Description       string
	RedirectURIs      []string
	Scopes            []string
	AutoConsent       bool
	IsPublic          bool
	AllowRegistration bool
}

type CreateClientResult struct {
	Client       db.Oauth2Client
	ClientSecret string // plaintext, shown once
}

type CreateClientHandler struct{ q *db.Queries }

func NewCreateClientHandler(q *db.Queries) *CreateClientHandler {
	return &CreateClientHandler{q: q}
}

func (h *CreateClientHandler) Handle(ctx context.Context, cmd CreateClientCommand) (*CreateClientResult, error) {
	clientID, err := GenerateClientID()
	if err != nil {
		return nil, fmt.Errorf("generating client_id: %w", err)
	}

	var secretHash pgtype.Text
	var plainSecret string
	if !cmd.IsPublic {
		plainSecret, err = GenerateClientSecret()
		if err != nil {
			return nil, fmt.Errorf("generating client_secret: %w", err)
		}
		h, err := HashClientSecret(plainSecret)
		if err != nil {
			return nil, fmt.Errorf("hashing client_secret: %w", err)
		}
		secretHash = pgtype.Text{String: h, Valid: true}
	}

	scopes := cmd.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	client, err := h.q.CreateOAuth2Client(ctx, db.CreateOAuth2ClientParams{
		ClientID:          clientID,
		ClientSecretHash:  secretHash,
		Name:              cmd.Name,
		Description:       pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
		RedirectUris:      cmd.RedirectURIs,
		Scopes:            scopes,
		AutoConsent:       cmd.AutoConsent,
		AllowRegistration: cmd.AllowRegistration,
	})
	if err != nil {
		return nil, err
	}
	return &CreateClientResult{Client: client, ClientSecret: plainSecret}, nil
}

// ---- List ----

type ListClientsHandler struct{ q *db.Queries }

func NewListClientsHandler(q *db.Queries) *ListClientsHandler {
	return &ListClientsHandler{q: q}
}

func (h *ListClientsHandler) Handle(ctx context.Context) ([]db.Oauth2Client, error) {
	return h.q.ListOAuth2Clients(ctx)
}

// ---- Get ----

type GetClientHandler struct{ q *db.Queries }

func NewGetClientHandler(q *db.Queries) *GetClientHandler {
	return &GetClientHandler{q: q}
}

func (h *GetClientHandler) Handle(ctx context.Context, id pgtype.UUID) (db.Oauth2Client, error) {
	return h.q.GetOAuth2ClientByID(ctx, id)
}

// ---- Update ----

type UpdateClientCommand struct {
	ID                pgtype.UUID
	Name              string
	Description       string
	RedirectURIs      []string
	Scopes            []string
	IsEnabled         bool
	AutoConsent       bool
	AllowRegistration bool
}

type UpdateClientHandler struct{ q *db.Queries }

func NewUpdateClientHandler(q *db.Queries) *UpdateClientHandler {
	return &UpdateClientHandler{q: q}
}

func (h *UpdateClientHandler) Handle(ctx context.Context, cmd UpdateClientCommand) (db.Oauth2Client, error) {
	return h.q.UpdateOAuth2Client(ctx, db.UpdateOAuth2ClientParams{
		ID:                cmd.ID,
		Name:              cmd.Name,
		Description:       pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
		RedirectUris:      cmd.RedirectURIs,
		Scopes:            cmd.Scopes,
		IsEnabled:         cmd.IsEnabled,
		AutoConsent:       cmd.AutoConsent,
		AllowRegistration: cmd.AllowRegistration,
	})
}

// ---- Delete ----

type DeleteClientCommand struct{ ID pgtype.UUID }

type DeleteClientHandler struct{ q *db.Queries }

func NewDeleteClientHandler(q *db.Queries) *DeleteClientHandler {
	return &DeleteClientHandler{q: q}
}

func (h *DeleteClientHandler) Handle(ctx context.Context, cmd DeleteClientCommand) error {
	return h.q.DeleteOAuth2Client(ctx, cmd.ID)
}

// ---- Rotate Secret ----

type RotateSecretCommand struct{ ID pgtype.UUID }

type RotateSecretResult struct{ ClientSecret string }

type RotateSecretHandler struct{ q *db.Queries }

func NewRotateSecretHandler(q *db.Queries) *RotateSecretHandler {
	return &RotateSecretHandler{q: q}
}

func (h *RotateSecretHandler) Handle(ctx context.Context, cmd RotateSecretCommand) (RotateSecretResult, error) {
	client, err := h.q.GetOAuth2ClientByID(ctx, cmd.ID)
	if err != nil {
		return RotateSecretResult{}, err
	}
	if !client.ClientSecretHash.Valid {
		return RotateSecretResult{}, ErrPublicClient
	}
	secret, err := GenerateClientSecret()
	if err != nil {
		return RotateSecretResult{}, fmt.Errorf("generating client_secret: %w", err)
	}
	hash, err := HashClientSecret(secret)
	if err != nil {
		return RotateSecretResult{}, fmt.Errorf("hashing client_secret: %w", err)
	}
	if err := h.q.UpdateOAuth2ClientSecret(ctx, db.UpdateOAuth2ClientSecretParams{
		ID:               cmd.ID,
		ClientSecretHash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		return RotateSecretResult{}, err
	}
	return RotateSecretResult{ClientSecret: secret}, nil
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
	httputil.WriteJSON(w, out)
}

func (a *API) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		RedirectURIs      []string `json:"redirect_uris"`
		Scopes            []string `json:"scopes"`
		AutoConsent       bool     `json:"auto_consent"`
		IsPublic          bool     `json:"is_public"`
		AllowRegistration bool     `json:"allow_registration"`
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
		Name:              req.Name,
		Description:       req.Description,
		RedirectURIs:      req.RedirectURIs,
		Scopes:            req.Scopes,
		AutoConsent:       req.AutoConsent,
		IsPublic:          req.IsPublic,
		AllowRegistration: req.AllowRegistration,
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
	httputil.WriteJSONStatus(w, http.StatusCreated, map[string]any{
		"client":        toClientResponse(result.Client),
		"client_secret": result.ClientSecret, // shown once
	})
}

func (a *API) GetClient(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
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
	httputil.WriteJSON(w, toClientResponse(client))
}

func (a *API) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		RedirectURIs      []string `json:"redirect_uris"`
		Scopes            []string `json:"scopes"`
		IsEnabled         bool     `json:"is_enabled"`
		AutoConsent       bool     `json:"auto_consent"`
		AllowRegistration bool     `json:"allow_registration"`
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
		ID:                id,
		Name:              req.Name,
		Description:       req.Description,
		RedirectURIs:      req.RedirectURIs,
		Scopes:            req.Scopes,
		IsEnabled:         req.IsEnabled,
		AutoConsent:       req.AutoConsent,
		AllowRegistration: req.AllowRegistration,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_client.update", "oauth2_client", audit.UUIDStr(id), map[string]any{
		"name": req.Name,
	})
	httputil.WriteJSON(w, toClientResponse(client))
}

func (a *API) DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
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

func (a *API) RotateSecret(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	result, err := a.rotateSecret.Handle(r.Context(), RotateSecretCommand{ID: id})
	if errors.Is(err, ErrPublicClient) {
		http.Error(w, "public clients do not have a secret", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "oauth2_client.rotate_secret", "oauth2_client", audit.UUIDStr(id), nil)
	httputil.WriteJSON(w, map[string]string{"client_secret": result.ClientSecret})
}
