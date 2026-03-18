package sqlite

import (
	"log/slog"
	"strings"
	"time"

	"minicloud/internal/repo"
)

const timeFormat = time.RFC3339

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		slog.Warn("failed to parse time from database", "value", s, "error", err)
	}
	return t
}

// isUniqueViolation checks whether an error is a SQLite UNIQUE constraint failure.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// paginationClause returns a SQL fragment and args for LIMIT/OFFSET.
// If p is nil, returns empty string and no args (no pagination).
func paginationClause(p *repo.Pagination) (string, []any) {
	if p == nil {
		return "", nil
	}
	return " LIMIT ? OFFSET ?", []any{p.Limit, p.Offset}
}
