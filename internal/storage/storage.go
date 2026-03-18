// Package storage handles raw file I/O on the local filesystem.
//
// Files are stored under a dedicated directory using UUID-based names,
// never user-supplied filenames. The directory is sharded by the first
// two characters of the UUID to avoid filesystem performance degradation
// from too many entries in a single directory.
//
// All writes are atomic: data is written to a temp file, fsynced, then
// renamed to the final path. This prevents partial/corrupt files on crash.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"minicloud/internal/domain"
)

// Store defines the interface for file storage operations.
// Implementations handle raw file I/O (local disk, S3, etc.).
type Store interface {
	Save(storageName string, r io.Reader, maxSize int64) (checksum string, written int64, err error)
	Open(storageName string) (*os.File, error)
	Delete(storageName string) error
	FilePath(storageName string) string
}

// Storage manages file I/O in a dedicated data directory.
type Storage struct {
	root string // <data_dir>/files/
	tmp  string // <data_dir>/tmp/
}

// New creates a Storage rooted at dataDir. Creates the necessary
// subdirectories (files/, tmp/) if they don't exist.
func New(dataDir string) (*Storage, error) {
	root := filepath.Join(dataDir, "files")
	tmp := filepath.Join(dataDir, "tmp")

	for _, dir := range []string{root, tmp} {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return &Storage{root: root, tmp: tmp}, nil
}

// FilePath returns the absolute path to a stored file. Useful for post-processing
// (e.g. EXIF extraction) after Save completes.
func (s *Storage) FilePath(storageName string) string {
	return s.filePath(storageName)
}

// Save writes data atomically and returns the SHA-256 checksum and byte count.
//
// Pipeline: temp file → write + hash → fsync → rename to final path.
// If any step fails, the temp file is cleaned up and no partial file remains.
func (s *Storage) Save(storageName string, r io.Reader, maxSize int64) (checksum string, written int64, err error) {
	tmpFile, err := os.CreateTemp(s.tmp, "upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Cleanup on any error path.
	defer func() {
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Limit to maxSize+1 so we can detect oversized uploads.
	limited := io.LimitReader(r, maxSize+1)

	// Compute SHA-256 while writing — single pass, no re-read.
	hash := sha256.New()
	tee := io.TeeReader(limited, hash)

	written, err = io.Copy(tmpFile, tee)
	if err != nil {
		return "", 0, fmt.Errorf("writing upload: %w", err)
	}

	if written > maxSize {
		return "", 0, fmt.Errorf("file exceeds maximum size of %d bytes: %w", maxSize, domain.ErrFileTooLarge)
	}

	// Fsync for durability — ensures data hits disk before rename.
	if err = tmpFile.Sync(); err != nil {
		return "", 0, fmt.Errorf("syncing file: %w", err)
	}
	if err = tmpFile.Close(); err != nil {
		return "", 0, fmt.Errorf("closing temp file: %w", err)
	}

	// Ensure the shard directory exists.
	finalPath := s.filePath(storageName)
	if err = os.MkdirAll(filepath.Dir(finalPath), 0750); err != nil {
		return "", 0, fmt.Errorf("creating shard dir: %w", err)
	}

	// Atomic rename (same filesystem guaranteed since tmp/ and files/ share parent).
	if err = os.Rename(tmpPath, finalPath); err != nil {
		return "", 0, fmt.Errorf("moving file to storage: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), written, nil
}

// Open returns a file handle for reading. Caller must close it.
func (s *Storage) Open(storageName string) (*os.File, error) {
	f, err := os.Open(s.filePath(storageName))
	if err != nil {
		return nil, fmt.Errorf("opening stored file: %w", err)
	}
	return f, nil
}

// Delete removes a file from storage. Idempotent — missing files are not an error.
func (s *Storage) Delete(storageName string) error {
	err := os.Remove(s.filePath(storageName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stored file: %w", err)
	}
	return nil
}

// HealthCheck verifies the storage directory is writable.
func (s *Storage) HealthCheck() error {
	f, err := os.CreateTemp(s.tmp, "healthcheck-*")
	if err != nil {
		return fmt.Errorf("storage not writable: %w", err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

// filePath maps a storage name to a sharded filesystem path.
// Example: "abcdef-1234-..." → "<root>/ab/abcdef-1234-..."
func (s *Storage) filePath(storageName string) string {
	if len(storageName) < 2 {
		return filepath.Join(s.root, storageName)
	}
	return filepath.Join(s.root, storageName[:2], storageName)
}
