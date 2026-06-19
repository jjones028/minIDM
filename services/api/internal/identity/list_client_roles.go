package identity

import (
	"net/http"

	"minIDM/internal/httputil"

	"github.com/jackc/pgx/v5/pgtype"
)

type clientRoleAssignment struct {
	RoleID      pgtype.UUID `json:"role_id"`
	RoleName    string      `json:"role_name"`
	Description pgtype.Text `json:"description"`
	ClientDBID  pgtype.UUID `json:"client_db_id"`
	ClientName  string      `json:"client_name"`
	AppClientID string      `json:"app_client_id"`
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
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.svc.ListClientRoles(r.Context(), id)
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
	httputil.WriteJSON(w, out)
}

func (a *API) RemoveIdentityClientRole(w http.ResponseWriter, r *http.Request) {
	identityID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roleID, err := httputil.ParseUUID(r.PathValue("roleId"))
	if err != nil {
		http.Error(w, "invalid_role_id", http.StatusBadRequest)
		return
	}
	if err := a.svc.RemoveClientRole(r.Context(), identityID, roleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListIdentityClientGroups(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	rows, err := a.svc.ListClientGroups(r.Context(), id)
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
	httputil.WriteJSON(w, out)
}

func (a *API) RemoveIdentityClientGroup(w http.ResponseWriter, r *http.Request) {
	identityID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	groupID, err := httputil.ParseUUID(r.PathValue("groupId"))
	if err != nil {
		http.Error(w, "invalid_group_id", http.StatusBadRequest)
		return
	}
	if err := a.svc.RemoveClientGroup(r.Context(), identityID, groupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
