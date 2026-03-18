package handler

import (
	"net/http"
	"strconv"

	"minicloud/internal/repo"
)

// parsePagination extracts limit and offset query parameters from the request.
// Returns nil if neither parameter is present (meaning "return all").
// Applies defaultLimit when limit is missing and caps at maxLimit.
func parsePagination(r *http.Request, defaultLimit, maxLimit int) *repo.Pagination {
	q := r.URL.Query()
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")

	// If neither param is provided, no pagination.
	if limitStr == "" && offsetStr == "" {
		return nil
	}

	limit := defaultLimit
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	return &repo.Pagination{Limit: limit, Offset: offset}
}
