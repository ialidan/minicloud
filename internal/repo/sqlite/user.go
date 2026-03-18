package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"minicloud/internal/domain"
	"minicloud/internal/repo"
)

// UserRepo implements repo.UserRepo using SQLite.
type UserRepo struct {
	db *sql.DB
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, role, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Role, user.IsActive,
		formatTime(user.CreatedAt), formatTime(user.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("username %q: %w", user.Username, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanRow(r.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, is_active, created_at, updated_at
		 FROM users WHERE id = ?`, id))
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.scanRow(r.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash, role, is_active, created_at, updated_at
		 FROM users WHERE username = ?`, username))
}

func (r *UserRepo) List(ctx context.Context, page *repo.Pagination) ([]domain.User, error) {
	query := `SELECT id, username, email, password_hash, role, is_active, created_at, updated_at
		 FROM users ORDER BY created_at`
	var args []any

	pagSQL, pagArgs := paginationClause(page)
	query += pagSQL
	args = append(args, pagArgs...)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var createdAt, updatedAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.Role, &u.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		u.CreatedAt = parseTime(createdAt)
		u.UpdatedAt = parseTime(updatedAt)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET username = ?, email = ?, password_hash = ?, role = ?,
		 is_active = ?, updated_at = ? WHERE id = ?`,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.IsActive, formatTime(user.UpdatedAt), user.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("username %q: %w", user.Username, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("updating user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *UserRepo) CountByRoleActive(ctx context.Context, role string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = ? AND is_active = 1", role).Scan(&count)
	return count, err
}

func (r *UserRepo) scanRow(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var createdAt, updatedAt string
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.Role, &u.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	u.CreatedAt = parseTime(createdAt)
	u.UpdatedAt = parseTime(updatedAt)
	return &u, nil
}
