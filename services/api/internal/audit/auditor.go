package audit

import (
	"context"
	"encoding/json"
	db "minIDM/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Auditor writes audit log entries. Log errors are silently dropped so audit
// failures never interrupt the primary operation.
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

// UUIDStr converts a pgtype.UUID to its standard string representation.
func UUIDStr(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
