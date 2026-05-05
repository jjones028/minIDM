package rbac

import (
	"context"
	db "minIDM/db/sqlc"
)

// Queries

type ListResourcesHandler struct {
	q *db.Queries
}

func NewListResourcesHandler(q *db.Queries) *ListResourcesHandler {
	return &ListResourcesHandler{q: q}
}

func (h *ListResourcesHandler) Handle(ctx context.Context) ([]db.Resource, error) {
	return h.q.ListResources(ctx)
}

type ListActionsHandler struct {
	q *db.Queries
}

func NewListActionsHandler(q *db.Queries) *ListActionsHandler {
	return &ListActionsHandler{q: q}
}

func (h *ListActionsHandler) Handle(ctx context.Context) ([]db.Action, error) {
	return h.q.ListActions(ctx)
}
