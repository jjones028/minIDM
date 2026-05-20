package rbac

import (
	"context"
	"crypto/sha256"
	"fmt"
	db "minIDM/db/sqlc"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

const sessionCookie = "session"

type contextKey string

const identityKey contextKey = "identity_id"

func IdentityFromContext(ctx context.Context) (pgtype.UUID, bool) {
	id, ok := ctx.Value(identityKey).(pgtype.UUID)
	return id, ok
}

// Authenticate validates the session cookie and injects the identity ID into the request context.
func Authenticate(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			h := sha256.Sum256([]byte(cookie.Value))
			session, err := queries.GetSessionByToken(r.Context(), fmt.Sprintf("%x", h[:]))
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ident, err := queries.GetIdentityByID(r.Context(), session.IdentityID)
			if err != nil || !ident.IsEnabled {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), identityKey, session.IdentityID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Require checks that the authenticated identity has the given resource+action permission.
func Require(resource, action string, queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identityID, ok := IdentityFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowed, err := queries.IdentityHasPermission(r.Context(), db.IdentityHasPermissionParams{
				IdentityID: identityID,
				Name:       resource,
				Name_2:     action,
			})
			if err != nil || !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

