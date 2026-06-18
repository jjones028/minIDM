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

type clientGroupResponse struct {
	ID          pgtype.UUID        `json:"id"`
	ClientID    pgtype.UUID        `json:"client_id"`
	Name        string             `json:"name"`
	Description pgtype.Text        `json:"description"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
}

func toClientGroupResponse(g db.Oauth2ClientGroup) clientGroupResponse {
	return clientGroupResponse{
		ID:          g.ID,
		ClientID:    g.ClientID,
		Name:        g.Name,
		Description: g.Description,
		CreatedAt:   g.CreatedAt,
	}
}

func (a *API) ListClientGroups(w http.ResponseWriter, r *http.Request) {
	clientID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	groups, err := a.q.ListClientGroups(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []db.Oauth2ClientGroup{}
	}
	out := make([]clientGroupResponse, len(groups))
	for i, g := range groups {
		out[i] = toClientGroupResponse(g)
	}
	httputil.WriteJSON(w, out)
}

func (a *API) CreateClientGroup(w http.ResponseWriter, r *http.Request) {
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
	group, err := a.q.CreateClientGroup(r.Context(), db.CreateClientGroupParams{
		ClientID:    clientID,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSONStatus(w, http.StatusCreated, toClientGroupResponse(group))
}

func (a *API) UpdateClientGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
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
	group, err := a.q.UpdateClientGroup(r.Context(), db.UpdateClientGroupParams{
		ID:          groupID,
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
	httputil.WriteJSON(w, toClientGroupResponse(group))
}

func (a *API) DeleteClientGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	if err := a.q.DeleteClientGroup(r.Context(), groupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Members of a group ---

func (a *API) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.q.ListIdentitiesInClientGroup(r.Context(), groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []db.ListIdentitiesInClientGroupRow{}
	}
	httputil.WriteJSON(w, rows)
}

func (a *API) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
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
	if err := a.q.AddIdentityToClientGroup(r.Context(), db.AddIdentityToClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	identityID, err := httputil.ParseUUID(r.PathValue("identityId"))
	if err != nil {
		http.Error(w, "invalid_identity_id", http.StatusBadRequest)
		return
	}
	if err := a.q.RemoveIdentityFromClientGroup(r.Context(), db.RemoveIdentityFromClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Roles assigned to a group ---

func (a *API) ListGroupRoles(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.q.ListRolesForClientGroup(r.Context(), groupID)
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

func (a *API) AddRoleToGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	roleID, err := httputil.ParseUUID(req.RoleID)
	if err != nil {
		http.Error(w, "invalid_role_id", http.StatusBadRequest)
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

func (a *API) RemoveRoleFromGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_role_id", http.StatusBadRequest)
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
