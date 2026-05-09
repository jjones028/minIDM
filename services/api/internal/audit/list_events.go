package audit

import (
	"context"
	db "minIDM/db/sqlc"
)

type ListEventsHandler struct {
	q *db.Queries
}

func NewListEventsHandler(q *db.Queries) *ListEventsHandler {
	return &ListEventsHandler{q: q}
}

func (h *ListEventsHandler) Handle(ctx context.Context, limit, offset int32) ([]db.AuditLog, error) {
	return h.q.ListAuditLogs(ctx, db.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	})
}
