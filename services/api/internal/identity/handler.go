package identity

import (
	"encoding/json"
	db "minIDM/db/sqlc"
	"net/http"
)

// Register initializes the identity feature and registers its routes.
// protect wraps handlers that require authentication and authorization.
func Register(mux *http.ServeMux, queries *db.Queries, protect func(http.Handler) http.Handler) {
	addRegistrationHandler := NewAddRegistrationHandler(queries)
	listIdentitiesHandler := NewListIdentitiesHandler(queries)
	api := NewAPI(addRegistrationHandler, listIdentitiesHandler)
	api.RegisterRoutes(mux, protect)
}

// API handles HTTP requests for the identity feature.
type API struct {
	addRegistration *AddRegistrationHandler
	listIdentities  *ListIdentitiesHandler
}

func NewAPI(addRegistration *AddRegistrationHandler, listIdentities *ListIdentitiesHandler) *API {
	return &API{
		addRegistration: addRegistration,
		listIdentities:  listIdentities,
	}
}

func (a *API) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/register", a.Register)
	mux.Handle("GET /api/identities", protect(http.HandlerFunc(a.List)))
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

	_, err := a.addRegistration.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
