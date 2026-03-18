package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"minicloud/internal/domain"
)

const dirColumns = `id, owner_id, parent_path, name, created_at`

// DirectoryRepo implements repo.DirectoryRepo using SQLite.
type DirectoryRepo struct {
	db *sql.DB
}

func scanDirectory(scanner interface{ Scan(...any) error }) (domain.Directory, error) {
	var d domain.Directory
	var createdAt string
	err := scanner.Scan(&d.ID, &d.OwnerID, &d.ParentPath, &d.Name, &createdAt)
	if err != nil {
		return d, err
	}
	d.CreatedAt = parseTime(createdAt)
	return d, nil
}

func scanDirectories(rows *sql.Rows) ([]domain.Directory, error) {
	var dirs []domain.Directory
	for rows.Next() {
		d, err := scanDirectory(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning directory: %w", err)
		}
		dirs = append(dirs, d)
	}
	return dirs, rows.Err()
}

func (r *DirectoryRepo) Create(ctx context.Context, dir *domain.Directory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO directories (id, owner_id, parent_path, name, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		dir.ID, dir.OwnerID, dir.ParentPath, dir.Name,
		formatTime(dir.CreatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("directory %q in %q: %w", dir.Name, dir.ParentPath, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating directory: %w", err)
	}
	return nil
}

func (r *DirectoryRepo) GetByID(ctx context.Context, id string) (*domain.Directory, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+dirColumns+` FROM directories WHERE id = ?`, id,
	)
	d, err := scanDirectory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting directory: %w", err)
	}
	return &d, nil
}

func (r *DirectoryRepo) ListByOwner(ctx context.Context, ownerID string, parentPath string) ([]domain.Directory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+dirColumns+`
		 FROM directories WHERE owner_id = ? AND parent_path = ?
		 ORDER BY name`,
		ownerID, parentPath,
	)
	if err != nil {
		return nil, fmt.Errorf("listing directories: %w", err)
	}
	defer rows.Close()
	return scanDirectories(rows)
}

// ListAllByOwner returns every directory for a user (for the move picker).
func (r *DirectoryRepo) ListAllByOwner(ctx context.Context, ownerID string) ([]domain.Directory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+dirColumns+`
		 FROM directories WHERE owner_id = ?
		 ORDER BY parent_path, name`,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing all directories: %w", err)
	}
	defer rows.Close()
	return scanDirectories(rows)
}

func (r *DirectoryRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM directories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting directory: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Exists checks whether a directory exists at the given full path for an owner.
// fullPath is e.g. "/photos/" — decomposed into parent_path="/" and name="photos".
// Root "/" always exists.
func (r *DirectoryRepo) Exists(ctx context.Context, ownerID string, fullPath string) (bool, error) {
	if fullPath == "/" {
		return true, nil
	}
	trimmed := strings.TrimSuffix(fullPath, "/")
	lastSlash := strings.LastIndex(trimmed, "/")
	parentPath := trimmed[:lastSlash+1]
	name := trimmed[lastSlash+1:]

	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM directories WHERE owner_id = ? AND parent_path = ? AND name = ?`,
		ownerID, parentPath, name,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking directory existence: %w", err)
	}
	return count > 0, nil
}
