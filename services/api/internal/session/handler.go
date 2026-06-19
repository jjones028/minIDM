package session

import (
	"encoding/json"
	"errors"
	"net/http"

	db "minIDM/db/sqlc"
	"minIDM/internal/audit"
)

const cookieName = "session"

type API struct {
	svc           *Service
	secureCookies bool
	auditor       *audit.Auditor
}

func Register(mux *http.ServeMux, queries *db.Queries, secureCookies bool, auditor *audit.Auditor) {
	api := &API{
		svc:           NewService(queries),
		secureCookies: secureCookies,
		auditor:       auditor,
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

	result, err := a.svc.Login(r.Context(), req.Email, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "invalid_credentials", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrAccountNotActive) {
		http.Error(w, "account_not_active", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a.auditor.Log(r.Context(), result.IdentityID, "session.login", "session", "", map[string]any{"email": req.Email})

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		if result, _ := a.svc.Logout(r.Context(), cookie.Value); result != nil {
			a.auditor.Log(r.Context(), result.IdentityID, "session.logout", "session", "", nil)
		}
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
