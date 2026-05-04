package rbac

import (
	"context"
	db "minIDM/db/sqlc"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

const identityKey contextKey = "identity_id"

func IdentityFromContext(ctx context.Context) (pgtype.UUID, bool) {
	id, ok := ctx.Value(identityKey).(pgtype.UUID)
	return id, ok
}

// Authenticate validates the Bearer token and injects the identity ID into the request context.
func Authenticate(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			session, err := queries.GetSessionByToken(r.Context(), token)
			if err != nil {
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

func bearerToken(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return token
}
