package audit

import (
	"context"
	"encoding/json"

	db "minIDM/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Auditor writes audit log entries and provides queries for listing them.
// Log errors are silently dropped so audit failures never interrupt the primary operation.
type Auditor struct {
	q *db.Queries
}

func New(q *db.Queries) *Auditor {
	return &Auditor{q: q}
}

// Log records an audit event. Pass a zero-value pgtype.UUID (Valid=false) for
// system-initiated events that have no authenticated actor.
func (a *Auditor) Log(ctx context.Context, actorID pgtype.UUID, action, resourceType, resourceID string, details map[string]any) {
	var detailsBytes []byte
	if details != nil {
		detailsBytes, _ = json.Marshal(details)
	}
	var resID pgtype.Text
	if resourceID != "" {
		resID = pgtype.Text{String: resourceID, Valid: true}
	}
	_ = a.q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resID,
		Details:      detailsBytes,
	})
}

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

func (a *Auditor) ListEvents(ctx context.Context, f ListEventsFilter) (ListEventsResult, error) {
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

	total, err := a.q.CountAuditLogsFiltered(ctx, filterParams)
	if err != nil {
		return ListEventsResult{}, err
	}

	rows, err := a.q.ListAuditLogsFiltered(ctx, db.ListAuditLogsFilteredParams{
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

func (a *Auditor) ResourceTypes(ctx context.Context) ([]string, error) {
	return a.q.ListDistinctAuditResourceTypes(ctx)
}

// UUIDStr converts a pgtype.UUID to its standard string representation.
func UUIDStr(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
