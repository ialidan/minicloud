package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"minicloud/internal/domain"
)

// SessionRepo implements repo.SessionRepo using SQLite.
type SessionRepo struct {
	db *sql.DB
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		session.ID, session.UserID,
		formatTime(session.ExpiresAt), formatTime(session.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

// GetByID returns the session if it exists and has not expired.
// Expired sessions are treated as not found.
func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	var s domain.Session
	var expiresAt, createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, created_at FROM sessions
		 WHERE id = ? AND expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		id,
	).Scan(&s.ID, &s.UserID, &expiresAt, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting session: %w", err)
	}
	s.ExpiresAt = parseTime(expiresAt)
	s.CreatedAt = parseTime(createdAt)
	return &s, nil
}

func (r *SessionRepo) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("deleting user sessions: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired sessions and returns the count removed.
func (r *SessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')")
	if err != nil {
		return 0, fmt.Errorf("deleting expired sessions: %w", err)
	}
	return result.RowsAffected()
}
