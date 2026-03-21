package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"minicloud/internal/domain"
	"minicloud/internal/media"
	"minicloud/internal/repo"
	"minicloud/internal/storage"
)

// FileService handles file upload, download, listing, and deletion.
type FileService struct {
	files   repo.FileRepo
	dirs    repo.DirectoryRepo
	store   storage.Store
	maxSize int64
	logger  *slog.Logger
}

// NewFileService creates a FileService.
func NewFileService(files repo.FileRepo, dirs repo.DirectoryRepo, store storage.Store, maxSize int64, logger *slog.Logger) *FileService {
	return &FileService{
		files:   files,
		dirs:    dirs,
		store:   store,
		maxSize: maxSize,
		logger:  logger,
	}
}

// Upload stores a file on disk and creates a DB record.
// The caller provides a reader (e.g. from a multipart part); the method
// streams data to disk, computes SHA-256 in a single pass, and records metadata.
func (s *FileService) Upload(ctx context.Context, ownerID, virtualPath, originalName string, r io.Reader) (*domain.File, error) {
	virtualPath = sanitizeVirtualPath(virtualPath)
	originalName = sanitizeFilename(originalName)
	if originalName == "" {
		return nil, fmt.Errorf("valid filename is required")
	}

	// MIME type from extension (more reliable than sniffing for most file types).
	ext := strings.ToLower(filepath.Ext(originalName))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = fallbackMIME(ext)
	}

	storageName := domain.NewID()

	checksum, size, err := s.store.Save(storageName, r, s.maxSize)
	if err != nil {
		return nil, fmt.Errorf("storing file: %w", err)
	}

	// Extract media metadata (EXIF) from the saved file. Best-effort: a nil
	// result simply means no metadata was found. We read from the stored file
	// rather than the upload stream because Save already consumed the reader.
	meta := s.extractMetadata(storageName, mimeType)

	now := time.Now().UTC()
	file := &domain.File{
		ID:           domain.NewID(),
		OwnerID:      ownerID,
		VirtualPath:  virtualPath,
		OriginalName: originalName,
		StorageName:  storageName,
		Size:         size,
		MimeType:     mimeType,
		Checksum:     checksum,
		CreatedAt:    now,
		UpdatedAt:    now,
		Media:        meta,
	}

	if err := s.files.Create(ctx, file); err != nil {
		// Rollback: remove the stored file if DB insert fails.
		s.store.Delete(storageName)
		return nil, fmt.Errorf("saving metadata: %w", err)
	}

	return file, nil
}

// Download returns the file metadata and an open reader.
// The requestor must own the file or be an admin.
// Caller must close the returned *os.File.
func (s *FileService) Download(ctx context.Context, fileID string, requestor *domain.User) (*domain.File, *os.File, error) {
	fileMeta, err := s.files.GetByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}

	if fileMeta.OwnerID != requestor.ID && requestor.Role != domain.RoleAdmin {
		return nil, nil, domain.ErrForbidden
	}

	f, err := s.store.Open(fileMeta.StorageName)
	if err != nil {
		return nil, nil, fmt.Errorf("opening file: %w", err)
	}

	return fileMeta, f, nil
}

// List returns files belonging to the user in a given virtual directory.
func (s *FileService) List(ctx context.Context, ownerID, virtualPath string, page *repo.Pagination) ([]domain.File, error) {
	virtualPath = sanitizeVirtualPath(virtualPath)
	return s.files.ListByOwner(ctx, ownerID, virtualPath, page)
}

// DirectoryContents holds files and directories for a given path.
type DirectoryContents struct {
	Files       []domain.File
	Directories []domain.Directory
}

// ListContents returns files and directories for a given path.
func (s *FileService) ListContents(ctx context.Context, ownerID, virtualPath string) (*DirectoryContents, error) {
	virtualPath = sanitizeVirtualPath(virtualPath)

	dirs, err := s.dirs.ListByOwner(ctx, ownerID, virtualPath)
	if err != nil {
		return nil, fmt.Errorf("listing directories: %w", err)
	}

	files, err := s.files.ListByOwner(ctx, ownerID, virtualPath, nil)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	return &DirectoryContents{
		Files:       files,
		Directories: dirs,
	}, nil
}

// ListAllDirectories returns all directories for a user, used by the move picker.
func (s *FileService) ListAllDirectories(ctx context.Context, ownerID string) ([]domain.Directory, error) {
	return s.dirs.ListAllByOwner(ctx, ownerID)
}

// categoryPrefixes maps browse categories to their MIME type prefixes.
var categoryPrefixes = map[string][]string{
	"media":     {"image/", "video/", "audio/"},
	"documents": {"text/", "application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.", "application/vnd.ms-"},
}

// ListByCategory returns files matching a category's MIME types.
// If virtualPath is empty, returns all matching files (for Level 1 folder grouping).
// If virtualPath is set, returns only matching files in that directory (Level 2).
func (s *FileService) ListByCategory(ctx context.Context, ownerID, category, virtualPath string, page *repo.Pagination) ([]domain.File, error) {
	prefixes, ok := categoryPrefixes[category]
	if !ok {
		return nil, fmt.Errorf("category %q: %w", category, domain.ErrUnknownCategory)
	}

	// When a path is specified, push the filter into SQL for efficiency.
	if virtualPath != "" {
		virtualPath = sanitizeVirtualPath(virtualPath)
		return s.files.ListByOwnerAndMimePrefixesInPath(ctx, ownerID, prefixes, virtualPath, page)
	}

	// No path filter — return everything (Level 1).
	return s.files.ListByOwnerAndMimePrefixes(ctx, ownerID, prefixes, page)
}

// Search returns files belonging to the user whose name matches the query.
// Returns empty slice for blank queries.
func (s *FileService) Search(ctx context.Context, ownerID, query string, page *repo.Pagination) ([]domain.File, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.File{}, nil
	}
	return s.files.SearchByOwner(ctx, ownerID, query, page)
}

// FindDuplicates returns files whose content (checksum) appears more than once.
func (s *FileService) FindDuplicates(ctx context.Context, ownerID string, page *repo.Pagination) ([]domain.File, error) {
	return s.files.FindDuplicates(ctx, ownerID, page)
}

// CreateDirectory creates a named subdirectory under the given parent path.
func (s *FileService) CreateDirectory(ctx context.Context, ownerID, parentPath, name string) (*domain.Directory, error) {
	parentPath = sanitizeVirtualPath(parentPath)
	name = sanitizeFilename(name)
	if name == "" {
		return nil, fmt.Errorf("valid directory name is required")
	}

	// Verify parent path exists (root always exists).
	if parentPath != "/" {
		exists, err := s.dirs.Exists(ctx, ownerID, parentPath)
		if err != nil {
			return nil, fmt.Errorf("checking parent: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("parent directory does not exist: %w", domain.ErrNotFound)
		}
	}

	dir := &domain.Directory{
		ID:         domain.NewID(),
		OwnerID:    ownerID,
		ParentPath: parentPath,
		Name:       name,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.dirs.Create(ctx, dir); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	return dir, nil
}

// MoveFile changes a file's virtual_path to a new destination directory.
// The requestor must own the file or be an admin.
func (s *FileService) MoveFile(ctx context.Context, fileID, destination string, requestor *domain.User) (*domain.File, error) {
	destination = sanitizeVirtualPath(destination)

	fileMeta, err := s.files.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if fileMeta.OwnerID != requestor.ID && requestor.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	// No-op if already there.
	if fileMeta.VirtualPath == destination {
		return fileMeta, nil
	}

	// Verify destination directory exists (root always exists).
	if destination != "/" {
		exists, err := s.dirs.Exists(ctx, fileMeta.OwnerID, destination)
		if err != nil {
			return nil, fmt.Errorf("checking destination: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("destination directory does not exist: %w", domain.ErrNotFound)
		}
	}

	if err := s.files.UpdateVirtualPath(ctx, fileID, destination); err != nil {
		return nil, fmt.Errorf("moving file: %w", err)
	}

	fileMeta.VirtualPath = destination
	return fileMeta, nil
}

// DeleteDirectory removes a directory if it's empty (no files or subdirectories).
// The requestor must own the directory or be an admin.
func (s *FileService) DeleteDirectory(ctx context.Context, dirID string, requestor *domain.User) error {
	dir, err := s.dirs.GetByID(ctx, dirID)
	if err != nil {
		return err
	}

	if dir.OwnerID != requestor.ID && requestor.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}

	fullPath := dir.ParentPath + dir.Name + "/"

	childFiles, err := s.files.ListByOwner(ctx, dir.OwnerID, fullPath, nil)
	if err != nil {
		return fmt.Errorf("checking child files: %w", err)
	}
	if len(childFiles) > 0 {
		return domain.ErrDirectoryNotEmpty
	}

	childDirs, err := s.dirs.ListByOwner(ctx, dir.OwnerID, fullPath)
	if err != nil {
		return fmt.Errorf("checking child directories: %w", err)
	}
	if len(childDirs) > 0 {
		return domain.ErrDirectoryNotEmpty
	}

	return s.dirs.Delete(ctx, dirID)
}

// Delete removes a file from the database and disk.
// The requestor must own the file or be an admin.
func (s *FileService) Delete(ctx context.Context, fileID string, requestor *domain.User) error {
	fileMeta, err := s.files.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	if fileMeta.OwnerID != requestor.ID && requestor.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}

	// Remove from DB first (authoritative), then disk (best effort).
	if err := s.files.Delete(ctx, fileID); err != nil {
		return err
	}

	s.store.Delete(fileMeta.StorageName) // orphaned files are harmless
	return nil
}

// extractMetadata opens a stored file and extracts media metadata (EXIF).
// Returns nil if the file is not a supported image or has no metadata.
func (s *FileService) extractMetadata(storageName, mimeType string) *domain.MediaMeta {
	f, err := s.store.Open(storageName)
	if err != nil {
		return nil
	}
	defer f.Close()
	meta := media.ExtractMetadata(f, mimeType)
	if meta == nil && media.IsExifCapable(mimeType) {
		s.logger.Debug("no EXIF metadata found in image", "storage_name", storageName, "mime_type", mimeType)
	}
	return meta
}

// ---------------------------------------------------------------------------
// MIME fallback for platforms where Go's mime package is incomplete
// ---------------------------------------------------------------------------

var fallbackMIMEs = map[string]string{
	// Images — Go's mime package needs /etc/mime.types on Linux; Alpine
	// doesn't ship it, so common image types must be listed here.
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".webp": "image/webp",
	".avif": "image/avif",
	".heic": "image/heic",
	".heif": "image/heif",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	// Video
	".mp4":  "video/mp4",
	".m4v":  "video/x-m4v",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".mkv":  "video/x-matroska",
	".webm": "video/webm",
	".3gp":  "video/3gpp",
	// Audio
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
}

func fallbackMIME(ext string) string {
	if m, ok := fallbackMIMEs[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

// ---------------------------------------------------------------------------
// Sanitization helpers
// ---------------------------------------------------------------------------

// sanitizeVirtualPath normalizes a virtual directory path.
// Always starts and ends with "/", resolves ".." components.
func sanitizeVirtualPath(p string) string {
	if p == "" {
		return "/"
	}
	// path.Clean (not filepath.Clean) for URL-style paths.
	p = path.Clean("/" + p)
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// sanitizeFilename extracts a safe display name from user input.
// The result is ONLY used for metadata/display — never as a filesystem path.
func sanitizeFilename(name string) string {
	// Strip directory components.
	name = filepath.Base(name)

	// Remove null bytes.
	name = strings.ReplaceAll(name, "\x00", "")

	// Reject reserved names.
	if name == "." || name == ".." || name == "" {
		return ""
	}

	// Cap length to filesystem maximum.
	if len(name) > 255 {
		name = name[:255]
	}

	return name
}
