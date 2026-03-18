// Package middleware provides HTTP middleware for the minicloud server.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey contextKey = "request_id"

	headerRequestID = "X-Request-ID"
)

// RequestID ensures every request has a unique ID. If the incoming request
// already carries an X-Request-ID header it is reused; otherwise a new
// 16-character hex ID is generated from crypto/rand.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = generateID()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		w.Header().Set(headerRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// generateID produces a 16-char hex string (8 random bytes).
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand never errors on supported platforms
	return hex.EncodeToString(b)
}
