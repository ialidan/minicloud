package domain

import "time"

// File represents uploaded file metadata.
// The actual file content lives on disk under StorageName (a UUID),
// never under user-controlled names.
type File struct {
	ID           string
	OwnerID      string
	VirtualPath  string // virtual directory, e.g. "/" or "/docs/"
	OriginalName string // user-facing filename
	StorageName  string // UUID-based filename on disk
	Size         int64
	MimeType     string
	Checksum     string // SHA-256 hex digest
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Media metadata (populated from EXIF for images).
	Media *MediaMeta
}

// MediaMeta holds optional metadata extracted from media files (EXIF, etc.).
type MediaMeta struct {
	TakenAt     *time.Time // EXIF DateTimeOriginal
	CameraMake  string     // EXIF Make
	CameraModel string     // EXIF Model
	Width       int        // pixels
	Height      int        // pixels
	Latitude    *float64   // GPS decimal degrees
	Longitude   *float64   // GPS decimal degrees
}
