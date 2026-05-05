package session

import (
	"encoding/json"
	"errors"
	db "minIDM/db/sqlc"
	"net/http"
)

const cookieName = "session"

type API struct {
	login         *LoginHandler
	q             *db.Queries
	secureCookies bool
}

func Register(mux *http.ServeMux, queries *db.Queries, secureCookies bool) {
	api := &API{
		login:         NewLoginHandler(queries),
		q:             queries,
		secureCookies: secureCookies,
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

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies, // set SECURE_COOKIES=true in production (requires HTTPS)
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		_ = a.q.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
