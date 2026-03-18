package sqlite

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"minicloud/internal/domain"
	"minicloud/migrations"
)

// testDB creates a temporary SQLite database with migrations applied.
func testDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath, slog.Default())
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate(t *testing.T) {
	db := testDB(t)

	// Running migrations again should be idempotent.
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatalf("second migration run should be idempotent: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	db := testDB(t)
	if err := db.HealthCheck(); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// UserRepo tests
// ---------------------------------------------------------------------------

func TestUserRepo_CreateAndGet(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	user := &domain.User{
		ID:           domain.NewID(),
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hashed",
		Role:         domain.RoleAdmin,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("expected username alice, got %s", got.Username)
	}
	if got.Role != domain.RoleAdmin {
		t.Errorf("expected role admin, got %s", got.Role)
	}

	got2, err := repo.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if got2.ID != user.ID {
		t.Errorf("expected same user ID")
	}
}

func TestUserRepo_CaseInsensitiveUsername(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	user := &domain.User{
		ID: domain.NewID(), Username: "Alice", PasswordHash: "h",
		Role: domain.RoleUser, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	// Duplicate with different case should fail.
	user2 := &domain.User{
		ID: domain.NewID(), Username: "alice", PasswordHash: "h",
		Role: domain.RoleUser, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	err := repo.Create(ctx, user2)
	if err == nil {
		t.Fatal("expected uniqueness error for case-insensitive duplicate")
	}
}

func TestUserRepo_NotFound(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_Count(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	now := time.Now().UTC().Truncate(time.Second)
	repo.Create(ctx, &domain.User{
		ID: domain.NewID(), Username: "u1", PasswordHash: "h",
		Role: domain.RoleUser, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestUserRepo_Update(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	user := &domain.User{
		ID: domain.NewID(), Username: "bob", PasswordHash: "h",
		Role: domain.RoleUser, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	repo.Create(ctx, user)

	user.Email = "bob@example.com"
	user.UpdatedAt = now.Add(time.Hour)
	if err := repo.Update(ctx, user); err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	if got.Email != "bob@example.com" {
		t.Errorf("expected updated email, got %s", got.Email)
	}
}

func TestUserRepo_List(t *testing.T) {
	db := testDB(t)
	repo := db.UserRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for _, name := range []string{"charlie", "dave"} {
		repo.Create(ctx, &domain.User{
			ID: domain.NewID(), Username: name, PasswordHash: "h",
			Role: domain.RoleUser, IsActive: true, CreatedAt: now, UpdatedAt: now,
		})
	}

	users, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// ---------------------------------------------------------------------------
// FileRepo tests
// ---------------------------------------------------------------------------

func createTestUser(t *testing.T, db *DB) *domain.User {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	user := &domain.User{
		ID: domain.NewID(), Username: "testuser-" + domain.NewID()[:8],
		PasswordHash: "h", Role: domain.RoleUser, IsActive: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.UserRepo().Create(context.Background(), user); err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return user
}

func TestFileRepo_CreateAndGet(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	file := &domain.File{
		ID: domain.NewID(), OwnerID: user.ID,
		VirtualPath: "/", OriginalName: "photo.jpg",
		StorageName: domain.NewID(), Size: 1024,
		MimeType: "image/jpeg", Checksum: "abc123",
		CreatedAt: now, UpdatedAt: now,
	}

	if err := repo.Create(ctx, file); err != nil {
		t.Fatalf("create file: %v", err)
	}

	got, err := repo.GetByID(ctx, file.ID)
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	if got.OriginalName != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %s", got.OriginalName)
	}
	if got.Size != 1024 {
		t.Errorf("expected size 1024, got %d", got.Size)
	}
}

func TestFileRepo_DuplicateNameInSameDir(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	makeFile := func() *domain.File {
		return &domain.File{
			ID: domain.NewID(), OwnerID: user.ID,
			VirtualPath: "/docs/", OriginalName: "readme.txt",
			StorageName: domain.NewID(), Size: 100,
			MimeType: "text/plain", Checksum: "xyz",
			CreatedAt: now, UpdatedAt: now,
		}
	}

	if err := repo.Create(ctx, makeFile()); err != nil {
		t.Fatal(err)
	}

	// Same name in same dir for same owner should fail.
	err := repo.Create(ctx, makeFile())
	if err == nil {
		t.Fatal("expected uniqueness error for duplicate filename in same directory")
	}
}

func TestFileRepo_ListByOwner(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Files in root
	for _, name := range []string{"a.txt", "b.txt"} {
		repo.Create(ctx, &domain.File{
			ID: domain.NewID(), OwnerID: user.ID,
			VirtualPath: "/", OriginalName: name,
			StorageName: domain.NewID(), Size: 10,
			MimeType: "text/plain", Checksum: "c",
			CreatedAt: now, UpdatedAt: now,
		})
	}
	// File in subdirectory
	repo.Create(ctx, &domain.File{
		ID: domain.NewID(), OwnerID: user.ID,
		VirtualPath: "/docs/", OriginalName: "c.txt",
		StorageName: domain.NewID(), Size: 10,
		MimeType: "text/plain", Checksum: "c",
		CreatedAt: now, UpdatedAt: now,
	})

	rootFiles, err := repo.ListByOwner(ctx, user.ID, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootFiles) != 2 {
		t.Errorf("expected 2 root files, got %d", len(rootFiles))
	}

	docFiles, err := repo.ListByOwner(ctx, user.ID, "/docs/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(docFiles) != 1 {
		t.Errorf("expected 1 doc file, got %d", len(docFiles))
	}
}

func TestFileRepo_Delete(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	file := &domain.File{
		ID: domain.NewID(), OwnerID: user.ID,
		VirtualPath: "/", OriginalName: "del.txt",
		StorageName: domain.NewID(), Size: 5,
		MimeType: "text/plain", Checksum: "d",
		CreatedAt: now, UpdatedAt: now,
	}
	repo.Create(ctx, file)

	if err := repo.Delete(ctx, file.ID); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GetByID(ctx, file.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestFileRepo_SearchByOwner(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Files in different directories.
	for _, f := range []struct{ path, name string }{
		{"/", "photo.jpg"},
		{"/", "photo_backup.jpg"},
		{"/docs/", "report.pdf"},
		{"/docs/", "photo_archive.zip"},
	} {
		repo.Create(ctx, &domain.File{
			ID: domain.NewID(), OwnerID: user.ID,
			VirtualPath: f.path, OriginalName: f.name,
			StorageName: domain.NewID(), Size: 100,
			MimeType: "application/octet-stream", Checksum: "c",
			CreatedAt: now, UpdatedAt: now,
		})
	}

	// Search across all directories.
	results, err := repo.SearchByOwner(ctx, user.ID, "photo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results for 'photo', got %d", len(results))
	}

	// Search with no matches.
	results, err = repo.SearchByOwner(ctx, user.ID, "nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}

	// Exact match.
	results, err = repo.SearchByOwner(ctx, user.ID, "report.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'report.pdf', got %d", len(results))
	}
}

func TestFileRepo_CascadeDeleteOnUser(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	fileRepo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	file := &domain.File{
		ID: domain.NewID(), OwnerID: user.ID,
		VirtualPath: "/", OriginalName: "cascade.txt",
		StorageName: domain.NewID(), Size: 5,
		MimeType: "text/plain", Checksum: "e",
		CreatedAt: now, UpdatedAt: now,
	}
	fileRepo.Create(ctx, file)

	// Delete the user — files should cascade delete.
	db.sql.ExecContext(ctx, "DELETE FROM users WHERE id = ?", user.ID)

	_, err := fileRepo.GetByID(ctx, file.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected file cascade deleted, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// SessionRepo tests
// ---------------------------------------------------------------------------

func TestSessionRepo_CreateAndGet(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.SessionRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	sess := &domain.Session{
		ID:        domain.NewID(),
		UserID:    user.ID,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := repo.Create(ctx, sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := repo.GetByID(ctx, sess.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, got.UserID)
	}
}

func TestSessionRepo_ExpiredNotFound(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.SessionRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	sess := &domain.Session{
		ID:        domain.NewID(),
		UserID:    user.ID,
		ExpiresAt: now.Add(-1 * time.Hour), // already expired
		CreatedAt: now.Add(-2 * time.Hour),
	}
	repo.Create(ctx, sess)

	_, err := repo.GetByID(ctx, sess.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected expired session to be not found, got %v", err)
	}
}

func TestSessionRepo_DeleteExpired(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.SessionRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// One valid, one expired.
	repo.Create(ctx, &domain.Session{
		ID: domain.NewID(), UserID: user.ID,
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
	})
	repo.Create(ctx, &domain.Session{
		ID: domain.NewID(), UserID: user.ID,
		ExpiresAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-2 * time.Hour),
	})

	deleted, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 expired session deleted, got %d", deleted)
	}
}

func TestSessionRepo_DeleteByUserID(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.SessionRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 3; i++ {
		repo.Create(ctx, &domain.Session{
			ID: domain.NewID(), UserID: user.ID,
			ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
		})
	}

	if err := repo.DeleteByUserID(ctx, user.ID); err != nil {
		t.Fatal(err)
	}

	// Verify all sessions for that user are gone (we can't query expired ones,
	// but we can check via direct SQL).
	var count int
	db.sql.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = ?", user.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 sessions after DeleteByUserID, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// DirectoryRepo tests
// ---------------------------------------------------------------------------

func TestDirectoryRepo_CreateAndList(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.DirectoryRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Create two directories at root.
	for _, name := range []string{"docs", "photos"} {
		repo.Create(ctx, &domain.Directory{
			ID: domain.NewID(), OwnerID: user.ID,
			ParentPath: "/", Name: name, CreatedAt: now,
		})
	}

	// Create subdirectory.
	repo.Create(ctx, &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/docs/", Name: "archive", CreatedAt: now,
	})

	rootDirs, err := repo.ListByOwner(ctx, user.ID, "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(rootDirs) != 2 {
		t.Errorf("expected 2 root dirs, got %d", len(rootDirs))
	}

	docsDirs, err := repo.ListByOwner(ctx, user.ID, "/docs/")
	if err != nil {
		t.Fatal(err)
	}
	if len(docsDirs) != 1 {
		t.Errorf("expected 1 subdir in /docs/, got %d", len(docsDirs))
	}
}

func TestDirectoryRepo_DuplicateName(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.DirectoryRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	dir := &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "docs", CreatedAt: now,
	}
	if err := repo.Create(ctx, dir); err != nil {
		t.Fatal(err)
	}

	dup := &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "docs", CreatedAt: now,
	}
	err := repo.Create(ctx, dup)
	if err == nil {
		t.Fatal("expected uniqueness error for duplicate directory name")
	}
}

func TestDirectoryRepo_Exists(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.DirectoryRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Root always exists.
	exists, err := repo.Exists(ctx, user.ID, "/")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("root should always exist")
	}

	// Non-existent directory.
	exists, err = repo.Exists(ctx, user.ID, "/nope/")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected /nope/ to not exist")
	}

	// Create and check.
	repo.Create(ctx, &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "real", CreatedAt: now,
	})
	exists, err = repo.Exists(ctx, user.ID, "/real/")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected /real/ to exist after creation")
	}
}

func TestDirectoryRepo_Delete(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.DirectoryRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	dir := &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "temp", CreatedAt: now,
	}
	repo.Create(ctx, dir)

	if err := repo.Delete(ctx, dir.ID); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GetByID(ctx, dir.ID)
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Delete non-existent.
	err = repo.Delete(ctx, "nonexistent")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing dir, got %v", err)
	}
}

func TestDirectoryRepo_ListAllByOwner(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.DirectoryRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	repo.Create(ctx, &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "a", CreatedAt: now,
	})
	repo.Create(ctx, &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/", Name: "b", CreatedAt: now,
	})
	repo.Create(ctx, &domain.Directory{
		ID: domain.NewID(), OwnerID: user.ID,
		ParentPath: "/a/", Name: "nested", CreatedAt: now,
	})

	all, err := repo.ListAllByOwner(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total dirs, got %d", len(all))
	}
}

// ---------------------------------------------------------------------------
// FileRepo ListByOwnerAndMimePrefixes test
// ---------------------------------------------------------------------------

func TestFileRepo_ListByOwnerAndMimePrefixes(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Create files with various MIME types.
	for _, f := range []struct{ name, mime string }{
		{"photo.jpg", "image/jpeg"},
		{"video.mp4", "video/mp4"},
		{"song.mp3", "audio/mpeg"},
		{"readme.txt", "text/plain"},
		{"report.pdf", "application/pdf"},
		{"data.bin", "application/octet-stream"},
	} {
		repo.Create(ctx, &domain.File{
			ID: domain.NewID(), OwnerID: user.ID,
			VirtualPath: "/", OriginalName: f.name,
			StorageName: domain.NewID(), Size: 100,
			MimeType: f.mime, Checksum: "c",
			CreatedAt: now, UpdatedAt: now,
		})
	}

	// Media prefixes: image/, video/, audio/
	media, err := repo.ListByOwnerAndMimePrefixes(ctx, user.ID, []string{"image/", "video/", "audio/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(media) != 3 {
		t.Errorf("expected 3 media files, got %d", len(media))
	}

	// Document prefixes: text/, application/pdf
	docs, err := repo.ListByOwnerAndMimePrefixes(ctx, user.ID, []string{"text/", "application/pdf"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 document files, got %d", len(docs))
	}

	// Empty prefixes.
	empty, err := repo.ListByOwnerAndMimePrefixes(ctx, user.ID, []string{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 files for empty prefixes, got %d", len(empty))
	}
}

// ---------------------------------------------------------------------------
// FileRepo UpdateVirtualPath test
// ---------------------------------------------------------------------------

func TestFileRepo_UpdateVirtualPath(t *testing.T) {
	db := testDB(t)
	user := createTestUser(t, db)
	repo := db.FileRepo()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	file := &domain.File{
		ID: domain.NewID(), OwnerID: user.ID,
		VirtualPath: "/", OriginalName: "moveme.txt",
		StorageName: domain.NewID(), Size: 10,
		MimeType: "text/plain", Checksum: "c",
		CreatedAt: now, UpdatedAt: now,
	}
	repo.Create(ctx, file)

	if err := repo.UpdateVirtualPath(ctx, file.ID, "/docs/"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetByID(ctx, file.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.VirtualPath != "/docs/" {
		t.Errorf("expected virtual_path /docs/, got %s", got.VirtualPath)
	}
}
