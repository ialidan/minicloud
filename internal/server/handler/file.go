package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"minicloud/internal/domain"
	"minicloud/internal/repo"
	"minicloud/internal/server/middleware"
	"minicloud/internal/service"
)

// multipartOverhead is extra space beyond the configured max upload size to
// accommodate multipart framing, headers, and boundary markers.
const multipartOverhead = 1 << 20 // 1 MiB

// fileService defines the file operations needed by FileHandler.
type fileService interface {
	Upload(ctx context.Context, ownerID, virtualPath, originalName string, r io.Reader) (*domain.File, error)
	Download(ctx context.Context, fileID string, requestor *domain.User) (*domain.File, *os.File, error)
	ListContents(ctx context.Context, ownerID, virtualPath string) (*service.DirectoryContents, error)
	ListByCategory(ctx context.Context, ownerID, category, virtualPath string, page *repo.Pagination) ([]domain.File, error)
	Search(ctx context.Context, ownerID, query string, page *repo.Pagination) ([]domain.File, error)
	FindDuplicates(ctx context.Context, ownerID string) ([]domain.File, error)
	MoveFile(ctx context.Context, fileID, destination string, requestor *domain.User) (*domain.File, error)
	Delete(ctx context.Context, fileID string, requestor *domain.User) error
	ListAllDirectories(ctx context.Context, ownerID string) ([]domain.Directory, error)
	CreateDirectory(ctx context.Context, ownerID, parentPath, name string) (*domain.Directory, error)
	DeleteDirectory(ctx context.Context, dirID string, requestor *domain.User) error
}

// FileHandler handles file upload, download, listing, and deletion.
type FileHandler struct {
	fileSvc       fileService
	maxUploadSize int64
}

func NewFileHandler(fileSvc fileService, maxUploadSize int64) *FileHandler {
	return &FileHandler{fileSvc: fileSvc, maxUploadSize: maxUploadSize}
}

// Upload handles multipart file uploads. Streams to disk without buffering
// entire files in memory.
//
//	POST /api/v1/files?path=/
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	// Enforce upload size at the HTTP level.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize+multipartOverhead)

	reader, err := r.MultipartReader()
	if err != nil {
		respondError(w, http.StatusBadRequest, "multipart form data required")
		return
	}

	virtualPath := r.URL.Query().Get("path")
	if virtualPath == "" {
		virtualPath = "/"
	}

	var uploaded []fileResponse

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			respondError(w, http.StatusBadRequest, "error reading upload")
			return
		}

		// Only process file parts named "file".
		if part.FormName() != "file" {
			part.Close()
			continue
		}

		filename := part.FileName()
		if filename == "" {
			part.Close()
			continue
		}

		file, err := h.fileSvc.Upload(r.Context(), user.ID, virtualPath, filename, part)
		part.Close()
		if err != nil {
			// Check for size exceeded.
			if errors.Is(err, domain.ErrFileTooLarge) {
				respondError(w, http.StatusRequestEntityTooLarge, err.Error())
				return
			}
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		uploaded = append(uploaded, toFileResponse(file))
	}

	if len(uploaded) == 0 {
		respondError(w, http.StatusBadRequest, "no files uploaded (use form field name 'file')")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"files": uploaded})
}

// ListDuplicates returns files that share a checksum (identical content).
//
//	GET /api/v1/files/duplicates
func (h *FileHandler) ListDuplicates(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	files, err := h.fileSvc.FindDuplicates(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not find duplicates")
		return
	}

	// Group by checksum.
	type duplicateGroup struct {
		Checksum string         `json:"checksum"`
		Size     int64          `json:"size"`
		Files    []fileResponse `json:"files"`
	}

	groups := make(map[string]*duplicateGroup)
	var order []string
	for i := range files {
		cs := files[i].Checksum
		g, ok := groups[cs]
		if !ok {
			g = &duplicateGroup{Checksum: cs, Size: files[i].Size}
			groups[cs] = g
			order = append(order, cs)
		}
		g.Files = append(g.Files, toFileResponse(&files[i]))
	}

	result := make([]duplicateGroup, 0, len(order))
	for _, cs := range order {
		result = append(result, *groups[cs])
	}

	respondJSON(w, http.StatusOK, map[string]any{"duplicates": result})
}

// List returns files and directories for the authenticated user.
// When the "q" query param is present, it searches by filename instead.
//
//	GET /api/v1/files?path=/
//	GET /api/v1/files?q=searchterm
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	page := parsePagination(r, 100, 500)

	// Search mode: return matching files, no directories.
	if q, ok := r.URL.Query()["q"]; ok {
		files, err := h.fileSvc.Search(r.Context(), user.ID, q[0], page)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to search files")
			return
		}
		fileResp := make([]fileResponse, 0, len(files))
		for i := range files {
			fileResp = append(fileResp, toFileResponse(&files[i]))
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"files":       fileResp,
			"directories": []dirResponse{},
		})
		return
	}

	// Category mode: return files filtered by MIME type category.
	if category := r.URL.Query().Get("category"); category != "" {
		virtualPath := r.URL.Query().Get("path")
		files, err := h.fileSvc.ListByCategory(r.Context(), user.ID, category, virtualPath, page)
		if err != nil {
			if strings.Contains(err.Error(), "unknown category") {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			respondError(w, http.StatusInternalServerError, "failed to list files by category")
			return
		}
		fileResp := make([]fileResponse, 0, len(files))
		for i := range files {
			fileResp = append(fileResp, toFileResponse(&files[i]))
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"files":       fileResp,
			"directories": []dirResponse{},
		})
		return
	}

	// Browse mode: return files and directories for path.
	virtualPath := r.URL.Query().Get("path")
	if virtualPath == "" {
		virtualPath = "/"
	}

	contents, err := h.fileSvc.ListContents(r.Context(), user.ID, virtualPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list files")
		return
	}

	fileResp := make([]fileResponse, 0, len(contents.Files))
	for i := range contents.Files {
		fileResp = append(fileResp, toFileResponse(&contents.Files[i]))
	}

	dirResp := make([]dirResponse, 0, len(contents.Directories))
	for i := range contents.Directories {
		dirResp = append(dirResp, toDirResponse(&contents.Directories[i]))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"files":       fileResp,
		"directories": dirResp,
	})
}

// Download serves a file's contents with proper headers.
// Supports Range requests via http.ServeContent.
//
//	GET /api/v1/files/{id}
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		respondError(w, http.StatusBadRequest, "file ID required")
		return
	}

	fileMeta, reader, err := h.fileSvc.Download(r.Context(), fileID, user)
	if err != nil {
		handleResourceError(w, err)
		return
	}
	defer reader.Close()

	// Set Content-Type and Content-Disposition before ServeContent.
	w.Header().Set("Content-Type", fileMeta.MimeType)
	w.Header().Set("Content-Disposition", contentDisposition(fileMeta.OriginalName))

	// ServeContent handles Range headers, If-Modified-Since, etc.
	http.ServeContent(w, r, fileMeta.OriginalName, fileMeta.CreatedAt, reader)
}

// MoveFile moves a file to a different directory.
//
//	PUT /api/v1/files/{id}/move
func (h *FileHandler) MoveFile(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		respondError(w, http.StatusBadRequest, "file ID required")
		return
	}

	var body struct {
		Destination string `json:"destination"`
	}
	if errMsg := decodeJSON(w, r, &body); errMsg != "" {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}

	file, err := h.fileSvc.MoveFile(r.Context(), fileID, body.Destination, user)
	if err != nil {
		handleResourceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"file": toFileResponse(file)})
}

// Delete removes a file from the database and disk.
//
//	DELETE /api/v1/files/{id}
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	fileID := chi.URLParam(r, "id")
	if fileID == "" {
		respondError(w, http.StatusBadRequest, "file ID required")
		return
	}

	if err := h.fileSvc.Delete(r.Context(), fileID, user); err != nil {
		handleResourceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "file deleted"})
}

// ListAllDirectories returns all directories for the current user.
// Used by the frontend move picker modal.
//
//	GET /api/v1/directories
func (h *FileHandler) ListAllDirectories(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	dirs, err := h.fileSvc.ListAllDirectories(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list directories")
		return
	}

	resp := make([]dirResponse, 0, len(dirs))
	for i := range dirs {
		resp = append(resp, toDirResponse(&dirs[i]))
	}

	respondJSON(w, http.StatusOK, map[string]any{"directories": resp})
}

// CreateDirectory creates a new subdirectory.
//
//	POST /api/v1/directories
func (h *FileHandler) CreateDirectory(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if errMsg := decodeJSON(w, r, &body); errMsg != "" {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}
	if body.Name == "" {
		respondError(w, http.StatusBadRequest, "directory name is required")
		return
	}

	dir, err := h.fileSvc.CreateDirectory(r.Context(), user.ID, body.Path, body.Name)
	if err != nil {
		handleResourceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"directory": toDirResponse(dir)})
}

// DeleteDirectory removes an empty directory.
//
//	DELETE /api/v1/directories/{id}
func (h *FileHandler) DeleteDirectory(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	dirID := chi.URLParam(r, "id")
	if dirID == "" {
		respondError(w, http.StatusBadRequest, "directory ID required")
		return
	}

	if err := h.fileSvc.DeleteDirectory(r.Context(), dirID, user); err != nil {
		handleResourceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "directory deleted"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type fileResponse struct {
	ID           string             `json:"id"`
	VirtualPath  string             `json:"virtual_path"`
	OriginalName string             `json:"original_name"`
	Size         int64              `json:"size"`
	MimeType     string             `json:"mime_type"`
	Checksum     string             `json:"checksum"`
	CreatedAt    string             `json:"created_at"`
	Media        *mediaMetaResponse `json:"media,omitempty"`
}

type mediaMetaResponse struct {
	TakenAt     *string  `json:"taken_at,omitempty"`
	CameraMake  string   `json:"camera_make,omitempty"`
	CameraModel string   `json:"camera_model,omitempty"`
	Width       int      `json:"width,omitempty"`
	Height      int      `json:"height,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
}

func toFileResponse(f *domain.File) fileResponse {
	resp := fileResponse{
		ID:           f.ID,
		VirtualPath:  f.VirtualPath,
		OriginalName: f.OriginalName,
		Size:         f.Size,
		MimeType:     f.MimeType,
		Checksum:     f.Checksum,
		CreatedAt:    f.CreatedAt.UTC().Format(time.RFC3339),
	}
	if f.Media != nil {
		mr := &mediaMetaResponse{
			CameraMake:  f.Media.CameraMake,
			CameraModel: f.Media.CameraModel,
			Width:       f.Media.Width,
			Height:      f.Media.Height,
			Latitude:    f.Media.Latitude,
			Longitude:   f.Media.Longitude,
		}
		if f.Media.TakenAt != nil {
			s := f.Media.TakenAt.UTC().Format(time.RFC3339)
			mr.TakenAt = &s
		}
		resp.Media = mr
	}
	return resp
}

type dirResponse struct {
	ID         string `json:"id"`
	ParentPath string `json:"parent_path"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
}

func toDirResponse(d *domain.Directory) dirResponse {
	return dirResponse{
		ID:         d.ID,
		ParentPath: d.ParentPath,
		Name:       d.Name,
		CreatedAt:  d.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func handleResourceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		respondError(w, http.StatusConflict, "already exists")
	case errors.Is(err, domain.ErrForbidden):
		respondError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, domain.ErrDirectoryNotEmpty):
		respondError(w, http.StatusConflict, "directory is not empty")
	case errors.Is(err, domain.ErrFileTooLarge):
		respondError(w, http.StatusRequestEntityTooLarge, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "operation failed")
	}
}

// contentDisposition builds a safe Content-Disposition header value.
func contentDisposition(filename string) string {
	// Escape characters that could break the header.
	safe := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r == '\n' || r == '\r' || r < 32 {
			return '_'
		}
		return r
	}, filename)
	return fmt.Sprintf(`attachment; filename="%s"`, safe)
}
