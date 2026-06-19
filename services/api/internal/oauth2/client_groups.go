package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/httputil"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// clientGroupResponse is the safe representation of a client group.
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

// GroupService manages per-client group definitions and membership.
type GroupService struct{ q *db.Queries }

func NewGroupService(q *db.Queries) *GroupService { return &GroupService{q: q} }

// ListForIdentity returns all client group memberships for an identity.
func (h *GroupService) ListForIdentity(ctx context.Context, identityID pgtype.UUID) ([]db.ListClientGroupsForIdentityRow, error) {
	rows, err := h.q.ListClientGroupsForIdentity(ctx, identityID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []db.ListClientGroupsForIdentityRow{}
	}
	return rows, nil
}

// RemoveFromIdentity removes an identity from a client group.
func (h *GroupService) RemoveFromIdentity(ctx context.Context, identityID, groupID pgtype.UUID) error {
	return h.q.RemoveIdentityFromClientGroup(ctx, db.RemoveIdentityFromClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	})
}

func (h *GroupService) listForClient(ctx context.Context, clientID pgtype.UUID) ([]db.Oauth2ClientGroup, error) {
	rows, err := h.q.ListClientGroups(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []db.Oauth2ClientGroup{}
	}
	return rows, nil
}

func (h *GroupService) createGroup(ctx context.Context, clientID pgtype.UUID, name, description string) (db.Oauth2ClientGroup, error) {
	return h.q.CreateClientGroup(ctx, db.CreateClientGroupParams{
		ClientID:    clientID,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (h *GroupService) updateGroup(ctx context.Context, groupID pgtype.UUID, name, description string) (db.Oauth2ClientGroup, error) {
	return h.q.UpdateClientGroup(ctx, db.UpdateClientGroupParams{
		ID:          groupID,
		Name:        name,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (h *GroupService) deleteGroup(ctx context.Context, groupID pgtype.UUID) error {
	return h.q.DeleteClientGroup(ctx, groupID)
}

func (h *GroupService) listMembers(ctx context.Context, groupID pgtype.UUID) ([]db.ListIdentitiesInClientGroupRow, error) {
	rows, err := h.q.ListIdentitiesInClientGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []db.ListIdentitiesInClientGroupRow{}
	}
	return rows, nil
}

func (h *GroupService) addMember(ctx context.Context, groupID, identityID pgtype.UUID) error {
	return h.q.AddIdentityToClientGroup(ctx, db.AddIdentityToClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	})
}

func (h *GroupService) removeMember(ctx context.Context, groupID, identityID pgtype.UUID) error {
	return h.q.RemoveIdentityFromClientGroup(ctx, db.RemoveIdentityFromClientGroupParams{
		IdentityID: identityID,
		GroupID:    groupID,
	})
}

func (h *GroupService) listRoles(ctx context.Context, groupID pgtype.UUID) ([]db.Oauth2ClientRole, error) {
	rows, err := h.q.ListRolesForClientGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []db.Oauth2ClientRole{}
	}
	return rows, nil
}

func (h *GroupService) addRole(ctx context.Context, groupID, roleID pgtype.UUID) error {
	return h.q.AssignRoleToClientGroup(ctx, db.AssignRoleToClientGroupParams{
		GroupID: groupID,
		RoleID:  roleID,
	})
}

func (h *GroupService) removeRole(ctx context.Context, groupID, roleID pgtype.UUID) error {
	return h.q.RemoveRoleFromClientGroup(ctx, db.RemoveRoleFromClientGroupParams{
		GroupID: groupID,
		RoleID:  roleID,
	})
}

// --- HTTP handlers ---

func (a *API) ListClientGroups(w http.ResponseWriter, r *http.Request) {
	clientID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	groups, err := a.groups.listForClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	group, err := a.groups.createGroup(r.Context(), clientID, req.Name, req.Description)
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
	group, err := a.groups.updateGroup(r.Context(), groupID, req.Name, req.Description)
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
	if err := a.groups.deleteGroup(r.Context(), groupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.groups.listMembers(r.Context(), groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	if err := a.groups.addMember(r.Context(), groupID, identityID); err != nil {
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
	if err := a.groups.removeMember(r.Context(), groupID, identityID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListGroupRoles(w http.ResponseWriter, r *http.Request) {
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.groups.listRoles(r.Context(), groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	if err := a.groups.addRole(r.Context(), groupID, roleID); err != nil {
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
	if err := a.groups.removeRole(r.Context(), groupID, roleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
