package oauth2

import (
	"context"
	"fmt"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ---- Create ----

type CreateClientCommand struct {
	Name         string
	Description  string
	RedirectURIs []string
	Scopes       []string
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
	secret, err := GenerateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("generating client_secret: %w", err)
	}
	secretHash, err := HashClientSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("hashing client_secret: %w", err)
	}

	scopes := cmd.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	client, err := h.q.CreateOAuth2Client(ctx, db.CreateOAuth2ClientParams{
		ClientID:         clientID,
		ClientSecretHash: secretHash,
		Name:             cmd.Name,
		Description:      pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
		RedirectUris:     cmd.RedirectURIs,
		Scopes:           scopes,
	})
	if err != nil {
		return nil, err
	}
	return &CreateClientResult{Client: client, ClientSecret: secret}, nil
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
	ID           pgtype.UUID
	Name         string
	Description  string
	RedirectURIs []string
	Scopes       []string
	IsEnabled    bool
}

type UpdateClientHandler struct{ q *db.Queries }

func NewUpdateClientHandler(q *db.Queries) *UpdateClientHandler {
	return &UpdateClientHandler{q: q}
}

func (h *UpdateClientHandler) Handle(ctx context.Context, cmd UpdateClientCommand) (db.Oauth2Client, error) {
	return h.q.UpdateOAuth2Client(ctx, db.UpdateOAuth2ClientParams{
		ID:           cmd.ID,
		Name:         cmd.Name,
		Description:  pgtype.Text{String: cmd.Description, Valid: cmd.Description != ""},
		RedirectUris: cmd.RedirectURIs,
		Scopes:       cmd.Scopes,
		IsEnabled:    cmd.IsEnabled,
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
