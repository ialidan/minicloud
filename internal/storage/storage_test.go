package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New(%s): %v", dir, err)
	}
	return s
}

func TestNew_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, sub := range []string{s.root, s.tmp} {
		info, err := os.Stat(sub)
		if err != nil {
			t.Fatalf("directory %s not created: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", sub)
		}
	}
}

func TestSave_And_Open(t *testing.T) {
	s := tempStorage(t)
	content := "hello, minicloud!"

	checksum, size, err := s.Save("test-file-001", strings.NewReader(content), 1<<20)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}

	// Verify checksum.
	h := sha256.Sum256([]byte(content))
	wantChecksum := hex.EncodeToString(h[:])
	if checksum != wantChecksum {
		t.Errorf("checksum = %s, want %s", checksum, wantChecksum)
	}

	// Verify file can be opened and read back.
	f, err := s.Open("test-file-001")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, _ := f.Read(buf)
	if string(buf[:n]) != content {
		t.Errorf("content = %q, want %q", string(buf[:n]), content)
	}
}

func TestSave_Sharding(t *testing.T) {
	s := tempStorage(t)
	name := "abcdef-1234-5678"

	_, _, err := s.Save(name, strings.NewReader("data"), 1<<20)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File should be at <root>/ab/<name>.
	expected := filepath.Join(s.root, "ab", name)
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected sharded path %s to exist: %v", expected, err)
	}
}

func TestSave_ExceedsMaxSize(t *testing.T) {
	s := tempStorage(t)
	content := strings.Repeat("x", 1000)

	_, _, err := s.Save("too-big", strings.NewReader(content), 100)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error = %v, want 'exceeds maximum size'", err)
	}

	// Temp file should be cleaned up.
	entries, _ := os.ReadDir(s.tmp)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "upload-") {
			t.Errorf("temp file %s not cleaned up", e.Name())
		}
	}
}

func TestSave_AtomicNoPartialFile(t *testing.T) {
	s := tempStorage(t)

	// Save a file that exceeds limit — final path should not exist.
	_, _, _ = s.Save("partial-test", strings.NewReader(strings.Repeat("x", 200)), 50)

	finalPath := s.filePath("partial-test")
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Errorf("partial file should not exist at %s", finalPath)
	}
}

func TestDelete(t *testing.T) {
	s := tempStorage(t)

	_, _, err := s.Save("delete-me", strings.NewReader("bye"), 1<<20)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Delete("delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// File should no longer be openable.
	_, err = s.Open("delete-me")
	if err == nil {
		t.Error("expected error opening deleted file")
	}
}

func TestDelete_Idempotent(t *testing.T) {
	s := tempStorage(t)

	// Deleting a non-existent file should not error.
	if err := s.Delete("does-not-exist"); err != nil {
		t.Errorf("Delete non-existent: %v", err)
	}
}

func TestOpen_NotFound(t *testing.T) {
	s := tempStorage(t)

	_, err := s.Open("nonexistent")
	if err == nil {
		t.Error("expected error opening nonexistent file")
	}
}

func TestHealthCheck(t *testing.T) {
	s := tempStorage(t)

	if err := s.HealthCheck(); err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
}

func TestFilePath_ShortName(t *testing.T) {
	s := tempStorage(t)

	// Names shorter than 2 chars shouldn't panic.
	p := s.filePath("a")
	if !strings.Contains(p, "a") {
		t.Errorf("filePath(a) = %s, expected to contain 'a'", p)
	}
}
