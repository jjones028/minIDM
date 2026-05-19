package audit

import (
	"context"
	db "minIDM/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type ListEventsFilter struct {
	ResourceType string
	ActionPrefix string
	ActorID      pgtype.UUID
	Since        pgtype.Timestamptz
	Until        pgtype.Timestamptz
	Limit        int32
	Offset       int32
}

type ListEventsResult struct {
	Total int64
	Rows  []db.ListAuditLogsFilteredRow
}

type ListEventsHandler struct {
	q *db.Queries
}

func NewListEventsHandler(q *db.Queries) *ListEventsHandler {
	return &ListEventsHandler{q: q}
}

func (h *ListEventsHandler) Handle(ctx context.Context, f ListEventsFilter) (ListEventsResult, error) {
	actionLike := ""
	if f.ActionPrefix != "" {
		actionLike = f.ActionPrefix + "%"
	}

	filterParams := db.CountAuditLogsFilteredParams{
		Column1: f.ResourceType,
		Column2: actionLike,
		Column3: f.ActorID,
		Column4: f.Since,
		Column5: f.Until,
	}

	total, err := h.q.CountAuditLogsFiltered(ctx, filterParams)
	if err != nil {
		return ListEventsResult{}, err
	}

	rows, err := h.q.ListAuditLogsFiltered(ctx, db.ListAuditLogsFilteredParams{
		Column1: f.ResourceType,
		Column2: actionLike,
		Column3: f.ActorID,
		Column4: f.Since,
		Column5: f.Until,
		Limit:   f.Limit,
		Offset:  f.Offset,
	})
	if err != nil {
		return ListEventsResult{}, err
	}

	return ListEventsResult{Total: total, Rows: rows}, nil
}
