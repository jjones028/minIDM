package rbac

import (
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/httputil"
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
	auditor              *audit.Auditor
}

func RegisterRoleRoutes(mux *http.ServeMux, q *db.Queries, protectRead, protectWrite func(http.Handler) http.Handler, auditor *audit.Auditor) {
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
		auditor:              auditor,
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
	httputil.WriteJSON(w, roles)
}

func (a *API) ListIdentityRoles(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	roles, err := a.listIdentityRoles.Handle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, roles)
}

func (a *API) AssignRole(w http.ResponseWriter, r *http.Request) {
	identityID, err := httputil.ParseUUID(r.PathValue("id"))
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
	if err := a.assignRole.Handle(r.Context(), AssignRoleCommand{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "identity.role.assign", "identity", audit.UUIDStr(identityID), map[string]any{
		"role_id": req.RoleID,
	})
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
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "role.create", "role", audit.UUIDStr(role.ID), map[string]any{
		"name": role.Name,
	})
	httputil.WriteJSONStatus(w, http.StatusCreated, role)
}

func (a *API) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
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
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "role.update", "role", audit.UUIDStr(role.ID), map[string]any{
		"name": role.Name,
	})
	httputil.WriteJSON(w, role)
}

func (a *API) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
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
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "role.delete", "role", audit.UUIDStr(id), nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) RemoveRole(w http.ResponseWriter, r *http.Request) {
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
	if err := a.removeRole.Handle(r.Context(), RemoveRoleCommand{
		IdentityID: identityID,
		RoleID:     roleID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "identity.role.remove", "identity", audit.UUIDStr(identityID), map[string]any{
		"role_id": audit.UUIDStr(roleID),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := a.listResources.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, resources)
}

func (a *API) ListActions(w http.ResponseWriter, r *http.Request) {
	actions, err := a.listActions.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, actions)
}

func (a *API) ListRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	perms, err := a.listRolePermissions.Handle(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, perms)
}

func (a *API) AddRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("id"))
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
	resourceID, err := httputil.ParseUUID(req.ResourceID)
	if err != nil {
		http.Error(w, "invalid_resource_id", http.StatusBadRequest)
		return
	}
	actionID, err := httputil.ParseUUID(req.ActionID)
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
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "role.permission.add", "role", audit.UUIDStr(roleID), map[string]any{
		"resource_id": req.ResourceID,
		"action_id":   req.ActionID,
	})
	httputil.WriteJSONStatus(w, http.StatusCreated, perm)
}

func (a *API) RemoveRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	permID, err := httputil.ParseUUID(r.PathValue("permId"))
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
	actorID, _ := IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "role.permission.remove", "role", audit.UUIDStr(roleID), map[string]any{
		"permission_id": audit.UUIDStr(permID),
	})
	w.WriteHeader(http.StatusNoContent)
}
