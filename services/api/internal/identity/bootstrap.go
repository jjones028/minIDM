package identity

import (
	"context"
	"fmt"
	db "minIDM/db/sqlc"
)

// Bootstrap creates an admin identity if no identities exist yet.
// It is a no-op when called on an already-populated database.
func Bootstrap(ctx context.Context, q *db.Queries, email, password string) error {
	count, err := q.CountIdentities(ctx)
	if err != nil {
		return fmt.Errorf("checking identities: %w", err)
	}
	if count > 0 {
		return nil
	}

	h := NewAddRegistrationHandler(q)
	if _, err := h.Handle(ctx, AddRegistrationCommand{Email: email, Password: password}); err != nil {
		return fmt.Errorf("creating bootstrap admin: %w", err)
	}
	fmt.Printf("Bootstrap admin created: %s\n", email)
	return nil
}
