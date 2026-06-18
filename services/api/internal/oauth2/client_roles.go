package oauth2

import (
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/httputil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type clientRoleResponse struct {
	ID          pgtype.UUID        `json:"id"`
	ClientID    pgtype.UUID        `json:"client_id"`
	Name        string             `json:"name"`
	Description pgtype.Text        `json:"description"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
}

func toClientRoleResponse(r db.Oauth2ClientRole) clientRoleResponse {
	return clientRoleResponse{
		ID:          r.ID,
		ClientID:    r.ClientID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}

func (a *API) ListClientRoles(w http.ResponseWriter, r *http.Request) {
	clientID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.q.ListClientRoles(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if roles == nil {
		roles = []db.Oauth2ClientRole{}
	}
	out := make([]clientRoleResponse, len(roles))
	for i, role := range roles {
		out[i] = toClientRoleResponse(role)
	}
	httputil.WriteJSON(w, out)
}

func (a *API) CreateClientRole(w http.ResponseWriter, r *http.Request) {
	clientID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name_required", http.StatusUnprocessableEntity)
		return
	}
	role, err := a.q.CreateClientRole(r.Context(), db.CreateClientRoleParams{
		ClientID:    clientID,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSONStatus(w, http.StatusCreated, toClientRoleResponse(role))
}

func (a *API) UpdateClientRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name_required", http.StatusUnprocessableEntity)
		return
	}
	role, err := a.q.UpdateClientRole(r.Context(), db.UpdateClientRoleParams{
		ID:          roleID,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, toClientRoleResponse(role))
}

func (a *API) DeleteClientRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	if err := a.q.DeleteClientRole(r.Context(), roleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Identity assignments for a role ---

func (a *API) ListIdentitiesWithRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.q.ListIdentitiesWithClientRole(r.Context(), roleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []db.ListIdentitiesWithClientRoleRow{}
	}
	httputil.WriteJSON(w, rows)
}

func (a *API) AssignIdentityToRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		IdentityID string `json:"identity_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	identityID, err := httputil.ParseUUID(req.IdentityID)
	if err != nil {
		http.Error(w, "invalid_identity_id", http.StatusBadRequest)
		return
	}
	if err := a.q.AssignIdentityToClientRole(r.Context(), db.AssignIdentityToClientRoleParams{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RemoveIdentityFromRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	identityID, err := httputil.ParseUUID(r.PathValue("identityId"))
	if err != nil {
		http.Error(w, "invalid_identity_id", http.StatusBadRequest)
		return
	}
	if err := a.q.RemoveIdentityFromClientRole(r.Context(), db.RemoveIdentityFromClientRoleParams{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Group assignments for a role ---

func (a *API) ListGroupsForRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.q.ListGroupsForClientRole(r.Context(), roleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []db.ListGroupsForClientRoleRow{}
	}
	httputil.WriteJSON(w, rows)
}

func (a *API) AssignGroupToRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		GroupID string `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	groupID, err := httputil.ParseUUID(req.GroupID)
	if err != nil {
		http.Error(w, "invalid_group_id", http.StatusBadRequest)
		return
	}
	if err := a.q.AssignRoleToClientGroup(r.Context(), db.AssignRoleToClientGroupParams{
		GroupID: groupID,
		RoleID:  roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RemoveGroupFromRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_group_id", http.StatusBadRequest)
		return
	}
	if err := a.q.RemoveRoleFromClientGroup(r.Context(), db.RemoveRoleFromClientGroupParams{
		GroupID: groupID,
		RoleID:  roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
