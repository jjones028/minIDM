package audit

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	db "minIDM/db/sqlc"
	"minIDM/internal/httputil"

	"github.com/jackc/pgx/v5/pgtype"
)

type Config struct {
	Queries     *db.Queries
	ProtectRead func(http.Handler) http.Handler
}

type API struct {
	svc *Auditor
}

// Register wires GET /api/audit-logs and GET /api/audit-logs/resource-types,
// and returns the Auditor for use by other packages.
func Register(mux *http.ServeMux, cfg Config) *Auditor {
	api := &API{svc: New(cfg.Queries)}
	mux.Handle("GET /api/audit-logs", cfg.ProtectRead(http.HandlerFunc(api.List)))
	mux.Handle("GET /api/audit-logs/resource-types", cfg.ProtectRead(http.HandlerFunc(api.ResourceTypes)))
	return api.svc
}

func (a *API) ResourceTypes(w http.ResponseWriter, r *http.Request) {
	types, err := a.svc.ResourceTypes(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if types == nil {
		types = []string{}
	}
	httputil.WriteJSON(w, types)
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

	result, err := a.svc.ListEvents(r.Context(), filter)
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

	httputil.WriteJSON(w, response{Total: result.Total, Logs: out})
}
