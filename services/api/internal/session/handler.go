package session

import (
	"encoding/json"
	"errors"
	db "minIDM/db/sqlc"
	"net/http"
	"strings"
)

type API struct {
	login *LoginHandler
	q     *db.Queries
}

func Register(mux *http.ServeMux, queries *db.Queries) {
	api := &API{
		login: NewLoginHandler(queries),
		q:     queries,
	}
	mux.HandleFunc("POST /api/login", api.Login)
	mux.HandleFunc("DELETE /api/session", api.Logout)
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	result, err := a.login.Handle(r.Context(), LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "invalid_credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
	})
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if ok && token != "" {
		_ = a.q.DeleteSession(r.Context(), token)
	}
	w.WriteHeader(http.StatusNoContent)
}
