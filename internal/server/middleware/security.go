package middleware

import (
	"net/http"
	"strings"
)

// SecureHeaders sets common security headers on every response and handles
// same-origin CORS preflight requests.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Existing security headers.
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0") // modern browsers: disable legacy filter
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// Content-Security-Policy — 'unsafe-inline' for scripts covers the
		// inline theme-init snippet in index.html; for styles it covers
		// Tailwind; blob: and data: for image/video previews.
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' blob: data:; media-src 'self' blob:; connect-src 'self'")

		// HSTS — only when the connection is (or was) over TLS.
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Same-origin CORS preflight handling.
		// Since the frontend is embedded in the binary this is a single-origin
		// app.  We only respond to preflight OPTIONS requests whose Origin
		// matches the Host header (i.e. same-origin).
		if r.Method == http.MethodOptions && r.Header.Get("Origin") != "" {
			origin := r.Header.Get("Origin")
			if isSameOrigin(origin, r.Host) {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Max-Age", "300")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isSameOrigin reports whether the given Origin URL refers to the same host as
// the request Host header.  The Origin value is a full URL
// (e.g. "https://example.com") whereas Host is just host[:port].
func isSameOrigin(origin, host string) bool {
	// Strip scheme (e.g. "https://example.com" → "example.com").
	after, ok := strings.CutPrefix(origin, "://")
	if !ok {
		idx := strings.Index(origin, "://")
		if idx < 0 {
			return false
		}
		after = origin[idx+3:]
	}
	// Remove any trailing slash or path.
	if i := strings.Index(after, "/"); i >= 0 {
		after = after[:i]
	}
	return strings.EqualFold(after, host)
}
