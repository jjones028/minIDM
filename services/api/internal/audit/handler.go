package audit

import (
	"encoding/json"
	db "minIDM/db/sqlc"
	"net/http"
	"strconv"
)

type API struct {
	auditor    *Auditor
	listEvents *ListEventsHandler
}

// Register wires GET /api/audit-logs and returns the Auditor for use by other packages.
func Register(mux *http.ServeMux, q *db.Queries, protectRead func(http.Handler) http.Handler) *Auditor {
	api := &API{
		auditor:    New(q),
		listEvents: NewListEventsHandler(q),
	}
	mux.Handle("GET /api/audit-logs", protectRead(http.HandlerFunc(api.List)))
	return api.auditor
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	limit := int32(100)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	events, err := a.listEvents.Handle(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type auditLogResponse struct {
		ID           interface{}      `json:"id"`
		ActorID      interface{}      `json:"actor_id"`
		Action       string           `json:"action"`
		ResourceType string           `json:"resource_type"`
		ResourceID   interface{}      `json:"resource_id"`
		Details      json.RawMessage  `json:"details"`
		CreatedAt    interface{}      `json:"created_at"`
	}

	out := make([]auditLogResponse, len(events))
	for i, e := range events {
		details := json.RawMessage("null")
		if len(e.Details) > 0 {
			details = json.RawMessage(e.Details)
		}
		out[i] = auditLogResponse{
			ID:           e.ID,
			ActorID:      e.ActorID,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			Details:      details,
			CreatedAt:    e.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
