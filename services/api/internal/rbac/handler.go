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
	mux.Handle("POST /api/roles", protectWrite(http.HandlerFunc(api.CreateRole)))
	mux.Handle("PATCH /api/roles/{id}", protectWrite(http.HandlerFunc(api.UpdateRole)))
	mux.Handle("DELETE /api/roles/{id}", protectWrite(http.HandlerFunc(api.DeleteRole)))
	mux.Handle("GET /api/roles/{id}/permissions", protectRead(http.HandlerFunc(api.ListRolePermissions)))
	mux.Handle("POST /api/roles/{id}/permissions", protectWrite(http.HandlerFunc(api.AddRolePermission)))
	mux.Handle("DELETE /api/roles/{id}/permissions/{permId}", protectWrite(http.HandlerFunc(api.RemoveRolePermission)))
	mux.Handle("GET /api/resources", protectRead(http.HandlerFunc(api.ListResources)))
	mux.Handle("GET /api/actions", protectRead(http.HandlerFunc(api.ListActions)))
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

func (a *RoleAPI) CreateRole(w http.ResponseWriter, r *http.Request) {
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
	role, err := a.q.CreateRole(r.Context(), db.CreateRoleParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(role)
}

func (a *RoleAPI) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
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
	role, err := a.q.UpdateRole(r.Context(), db.UpdateRoleParams{
		ID:          id,
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

func (a *RoleAPI) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	role, err := a.q.GetRoleByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not_found", http.StatusNotFound)
		return
	}
	if role.IsBuiltin {
		http.Error(w, "builtin_role_protected", http.StatusForbidden)
		return
	}
	if err := a.q.DeleteRole(r.Context(), id); err != nil {
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

func (a *RoleAPI) ListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := a.q.ListResources(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func (a *RoleAPI) ListActions(w http.ResponseWriter, r *http.Request) {
	actions, err := a.q.ListActions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

func (a *RoleAPI) ListRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	perms, err := a.q.ListPermissionsForRole(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perms)
}

func (a *RoleAPI) AddRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	// Block mutations on built-in roles
	role, err := a.q.GetRoleByID(r.Context(), roleID)
	if err != nil {
		http.Error(w, "not_found", http.StatusNotFound)
		return
	}
	if role.IsBuiltin {
		http.Error(w, "builtin_role_protected", http.StatusForbidden)
		return
	}
	var req struct {
		ResourceID string `json:"resource_id"`
		ActionID   string `json:"action_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	resourceID, err := parseUUID(req.ResourceID)
	if err != nil {
		http.Error(w, "invalid_resource_id", http.StatusBadRequest)
		return
	}
	actionID, err := parseUUID(req.ActionID)
	if err != nil {
		http.Error(w, "invalid_action_id", http.StatusBadRequest)
		return
	}
	perm, err := a.q.AddPermissionToRole(r.Context(), db.AddPermissionToRoleParams{
		RoleID:     roleID,
		ResourceID: resourceID,
		ActionID:   actionID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(perm)
}

func (a *RoleAPI) RemoveRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	permID, err := parseUUID(r.PathValue("permId"))
	if err != nil {
		http.Error(w, "invalid_perm_id", http.StatusBadRequest)
		return
	}
	// Block mutations on built-in roles
	role, err := a.q.GetRoleByID(r.Context(), roleID)
	if err != nil {
		http.Error(w, "not_found", http.StatusNotFound)
		return
	}
	if role.IsBuiltin {
		http.Error(w, "builtin_role_protected", http.StatusForbidden)
		return
	}
	if err := a.q.RemovePermissionFromRole(r.Context(), db.RemovePermissionFromRoleParams{
		ID:     permID,
		RoleID: roleID,
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
