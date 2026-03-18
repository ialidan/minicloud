package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"minicloud/internal/domain"
)

const maxJSONBodySize = 1 << 20 // 1 MiB

// respondJSON writes a success response with the data wrapped in {"data": ...}.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"data": data}) //nolint:errcheck
}

// respondError writes an error response as {"error": "message"}.
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message}) //nolint:errcheck
}

// decodeJSON reads a JSON body with a 1 MiB limit.
// Returns a user-friendly error string if decoding fails, or empty on success.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) string {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodySize)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return "invalid JSON body"
	}
	return ""
}

// userResponse is the public representation of a User (no password hash).
type userResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
