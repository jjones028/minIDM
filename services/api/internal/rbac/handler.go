package rbac

import (
	"encoding/json"
	db "minIDM/db/sqlc"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

type RoleAPI struct {
	q *db.Queries
}

func RegisterRoleRoutes(mux *http.ServeMux, q *db.Queries, protectRead, protectWrite func(http.Handler) http.Handler) {
	api := &RoleAPI{q: q}
	mux.Handle("GET /api/roles", protectRead(http.HandlerFunc(api.ListRoles)))
	mux.Handle("GET /api/identities/{id}/roles", protectRead(http.HandlerFunc(api.ListIdentityRoles)))
	mux.Handle("POST /api/identities/{id}/roles", protectWrite(http.HandlerFunc(api.AssignRole)))
	mux.Handle("DELETE /api/identities/{id}/roles/{roleId}", protectWrite(http.HandlerFunc(api.RemoveRole)))
}

func (a *RoleAPI) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.q.ListRoles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (a *RoleAPI) ListIdentityRoles(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.q.ListRolesForIdentity(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (a *RoleAPI) AssignRole(w http.ResponseWriter, r *http.Request) {
	identityID, err := parseUUID(r.PathValue("id"))
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
	roleID, err := parseUUID(req.RoleID)
	if err != nil {
		http.Error(w, "invalid_role_id", http.StatusBadRequest)
		return
	}
	if err := a.q.AssignRoleToIdentity(r.Context(), db.AssignRoleToIdentityParams{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *RoleAPI) RemoveRole(w http.ResponseWriter, r *http.Request) {
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
	if err := a.q.RemoveRoleFromIdentity(r.Context(), db.RemoveRoleFromIdentityParams{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}
