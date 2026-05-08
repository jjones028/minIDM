package identity

import (
	"encoding/json"
	"errors"
	db "minIDM/db/sqlc"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func Register(mux *http.ServeMux, queries *db.Queries, protect func(http.Handler) http.Handler) {
	api := &API{
		addRegistration:      NewAddRegistrationHandler(queries),
		listIdentities:       NewListIdentitiesHandler(queries),
		getIdentity:          NewGetIdentityHandler(queries),
		listIdentitySessions: NewListIdentitySessionsHandler(queries),
	}
	api.RegisterRoutes(mux, protect)
}

type API struct {
	addRegistration      *AddRegistrationHandler
	listIdentities       *ListIdentitiesHandler
	getIdentity          *GetIdentityHandler
	listIdentitySessions *ListIdentitySessionsHandler
}

func (a *API) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/register", a.Register)
	mux.Handle("GET /api/identities", protect(http.HandlerFunc(a.List)))
	mux.Handle("GET /api/identities/{id}", protect(http.HandlerFunc(a.Get)))
	mux.Handle("GET /api/identities/{id}/sessions", protect(http.HandlerFunc(a.ListSessions)))
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
		CreatedAt pgtype.Timestamptz `json:"created_at"`
		ExpiresAt pgtype.Timestamptz `json:"expires_at"`
	}
	result := make([]sessionInfo, len(sessions))
	for i, s := range sessions {
		result[i] = sessionInfo{CreatedAt: s.CreatedAt, ExpiresAt: s.ExpiresAt}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
	_, err := a.addRegistration.Handle(r.Context(), AddRegistrationCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(s)
	return id, err
}
