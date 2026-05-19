package audit

import (
	"encoding/json"
	db "minIDM/db/sqlc"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type API struct {
	auditor    *Auditor
	listEvents *ListEventsHandler
}

// Register wires GET /api/audit-logs and GET /api/audit-logs/resource-types,
// and returns the Auditor for use by other packages.
func Register(mux *http.ServeMux, q *db.Queries, protectRead func(http.Handler) http.Handler) *Auditor {
	api := &API{
		auditor:    New(q),
		listEvents: NewListEventsHandler(q),
	}
	mux.Handle("GET /api/audit-logs", protectRead(http.HandlerFunc(api.List)))
	mux.Handle("GET /api/audit-logs/resource-types", protectRead(http.HandlerFunc(api.ResourceTypes)))
	return api.auditor
}

func (a *API) ResourceTypes(w http.ResponseWriter, r *http.Request) {
	types, err := a.auditor.q.ListDistinctAuditResourceTypes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if types == nil {
		types = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func (a *API) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := int32(50)
	offset := int32(0)
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	filter := ListEventsFilter{
		ResourceType: q.Get("resource_type"),
		ActionPrefix: q.Get("action"),
		Limit:        limit,
		Offset:       offset,
	}

	if v := q.Get("actor_id"); v != "" {
		var uid pgtype.UUID
		if err := uid.Scan(v); err == nil {
			filter.ActorID = uid
		}
	}
	if v := q.Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Since = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if v := q.Get("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Until = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	result, err := a.listEvents.Handle(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type logEntry struct {
		ID           interface{}     `json:"id"`
		ActorID      interface{}     `json:"actor_id"`
		ActorEmail   interface{}     `json:"actor_email"`
		Action       string          `json:"action"`
		ResourceType string          `json:"resource_type"`
		ResourceID   interface{}     `json:"resource_id"`
		Details      json.RawMessage `json:"details"`
		CreatedAt    interface{}     `json:"created_at"`
	}
	type response struct {
		Total int64      `json:"total"`
		Logs  []logEntry `json:"logs"`
	}

	out := make([]logEntry, len(result.Rows))
	for i, e := range result.Rows {
		details := json.RawMessage("null")
		if len(e.Details) > 0 {
			details = json.RawMessage(e.Details)
		}
		var actorEmail interface{}
		if e.ActorEmail.Valid {
			actorEmail = e.ActorEmail.String
		}
		out[i] = logEntry{
			ID:           e.ID,
			ActorID:      e.ActorID,
			ActorEmail:   actorEmail,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			Details:      details,
			CreatedAt:    e.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{Total: result.Total, Logs: out})
}
