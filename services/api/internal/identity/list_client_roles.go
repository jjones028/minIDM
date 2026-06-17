package identity

import (
	"encoding/json"
	"net/http"

	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type clientRoleAssignment struct {
	RoleID       pgtype.UUID `json:"role_id"`
	RoleName     string      `json:"role_name"`
	Description  pgtype.Text `json:"description"`
	ClientDBID   pgtype.UUID `json:"client_db_id"`
	ClientName   string      `json:"client_name"`
	AppClientID  string      `json:"app_client_id"`
}

type clientGroupMembership struct {
	GroupID     pgtype.UUID `json:"group_id"`
	GroupName   string      `json:"group_name"`
	Description pgtype.Text `json:"description"`
	ClientDBID  pgtype.UUID `json:"client_db_id"`
	ClientName  string      `json:"client_name"`
	AppClientID string      `json:"app_client_id"`
}

func (a *API) ListIdentityClientRoles(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.q.ListDirectClientRolesForIdentity(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]clientRoleAssignment, len(rows))
	for i, row := range rows {
		out[i] = clientRoleAssignment{
			RoleID:      row.ID,
			RoleName:    row.Name,
			Description: row.Description,
			ClientDBID:  row.ClientID,
			ClientName:  row.ClientName,
			AppClientID: row.AppClientID,
		}
	}
	if out == nil {
		out = []clientRoleAssignment{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (a *API) RemoveIdentityClientRole(w http.ResponseWriter, r *http.Request) {
	identityID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roleID, err := parseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_role_id", http.StatusBadRequest)
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

func (a *API) ListIdentityClientGroups(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.q.ListClientGroupsForIdentity(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]clientGroupMembership, len(rows))
	for i, row := range rows {
		out[i] = clientGroupMembership{
			GroupID:     row.ID,
			GroupName:   row.Name,
			Description: row.Description,
			ClientDBID:  row.ClientID,
			ClientName:  row.ClientName,
			AppClientID: row.AppClientID,
		}
	}
	if out == nil {
		out = []clientGroupMembership{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (a *API) RemoveIdentityClientGroup(w http.ResponseWriter, r *http.Request) {
	identityID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_group_id", http.StatusBadRequest)
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
