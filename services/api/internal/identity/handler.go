package identity

import (
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
	"minIDM/internal/rbac"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Register(mux *http.ServeMux, queries *db.Queries, protectRead, protectWrite func(http.Handler) http.Handler, auditor *audit.Auditor) {
	api := &API{
		addRegistration:       NewAddRegistrationHandler(queries),
		listIdentities:        NewListIdentitiesHandler(queries),
		getIdentity:           NewGetIdentityHandler(queries),
		listIdentitySessions:  NewListIdentitySessionsHandler(queries),
		revokeIdentitySession: NewRevokeIdentitySessionHandler(queries),
		resetPassword:         NewResetPasswordHandler(queries),
		setEnabled:            NewSetEnabledHandler(queries),
		auditor:               auditor,
	}
	api.RegisterRoutes(mux, protectRead, protectWrite)
}

type API struct {
	addRegistration       *AddRegistrationHandler
	listIdentities        *ListIdentitiesHandler
	getIdentity           *GetIdentityHandler
	listIdentitySessions  *ListIdentitySessionsHandler
	revokeIdentitySession *RevokeIdentitySessionHandler
	resetPassword         *ResetPasswordHandler
	setEnabled            *SetEnabledHandler
	auditor               *audit.Auditor
}

func (a *API) RegisterRoutes(mux *http.ServeMux, protectRead, protectWrite func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/register", a.Register)
	mux.Handle("GET /api/identities", protectRead(http.HandlerFunc(a.List)))
	mux.Handle("GET /api/identities/{id}", protectRead(http.HandlerFunc(a.Get)))
	mux.Handle("GET /api/identities/{id}/sessions", protectRead(http.HandlerFunc(a.ListSessions)))
	mux.Handle("DELETE /api/identities/{id}/sessions/{handle}", protectWrite(http.HandlerFunc(a.RevokeSession)))
	mux.Handle("POST /api/identities/{id}/reset-password", protectWrite(http.HandlerFunc(a.ResetPassword)))
	mux.Handle("PATCH /api/identities/{id}/enabled", protectWrite(http.HandlerFunc(a.SetEnabled)))
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	identities, err := a.listIdentities.Handle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(identities)
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	ident, err := a.getIdentity.Handle(r.Context(), id)
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *API) ListSessions(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	sessions, err := a.listIdentitySessions.Handle(r.Context(), id)
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
			Handle:    sessionHandle(s.Token),
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (a *API) RevokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid_id", http.StatusBadRequest)
		return
	}
	handle := r.PathValue("handle")
	if handle == "" {
		http.Error(w, "missing_handle", http.StatusBadRequest)
		return
	}
	if err := a.revokeIdentitySession.Handle(r.Context(), RevokeIdentitySessionCommand{
		IdentityID: id,
		Handle:     handle,
	}); err != nil {
		if err.Error() == "session not found" {
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
	result, err := a.addRegistration.Handle(r.Context(), AddRegistrationCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.auditor.Log(r.Context(), pgtype.UUID{}, "identity.register", "identity", audit.UUIDStr(result.ID), map[string]any{
		"email": req.Email,
	})
	w.WriteHeader(http.StatusCreated)
}

func (a *API) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
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
	if err := a.resetPassword.Handle(r.Context(), ResetPasswordCommand{
		IdentityID:  id,
		NewPassword: req.Password,
	}); err != nil {
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
	id, err := parseUUID(r.PathValue("id"))
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
	row, err := a.setEnabled.Handle(r.Context(), SetEnabledCommand{
		IdentityID: id,
		Enabled:    req.Enabled,
	})
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":         row.ID,
		"is_enabled": row.IsEnabled,
		"updated_at": row.UpdatedAt,
	})
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}
