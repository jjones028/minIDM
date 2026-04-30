package identity

import (
	"encoding/json"
	db "my-idm/db/sqlc"
	"net/http"
)

// Register initializes the identity feature and registers its routes.
func Register(mux *http.ServeMux, queries *db.Queries) {
	addRegistrationHandler := NewAddRegistrationHandler(queries)
	api := NewAPI(addRegistrationHandler)
	api.RegisterRoutes(mux)
}

// API handles HTTP requests for the identity feature.
type API struct {
	register *AddRegistrationHandler
}

func NewAPI(register *AddRegistrationHandler) *API {
	return &API{
		register: register,
	}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/register", a.Register)
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

	cmd := AddRegistrationCommand{
		Email:    req.Email,
		Password: req.Password,
	}

	_, err := a.register.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
