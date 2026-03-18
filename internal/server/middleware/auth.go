package middleware

import (
	"context"
	"fmt"
	"net/http"

	"minicloud/internal/domain"
)

const (
	// UserKey is the context key for the authenticated user.
	UserKey contextKey = "user"
	// SessionKey is the context key for the current session.
	SessionKey contextKey = "session"

	// CookieName is the session cookie name.
	CookieName = "minicloud_session"
)

// SessionValidator validates a session ID and returns the user + session.
// Defined as an interface to avoid a circular import with the service package.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID string) (*domain.User, *domain.Session, error)
}

// RequireAuth validates the session cookie and attaches the user to context.
// Returns 401 JSON for unauthenticated requests.
func RequireAuth(validator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				jsonError(w, "authentication required", http.StatusUnauthorized)
				return
			}

			user, session, err := validator.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				jsonError(w, "invalid or expired session", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, user)
			ctx = context.WithValue(ctx, SessionKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin checks that the authenticated user has admin role.
// Must be used after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.Role != domain.RoleAdmin {
			jsonError(w, "admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext extracts the authenticated user from context.
func UserFromContext(ctx context.Context) *domain.User {
	if u, ok := ctx.Value(UserKey).(*domain.User); ok {
		return u
	}
	return nil
}

// SessionFromContext extracts the current session from context.
func SessionFromContext(ctx context.Context) *domain.Session {
	if s, ok := ctx.Value(SessionKey).(*domain.Session); ok {
		return s
	}
	return nil
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, message)
}
