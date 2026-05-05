package rbac

import (
	"encoding/json"
	"errors"
	db "minIDM/db/sqlc"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

type API struct {
	listRoles            *ListRolesHandler
	createRole           *CreateRoleHandler
	updateRole           *UpdateRoleHandler
	deleteRole           *DeleteRoleHandler
	listRolePermissions  *ListRolePermissionsHandler
	addRolePermission    *AddRolePermissionHandler
	removeRolePermission *RemoveRolePermissionHandler
	listResources        *ListResourcesHandler
	listActions          *ListActionsHandler
	listIdentityRoles    *ListIdentityRolesHandler
	assignRole           *AssignRoleHandler
	removeRole           *RemoveRoleHandler
}

func RegisterRoleRoutes(mux *http.ServeMux, q *db.Queries, protectRead, protectWrite func(http.Handler) http.Handler) {
	api := &API{
		listRoles:            NewListRolesHandler(q),
		createRole:           NewCreateRoleHandler(q),
		updateRole:           NewUpdateRoleHandler(q),
		deleteRole:           NewDeleteRoleHandler(q),
		listRolePermissions:  NewListRolePermissionsHandler(q),
		addRolePermission:    NewAddRolePermissionHandler(q),
		removeRolePermission: NewRemoveRolePermissionHandler(q),
		listResources:        NewListResourcesHandler(q),
		listActions:          NewListActionsHandler(q),
		listIdentityRoles:    NewListIdentityRolesHandler(q),
		assignRole:           NewAssignRoleHandler(q),
		removeRole:           NewRemoveRoleHandler(q),
	}

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

func (a *API) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.listRoles.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (a *API) ListIdentityRoles(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.listIdentityRoles.Handle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (a *API) AssignRole(w http.ResponseWriter, r *http.Request) {
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
	if err := a.assignRole.Handle(r.Context(), AssignRoleCommand{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) CreateRole(w http.ResponseWriter, r *http.Request) {
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
	role, err := a.createRole.Handle(r.Context(), CreateRoleCommand{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(role)
}

func (a *API) UpdateRole(w http.ResponseWriter, r *http.Request) {
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
	role, err := a.updateRole.Handle(r.Context(), UpdateRoleCommand{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

func (a *API) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	if err := a.deleteRole.Handle(r.Context(), DeleteRoleCommand{ID: id}); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrBuiltinRoleProtected) {
			http.Error(w, "builtin_role_protected", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RemoveRole(w http.ResponseWriter, r *http.Request) {
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
	if err := a.removeRole.Handle(r.Context(), RemoveRoleCommand{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := a.listResources.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func (a *API) ListActions(w http.ResponseWriter, r *http.Request) {
	actions, err := a.listActions.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

func (a *API) ListRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	perms, err := a.listRolePermissions.Handle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perms)
}

func (a *API) AddRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
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
	perm, err := a.addRolePermission.Handle(r.Context(), AddRolePermissionCommand{
		RoleID:     roleID,
		ResourceID: resourceID,
		ActionID:   actionID,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrBuiltinRoleProtected) {
			http.Error(w, "builtin_role_protected", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(perm)
}

func (a *API) RemoveRolePermission(w http.ResponseWriter, r *http.Request) {
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
	if err := a.removeRolePermission.Handle(r.Context(), RemoveRolePermissionCommand{
		ID:     permID,
		RoleID: roleID,
	}); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrBuiltinRoleProtected) {
			http.Error(w, "builtin_role_protected", http.StatusForbidden)
			return
		}
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
