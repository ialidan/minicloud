package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// rateBucket tracks failed authentication attempts for a single IP.
type rateBucket struct {
	mu          sync.Mutex
	attempts    int
	windowStart time.Time
	lockedUntil time.Time
}

// rateLimiter holds the per-IP buckets and configuration.
type rateLimiter struct {
	buckets        sync.Map // map[string]*rateBucket
	maxAttempts    int
	windowDuration time.Duration
	lockoutDuration time.Duration
}

// NewRateLimiter returns middleware that rate-limits by client IP. After
// maxAttempts failed responses (non-2xx) within window, the IP is locked
// out for lockout. A background goroutine cleans up stale entries every
// 10 minutes; pass a cancellable context to stop it.
func NewRateLimiter(maxAttempts int, window, lockout time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		maxAttempts:     maxAttempts,
		windowDuration:  window,
		lockoutDuration: lockout,
	}

	// Background cleanup goroutine — uses a detached context so it runs
	// for the lifetime of the process. In production you'd wire this to
	// the server's root context; for simplicity we use a package-level
	// context that is effectively never cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // available for testing; in production the server stops the process
	go rl.cleanup(ctx)

	return rl.middleware
}

// NewRateLimiterWithContext is like NewRateLimiter but accepts a context
// for cancelling the background cleanup goroutine.
func NewRateLimiterWithContext(ctx context.Context, maxAttempts int, window, lockout time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		maxAttempts:     maxAttempts,
		windowDuration:  window,
		lockoutDuration: lockout,
	}

	go rl.cleanup(ctx)

	return rl.middleware
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wrote {
		sr.status = code
		sr.wrote = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if !sr.wrote {
		sr.status = http.StatusOK
		sr.wrote = true
	}
	return sr.ResponseWriter.Write(b)
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		now := time.Now()

		bucket := rl.getOrCreate(ip)
		bucket.mu.Lock()

		// Check if the IP is currently locked out.
		if now.Before(bucket.lockedUntil) {
			bucket.mu.Unlock()
			jsonError(w, "too many requests, try again later", http.StatusTooManyRequests)
			return
		}

		// Reset window if it has expired.
		if now.Sub(bucket.windowStart) > rl.windowDuration {
			bucket.attempts = 0
			bucket.windowStart = now
		}

		bucket.mu.Unlock()

		// Wrap the response writer to capture the status code.
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Only count failed attempts (non-2xx).
		if rec.status < 200 || rec.status >= 300 {
			bucket.mu.Lock()
			// Re-check window in case it expired during handler execution.
			if now.Sub(bucket.windowStart) > rl.windowDuration {
				bucket.attempts = 0
				bucket.windowStart = now
			}
			bucket.attempts++
			if bucket.attempts >= rl.maxAttempts {
				bucket.lockedUntil = now.Add(rl.lockoutDuration)
				bucket.attempts = 0
			}
			bucket.mu.Unlock()
		}
	})
}

// getOrCreate returns the rateBucket for the given IP, creating one if needed.
func (rl *rateLimiter) getOrCreate(ip string) *rateBucket {
	if v, ok := rl.buckets.Load(ip); ok {
		return v.(*rateBucket)
	}
	b := &rateBucket{windowStart: time.Now()}
	actual, _ := rl.buckets.LoadOrStore(ip, b)
	return actual.(*rateBucket)
}

// cleanup periodically removes stale buckets. It runs until ctx is done.
func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			rl.buckets.Range(func(key, value any) bool {
				b := value.(*rateBucket)
				b.mu.Lock()
				stale := now.After(b.lockedUntil) && now.Sub(b.windowStart) > rl.windowDuration
				b.mu.Unlock()
				if stale {
					rl.buckets.Delete(key)
				}
				return true
			})
		}
	}
}

// clientIP extracts the client IP from the request. chi's RealIP middleware
// already sets r.RemoteAddr to the real client IP, but it may include a port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
