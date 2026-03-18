// Package handler contains HTTP handlers for the minicloud API.
package handler

import (
	"encoding/json"
	"net/http"
	"sync"
)

// ReadinessChecker returns nil when a subsystem is healthy, or an error
// describing why it is not ready (e.g. database unreachable).
type ReadinessChecker func() error

// Health handles Kubernetes-style liveness and readiness probes.
// Register checkers (DB, storage) via RegisterChecker; they are all
// evaluated on every /readyz call.
type Health struct {
	mu       sync.RWMutex
	checkers map[string]ReadinessChecker
}

func NewHealth() *Health {
	return &Health{
		checkers: make(map[string]ReadinessChecker),
	}
}

// RegisterChecker adds a named readiness checker.
func (h *Health) RegisterChecker(name string, check ReadinessChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = check
}

// Liveness returns 200 if the process is running. This is intentionally
// unconditional — if the binary is alive enough to respond, it's live.
func (h *Health) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Readiness runs all registered checkers. Returns 200 when all pass,
// 503 with details when any fail.
func (h *Health) Readiness(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	failures := make(map[string]string)
	for name, check := range h.checkers {
		if err := check(); err != nil {
			failures[name] = err.Error()
		}
	}

	if len(failures) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not ready",
			"errors": failures,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
