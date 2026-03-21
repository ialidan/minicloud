package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns middleware that adds a deadline to the request context.
// If the handler does not complete within the duration, the context is
// cancelled (but the response is NOT hijacked — the handler must check
// ctx.Err() or ctx.Done() to abort gracefully).
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
