package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"minicloud/internal/domain"
	"minicloud/internal/repo"
)

// fileColumns is the canonical column list for the files table.
const fileColumns = `id, owner_id, virtual_path, original_name, storage_name,
		 size, mime_type, checksum, created_at, updated_at,
		 taken_at, camera_make, camera_model, width, height, latitude, longitude`

// FileRepo implements repo.FileRepo using SQLite.
type FileRepo struct {
	db *sql.DB
}

// scanFile scans a single file row into a domain.File.
func scanFile(scanner interface{ Scan(...any) error }) (domain.File, error) {
	var f domain.File
	var createdAt, updatedAt string
	var takenAt, cameraMake, cameraModel sql.NullString
	var width, height sql.NullInt64
	var lat, lon sql.NullFloat64

	err := scanner.Scan(&f.ID, &f.OwnerID, &f.VirtualPath, &f.OriginalName,
		&f.StorageName, &f.Size, &f.MimeType, &f.Checksum,
		&createdAt, &updatedAt,
		&takenAt, &cameraMake, &cameraModel, &width, &height, &lat, &lon)
	if err != nil {
		return f, err
	}
	f.CreatedAt = parseTime(createdAt)
	f.UpdatedAt = parseTime(updatedAt)
	f.Media = scanMediaMeta(takenAt, cameraMake, cameraModel, width, height, lat, lon)
	return f, nil
}

// scanMediaMeta builds a MediaMeta from nullable DB columns.
// Returns nil if no media columns are populated.
func scanMediaMeta(
	takenAt, cameraMake, cameraModel sql.NullString,
	width, height sql.NullInt64,
	lat, lon sql.NullFloat64,
) *domain.MediaMeta {
	if !takenAt.Valid && !cameraMake.Valid && !cameraModel.Valid &&
		!width.Valid && !height.Valid && !lat.Valid && !lon.Valid {
		return nil
	}

	meta := &domain.MediaMeta{}
	if takenAt.Valid {
		t := parseTime(takenAt.String)
		meta.TakenAt = &t
	}
	if cameraMake.Valid {
		meta.CameraMake = cameraMake.String
	}
	if cameraModel.Valid {
		meta.CameraModel = cameraModel.String
	}
	if width.Valid {
		meta.Width = int(width.Int64)
	}
	if height.Valid {
		meta.Height = int(height.Int64)
	}
	if lat.Valid {
		v := lat.Float64
		meta.Latitude = &v
	}
	if lon.Valid {
		v := lon.Float64
		meta.Longitude = &v
	}
	return meta
}

// scanFiles scans all rows into a slice of domain.File.
func scanFiles(rows *sql.Rows) ([]domain.File, error) {
	var files []domain.File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *FileRepo) Create(ctx context.Context, file *domain.File) error {
	takenAt, cameraMake, cameraModel, width, height, lat, lon := mediaMetaToArgs(file.Media)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO files (id, owner_id, virtual_path, original_name, storage_name,
		 size, mime_type, checksum, created_at, updated_at,
		 taken_at, camera_make, camera_model, width, height, latitude, longitude)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.ID, file.OwnerID, file.VirtualPath, file.OriginalName, file.StorageName,
		file.Size, file.MimeType, file.Checksum,
		formatTime(file.CreatedAt), formatTime(file.UpdatedAt),
		takenAt, cameraMake, cameraModel, width, height, lat, lon,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("file %q in %q: %w", file.OriginalName, file.VirtualPath, domain.ErrAlreadyExists)
		}
		return fmt.Errorf("creating file record: %w", err)
	}
	return nil
}

// mediaMetaToArgs converts a MediaMeta into nullable SQL arguments.
func mediaMetaToArgs(m *domain.MediaMeta) (takenAt, cameraMake, cameraModel any, width, height, lat, lon any) {
	if m == nil {
		return nil, nil, nil, nil, nil, nil, nil
	}
	if m.TakenAt != nil {
		takenAt = formatTime(*m.TakenAt)
	}
	if m.CameraMake != "" {
		cameraMake = m.CameraMake
	}
	if m.CameraModel != "" {
		cameraModel = m.CameraModel
	}
	if m.Width > 0 {
		width = m.Width
	}
	if m.Height > 0 {
		height = m.Height
	}
	if m.Latitude != nil {
		lat = *m.Latitude
	}
	if m.Longitude != nil {
		lon = *m.Longitude
	}
	return
}

func (r *FileRepo) GetByID(ctx context.Context, id string) (*domain.File, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+fileColumns+` FROM files WHERE id = ?`, id,
	)
	f, err := scanFile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting file: %w", err)
	}
	return &f, nil
}

// ListByOwner returns files belonging to ownerID in the given virtual directory.
// Pass "/" for root directory.
func (r *FileRepo) ListByOwner(ctx context.Context, ownerID string, virtualPath string, page *repo.Pagination) ([]domain.File, error) {
	query := `SELECT ` + fileColumns + `
		 FROM files WHERE owner_id = ? AND virtual_path = ?
		 ORDER BY original_name`
	args := []any{ownerID, virtualPath}

	pagSQL, pagArgs := paginationClause(page)
	query += pagSQL
	args = append(args, pagArgs...)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

// ListByOwnerAndMimePrefixes returns files belonging to ownerID whose
// mime_type starts with any of the given prefixes (e.g. "image/", "video/").
// Searches across all virtual paths.
func (r *FileRepo) ListByOwnerAndMimePrefixes(ctx context.Context, ownerID string, prefixes []string, page *repo.Pagination) ([]domain.File, error) {
	if len(prefixes) == 0 {
		return []domain.File{}, nil
	}

	// Build (mime_type LIKE ? OR mime_type LIKE ? ...) dynamically.
	clauses := make([]string, len(prefixes))
	args := make([]any, 0, len(prefixes)+1)
	args = append(args, ownerID)
	for i, p := range prefixes {
		clauses[i] = "mime_type LIKE ?"
		args = append(args, p+"%")
	}

	query := `SELECT ` + fileColumns + `
		 FROM files WHERE owner_id = ? AND (` + strings.Join(clauses, " OR ") + `)
		 ORDER BY COALESCE(taken_at, created_at) DESC, original_name`

	pagSQL, pagArgs := paginationClause(page)
	query += pagSQL
	args = append(args, pagArgs...)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing files by mime: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

// ListByOwnerAndMimePrefixesInPath returns files belonging to ownerID whose
// mime_type starts with any of the given prefixes and are in the specified
// virtual directory. This pushes the path filter into SQL instead of
// filtering in Go.
func (r *FileRepo) ListByOwnerAndMimePrefixesInPath(ctx context.Context, ownerID string, prefixes []string, virtualPath string, page *repo.Pagination) ([]domain.File, error) {
	if len(prefixes) == 0 {
		return []domain.File{}, nil
	}

	clauses := make([]string, len(prefixes))
	args := make([]any, 0, len(prefixes)+2)
	args = append(args, ownerID, virtualPath)
	for i, p := range prefixes {
		clauses[i] = "mime_type LIKE ?"
		args = append(args, p+"%")
	}

	query := `SELECT ` + fileColumns + `
		 FROM files WHERE owner_id = ? AND virtual_path = ? AND (` + strings.Join(clauses, " OR ") + `)
		 ORDER BY COALESCE(taken_at, created_at) DESC, original_name`

	pagSQL, pagArgs := paginationClause(page)
	query += pagSQL
	args = append(args, pagArgs...)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing files by mime in path: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

// SearchByOwner returns files belonging to ownerID whose original_name
// contains the query string (case-insensitive for ASCII). Searches across
// all virtual paths.
func (r *FileRepo) SearchByOwner(ctx context.Context, ownerID string, query string, page *repo.Pagination) ([]domain.File, error) {
	q := `SELECT ` + fileColumns + `
		 FROM files WHERE owner_id = ? AND original_name LIKE '%' || ? || '%'
		 ORDER BY original_name`
	args := []any{ownerID, query}

	pagSQL, pagArgs := paginationClause(page)
	q += pagSQL
	args = append(args, pagArgs...)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("searching files: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

// FindDuplicates returns all files whose checksum appears more than once
// for the given owner. Files are ordered by checksum so callers can group them.
func (r *FileRepo) FindDuplicates(ctx context.Context, ownerID string) ([]domain.File, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+fileColumns+`
		 FROM files
		 WHERE owner_id = ? AND checksum IN (
		   SELECT checksum FROM files WHERE owner_id = ?
		   GROUP BY checksum HAVING COUNT(*) > 1
		 )
		 ORDER BY checksum, original_name`,
		ownerID, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding duplicates: %w", err)
	}
	defer rows.Close()
	return scanFiles(rows)
}

func (r *FileRepo) UpdateVirtualPath(ctx context.Context, id string, newPath string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE files SET virtual_path = ?, updated_at = ? WHERE id = ?`,
		newPath, formatTime(time.Now().UTC()), id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("file already exists at destination: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("updating file path: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *FileRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM files WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
