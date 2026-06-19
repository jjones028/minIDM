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
	ClientSecret string
}

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

// ClientService manages OAuth2 client registration and secrets.
type ClientService struct {
	q *db.Queries
}

func NewClientService(q *db.Queries) *ClientService {
	return &ClientService{q: q}
}

func (s *ClientService) Create(ctx context.Context, cmd CreateClientCommand) (*CreateClientResult, error) {
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

	client, err := s.q.CreateOAuth2Client(ctx, db.CreateOAuth2ClientParams{
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

func (s *ClientService) List(ctx context.Context) ([]db.Oauth2Client, error) {
	return s.q.ListOAuth2Clients(ctx)
}

func (s *ClientService) Get(ctx context.Context, id pgtype.UUID) (db.Oauth2Client, error) {
	return s.q.GetOAuth2ClientByID(ctx, id)
}

func (s *ClientService) Update(ctx context.Context, cmd UpdateClientCommand) (db.Oauth2Client, error) {
	return s.q.UpdateOAuth2Client(ctx, db.UpdateOAuth2ClientParams{
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

func (s *ClientService) Delete(ctx context.Context, id pgtype.UUID) error {
	return s.q.DeleteOAuth2Client(ctx, id)
}

func (s *ClientService) RotateSecret(ctx context.Context, id pgtype.UUID) (string, error) {
	client, err := s.q.GetOAuth2ClientByID(ctx, id)
	if err != nil {
		return "", err
	}
	if !client.ClientSecretHash.Valid {
		return "", ErrPublicClient
	}
	secret, err := GenerateClientSecret()
	if err != nil {
		return "", fmt.Errorf("generating client_secret: %w", err)
	}
	hash, err := HashClientSecret(secret)
	if err != nil {
		return "", fmt.Errorf("hashing client_secret: %w", err)
	}
	if err := s.q.UpdateOAuth2ClientSecret(ctx, db.UpdateOAuth2ClientSecretParams{
		ID:               id,
		ClientSecretHash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		return "", err
	}
	return secret, nil
}

// --- HTTP handlers ---

func (a *API) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := a.clients.List(r.Context())
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
	result, err := a.clients.Create(r.Context(), CreateClientCommand{
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
		"client_secret": result.ClientSecret,
	})
}

func (a *API) GetClient(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	client, err := a.clients.Get(r.Context(), id)
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
	client, err := a.clients.Update(r.Context(), UpdateClientCommand{
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
	if err := a.clients.Delete(r.Context(), id); err != nil {
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
	secret, err := a.clients.RotateSecret(r.Context(), id)
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
	httputil.WriteJSON(w, map[string]string{"client_secret": secret})
}
