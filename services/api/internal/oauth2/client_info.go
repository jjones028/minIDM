package oauth2

import (
	"encoding/json"
	"net/http"

	db "minIDM/db/sqlc"
)

// ClientInfoHandler serves GET /api/oauth2/client-info?client_id=X.
// Public endpoint — returns only non-sensitive display info for the consent page.
type ClientInfoHandler struct {
	q *db.Queries
}

func NewClientInfoHandler(q *db.Queries) *ClientInfoHandler {
	return &ClientInfoHandler{q: q}
}

func (h *ClientInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}
	info, err := h.q.GetOAuth2ClientPublicInfo(r.Context(), clientID)
	if err != nil {
		http.Error(w, "invalid_client", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":         info.Name,
		"description":  info.Description,
		"scopes":       info.Scopes,
		"auto_consent": info.AutoConsent,
	})
}
