package identity

import (
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/httputil"
	"minIDM/internal/rbac"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Register(mux *http.ServeMux, queries *db.Queries, protectRead, protectWrite func(http.Handler) http.Handler, auditor *audit.Auditor, registrationEnabled bool) {
	api := &API{
		svc:     NewService(queries, registrationEnabled),
		auditor: auditor,
	}
	api.RegisterRoutes(mux, protectRead, protectWrite)
}

type API struct {
	svc     *Service
	auditor *audit.Auditor
}

func (a *API) RegisterRoutes(mux *http.ServeMux, protectRead, protectWrite func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/register", a.Register)
	mux.Handle("POST /api/identities", protectWrite(http.HandlerFunc(a.Create)))
	mux.Handle("GET /api/identities", protectRead(http.HandlerFunc(a.List)))
	mux.Handle("GET /api/identities/{id}", protectRead(http.HandlerFunc(a.Get)))
	mux.Handle("GET /api/identities/{id}/sessions", protectRead(http.HandlerFunc(a.ListSessions)))
	mux.Handle("DELETE /api/identities/{id}/sessions/{handle}", protectWrite(http.HandlerFunc(a.RevokeSession)))
	mux.Handle("POST /api/identities/{id}/reset-password", protectWrite(http.HandlerFunc(a.ResetPassword)))
	mux.Handle("PATCH /api/identities/{id}/enabled", protectWrite(http.HandlerFunc(a.SetEnabled)))
	mux.Handle("GET /api/identities/{id}/client-roles", protectRead(http.HandlerFunc(a.ListIdentityClientRoles)))
	mux.Handle("DELETE /api/identities/{id}/client-roles/{roleId}", protectWrite(http.HandlerFunc(a.RemoveIdentityClientRole)))
	mux.Handle("GET /api/identities/{id}/client-groups", protectRead(http.HandlerFunc(a.ListIdentityClientGroups)))
	mux.Handle("DELETE /api/identities/{id}/client-groups/{groupId}", protectWrite(http.HandlerFunc(a.RemoveIdentityClientGroup)))
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	identities, err := a.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, identities)
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	ident, err := a.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := struct {
		ID        pgtype.UUID        `json:"id"`
		SubjectID string             `json:"subject_id"`
		Email     string             `json:"email"`
		IsEnabled bool               `json:"is_enabled"`
		CreatedAt pgtype.Timestamptz `json:"created_at"`
		UpdatedAt pgtype.Timestamptz `json:"updated_at"`
	}{
		ID:        ident.ID,
		SubjectID: ident.SubjectID,
		Email:     ident.Email,
		IsEnabled: ident.IsEnabled,
		CreatedAt: ident.CreatedAt,
		UpdatedAt: ident.UpdatedAt,
	}
	httputil.WriteJSON(w, resp)
}

func (a *API) ListSessions(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	sessions, err := a.svc.ListSessions(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type sessionInfo struct {
		Handle    string             `json:"handle"`
		CreatedAt pgtype.Timestamptz `json:"created_at"`
		ExpiresAt pgtype.Timestamptz `json:"expires_at"`
	}
	result := make([]sessionInfo, len(sessions))
	for i, s := range sessions {
		result[i] = sessionInfo{
			Handle:    sessionHandle(s.TokenHash),
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		}
	}
	httputil.WriteJSON(w, result)
}

func (a *API) RevokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	handle := r.PathValue("handle")
	if handle == "" {
		http.Error(w, "missing_handle", http.StatusBadRequest)
		return
	}
	if err := a.svc.RevokeSession(r.Context(), id, handle); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			http.Error(w, "not_found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "identity.session.revoke", "session", handle, map[string]any{
		"identity_id": r.PathValue("id"),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password_too_short", http.StatusUnprocessableEntity)
		return
	}
	result, err := a.svc.Register(r.Context(), req.Email, req.Password)
	if errors.Is(err, ErrRegistrationDisabled) {
		http.Error(w, "registration_disabled", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.auditor.Log(r.Context(), pgtype.UUID{}, "identity.register", "identity", audit.UUIDStr(result.Identity.ID), map[string]any{
		"email": req.Email,
	})
	if result.Pending {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *API) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	result, err := a.svc.Create(r.Context(), req.Email, req.Password)
	if errors.Is(err, ErrPasswordTooShort) {
		http.Error(w, "password_too_short", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "identity.create", "identity", audit.UUIDStr(result.ID), map[string]any{
		"email": req.Email,
	})
	w.WriteHeader(http.StatusCreated)
}

func (a *API) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if err := a.svc.ResetPassword(r.Context(), id, req.Password); err != nil {
		if errors.Is(err, ErrPasswordTooShort) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	a.auditor.Log(r.Context(), actorID, "identity.password.reset", "identity", audit.UUIDStr(id), nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) SetEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	row, err := a.svc.SetEnabled(r.Context(), id, req.Enabled)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actorID, _ := rbac.IdentityFromContext(r.Context())
	action := "identity.disable"
	if req.Enabled {
		action = "identity.enable"
	}
	a.auditor.Log(r.Context(), actorID, action, "identity", audit.UUIDStr(id), nil)
	httputil.WriteJSON(w, map[string]any{
		"id":         row.ID,
		"is_enabled": row.IsEnabled,
		"updated_at": row.UpdatedAt,
	})
}
