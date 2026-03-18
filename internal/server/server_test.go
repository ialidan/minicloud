package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"minicloud/internal/config"
	"minicloud/internal/repo/sqlite"
	"minicloud/internal/service"
	"minicloud/internal/storage"
	"minicloud/migrations"
)

// testEnv spins up a fully wired Server backed by a temp SQLite DB and
// temp storage directory. Returns the httptest.Server and a cleanup func.
type testEnv struct {
	ts      *httptest.Server
	authSvc *service.AuthService
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()

	// Database.
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlite.Open(dbPath, slog.Default())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Storage.
	store, err := storage.New(dir)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	// Services.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authSvc := service.NewAuthService(db.UserRepo(), db.SessionRepo(), logger)
	fileSvc := service.NewFileService(db.FileRepo(), db.DirectoryRepo(), store, 10<<20, logger) // 10 MiB

	// Config.
	cfg := config.Defaults()
	cfg.Server.SecureCookies = false

	// Server.
	srv := New(cfg, logger, authSvc, fileSvc)
	ts := httptest.NewServer(srv.http.Handler)
	t.Cleanup(ts.Close)

	return &testEnv{ts: ts, authSvc: authSvc}
}

// setupAdmin creates the initial admin via setup token and returns the session cookie.
func (e *testEnv) setupAdmin(t *testing.T, username, password string) *http.Cookie {
	t.Helper()

	token, err := e.authSvc.InitSetup(context.Background())
	if err != nil {
		t.Fatalf("init setup: %v", err)
	}

	body := fmt.Sprintf(`{"token":%q,"username":%q,"password":%q}`, token, username, password)
	resp := e.post(t, "/api/v1/auth/setup", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup: status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	return e.login(t, username, password)
}

// login authenticates and returns the session cookie.
func (e *testEnv) login(t *testing.T, username, password string) *http.Cookie {
	t.Helper()

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	resp := e.post(t, "/api/v1/auth/login", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	for _, c := range resp.Cookies() {
		if c.Name == "minicloud_session" {
			return c
		}
	}
	t.Fatal("no session cookie in login response")
	return nil
}

func (e *testEnv) get(t *testing.T, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", e.ts.URL+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (e *testEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.ts.URL+path, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (e *testEnv) postAuth(t *testing.T, path, body string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", e.ts.URL+path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (e *testEnv) del(t *testing.T, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", e.ts.URL+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

func jsonData(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if data, ok := result["data"].(map[string]any); ok {
		return data
	}
	return result
}

// ---------------------------------------------------------------------------
// Health endpoint tests
// ---------------------------------------------------------------------------

func TestHealth_Liveness(t *testing.T) {
	env := newTestEnv(t)
	resp := env.get(t, "/healthz", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHealth_Readiness(t *testing.T) {
	env := newTestEnv(t)
	resp := env.get(t, "/readyz", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Auth endpoint tests
// ---------------------------------------------------------------------------

func TestAuth_CheckSetup_NeedsSetup(t *testing.T) {
	env := newTestEnv(t)
	env.authSvc.InitSetup(context.Background())

	resp := env.get(t, "/api/v1/auth/setup", nil)
	data := jsonData(t, resp)

	if data["needs_setup"] != true {
		t.Errorf("needs_setup = %v, want true", data["needs_setup"])
	}
}

func TestAuth_Setup(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.authSvc.InitSetup(context.Background())

	body := fmt.Sprintf(`{"token":%q,"username":"admin","password":"securepass123"}`, token)
	resp := env.post(t, "/api/v1/auth/setup", body)
	data := jsonData(t, resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	user := data["user"].(map[string]any)
	if user["username"] != "admin" {
		t.Errorf("username = %v, want admin", user["username"])
	}
	if user["role"] != "admin" {
		t.Errorf("role = %v, want admin", user["role"])
	}
}

func TestAuth_Setup_InvalidToken(t *testing.T) {
	env := newTestEnv(t)
	env.authSvc.InitSetup(context.Background())

	body := `{"token":"wrong","username":"admin","password":"securepass123"}`
	resp := env.post(t, "/api/v1/auth/setup", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_Setup_DoubleSetup(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.authSvc.InitSetup(context.Background())

	body := fmt.Sprintf(`{"token":%q,"username":"admin","password":"securepass123"}`, token)
	resp := env.post(t, "/api/v1/auth/setup", body)
	resp.Body.Close()

	// Second attempt should fail.
	resp2 := env.post(t, "/api/v1/auth/setup", body)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("second setup: status = %d, want 403", resp2.StatusCode)
	}
}

func TestAuth_Login(t *testing.T) {
	env := newTestEnv(t)
	env.setupAdmin(t, "admin", "securepass123")

	body := `{"username":"admin","password":"securepass123"}`
	resp := env.post(t, "/api/v1/auth/login", body)
	data := jsonData(t, resp)

	user := data["user"].(map[string]any)
	if user["username"] != "admin" {
		t.Errorf("username = %v, want admin", user["username"])
	}

	var hasCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "minicloud_session" {
			hasCookie = true
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
		}
	}
	if !hasCookie {
		t.Error("expected session cookie")
	}
}

func TestAuth_Login_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	env.setupAdmin(t, "admin", "securepass123")

	body := `{"username":"admin","password":"wrongpassword"}`
	resp := env.post(t, "/api/v1/auth/login", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_Login_MissingFields(t *testing.T) {
	env := newTestEnv(t)

	body := `{"username":"admin"}`
	resp := env.post(t, "/api/v1/auth/login", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAuth_Me(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.get(t, "/api/v1/auth/me", cookie)
	data := jsonData(t, resp)

	user := data["user"].(map[string]any)
	if user["username"] != "admin" {
		t.Errorf("username = %v, want admin", user["username"])
	}
}

func TestAuth_Me_NoAuth(t *testing.T) {
	env := newTestEnv(t)

	resp := env.get(t, "/api/v1/auth/me", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_Logout(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.postAuth(t, "/api/v1/auth/logout", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// Cookie should be deleted (MaxAge = -1).
	for _, c := range resp.Cookies() {
		if c.Name == "minicloud_session" && c.MaxAge != -1 {
			t.Error("session cookie should have MaxAge=-1 after logout")
		}
	}

	// Session should be invalid now.
	resp2 := env.get(t, "/api/v1/auth/me", cookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout: status = %d, want 401", resp2.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// File endpoint tests
// ---------------------------------------------------------------------------

func (e *testEnv) uploadFile(t *testing.T, cookie *http.Cookie, filename, content string) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte(content))
	w.Close()

	req, _ := http.NewRequest("POST", e.ts.URL+"/api/v1/files?path=/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("upload status = %d, body = %s", resp.StatusCode, body)
	}

	data := jsonData(t, resp)
	files := data["files"].([]any)
	return files[0].(map[string]any)
}

func (e *testEnv) uploadFileToPath(t *testing.T, cookie *http.Cookie, path, filename, content string) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write([]byte(content))
	w.Close()

	req, _ := http.NewRequest("POST", e.ts.URL+"/api/v1/files?path="+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("upload status = %d, body = %s", resp.StatusCode, body)
	}

	data := jsonData(t, resp)
	files := data["files"].([]any)
	return files[0].(map[string]any)
}

func TestFile_Upload(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	file := env.uploadFile(t, cookie, "hello.txt", "hello world")

	if file["original_name"] != "hello.txt" {
		t.Errorf("name = %v, want hello.txt", file["original_name"])
	}
	if file["size"] != float64(11) {
		t.Errorf("size = %v, want 11", file["size"])
	}
	if file["id"] == "" {
		t.Error("expected file ID")
	}
}

func TestFile_Upload_NoAuth(t *testing.T) {
	env := newTestEnv(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.txt")
	part.Write([]byte("data"))
	w.Close()

	req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/files", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestFile_List(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Upload two files.
	env.uploadFile(t, cookie, "a.txt", "aaa")
	env.uploadFile(t, cookie, "b.txt", "bbb")

	resp := env.get(t, "/api/v1/files?path=/", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 2 {
		t.Errorf("file count = %d, want 2", len(files))
	}
}

func TestFile_List_Empty(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.get(t, "/api/v1/files?path=/", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 0 {
		t.Errorf("file count = %d, want 0", len(files))
	}
}

func TestFile_Download(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	file := env.uploadFile(t, cookie, "readme.txt", "read me!")
	fileID := file["id"].(string)

	resp := env.get(t, "/api/v1/files/"+fileID, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download: status = %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "read me!" {
		t.Errorf("content = %q, want 'read me!'", body)
	}

	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestFile_Download_NotFound(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.get(t, "/api/v1/files/nonexistent-id", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestFile_Delete(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	file := env.uploadFile(t, cookie, "deleteme.txt", "gone")
	fileID := file["id"].(string)

	resp := env.del(t, "/api/v1/files/"+fileID, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: status = %d", resp.StatusCode)
	}

	// Verify it's gone.
	resp2 := env.get(t, "/api/v1/files/"+fileID, cookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want 404", resp2.StatusCode)
	}
}

func TestFile_Delete_NotFound(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.del(t, "/api/v1/files/nonexistent", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestFile_Search(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Upload files in different directories.
	env.uploadFile(t, cookie, "photo.jpg", "img1")
	env.uploadFile(t, cookie, "notes.txt", "text")
	env.uploadFileToPath(t, cookie, "/docs/", "photo_backup.jpg", "img2")

	// Search for "photo" — should match across directories.
	resp := env.get(t, "/api/v1/files?q=photo", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 2 {
		t.Errorf("search 'photo': file count = %d, want 2", len(files))
	}

	// Search for "notes" — single match.
	resp2 := env.get(t, "/api/v1/files?q=notes", cookie)
	data2 := jsonData(t, resp2)

	files2 := data2["files"].([]any)
	if len(files2) != 1 {
		t.Errorf("search 'notes': file count = %d, want 1", len(files2))
	}
}

func TestFile_Search_Empty(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	env.uploadFile(t, cookie, "test.txt", "data")

	// Empty query should return empty list.
	resp := env.get(t, "/api/v1/files?q=", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 0 {
		t.Errorf("empty search: file count = %d, want 0", len(files))
	}

	// No matches.
	resp2 := env.get(t, "/api/v1/files?q=nonexistent", cookie)
	data2 := jsonData(t, resp2)

	files2 := data2["files"].([]any)
	if len(files2) != 0 {
		t.Errorf("no-match search: file count = %d, want 0", len(files2))
	}
}

// ---------------------------------------------------------------------------
// Directory endpoint tests
// ---------------------------------------------------------------------------

func TestDirectory_Create(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	body := `{"path":"/","name":"photos"}`
	resp := env.postAuth(t, "/api/v1/directories", body, cookie)

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status = %d, body = %s", resp.StatusCode, b)
	}

	data := jsonData(t, resp)
	dir := data["directory"].(map[string]any)
	if dir["name"] != "photos" {
		t.Errorf("name = %v, want photos", dir["name"])
	}

	// Verify directory shows up in file listing.
	resp2 := env.get(t, "/api/v1/files?path=/", cookie)
	data2 := jsonData(t, resp2)

	dirs := data2["directories"].([]any)
	if len(dirs) != 1 {
		t.Errorf("directory count = %d, want 1", len(dirs))
	}
}

func TestDirectory_Create_Duplicate(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	body := `{"path":"/","name":"docs"}`
	resp := env.postAuth(t, "/api/v1/directories", body, cookie)
	resp.Body.Close()

	resp2 := env.postAuth(t, "/api/v1/directories", body, cookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp2.StatusCode)
	}
}

func TestDirectory_Delete_NotEmpty(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Create directory.
	body := `{"path":"/","name":"full"}`
	resp := env.postAuth(t, "/api/v1/directories", body, cookie)
	data := jsonData(t, resp)
	dirID := data["directory"].(map[string]any)["id"].(string)

	// Upload a file into it.
	env.uploadFileToPath(t, cookie, "/full/", "test.txt", "data")

	// Try to delete — should fail.
	resp2 := env.del(t, "/api/v1/directories/"+dirID, cookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp2.StatusCode)
	}
}

func TestDirectory_Delete_Empty(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	body := `{"path":"/","name":"empty"}`
	resp := env.postAuth(t, "/api/v1/directories", body, cookie)
	data := jsonData(t, resp)
	dirID := data["directory"].(map[string]any)["id"].(string)

	resp2 := env.del(t, "/api/v1/directories/"+dirID, cookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp2.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// File move tests
// ---------------------------------------------------------------------------

func TestFile_Move(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Create target directory.
	env.postAuth(t, "/api/v1/directories", `{"path":"/","name":"archive"}`, cookie).Body.Close()

	// Upload file at root.
	file := env.uploadFile(t, cookie, "moveme.txt", "data")
	fileID := file["id"].(string)

	// Move to /archive/.
	body := `{"destination":"/archive/"}`
	req, _ := http.NewRequest("PUT", env.ts.URL+"/api/v1/files/"+fileID+"/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("move status = %d, body = %s", resp.StatusCode, b)
	}

	// Verify file is no longer at root.
	resp2 := env.get(t, "/api/v1/files?path=/", cookie)
	data2 := jsonData(t, resp2)
	rootFiles := data2["files"].([]any)
	if len(rootFiles) != 0 {
		t.Errorf("root files = %d, want 0 after move", len(rootFiles))
	}

	// Verify file is in /archive/.
	resp3 := env.get(t, "/api/v1/files?path=/archive/", cookie)
	data3 := jsonData(t, resp3)
	archiveFiles := data3["files"].([]any)
	if len(archiveFiles) != 1 {
		t.Errorf("archive files = %d, want 1", len(archiveFiles))
	}
}

func TestFile_Move_NotFound(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	file := env.uploadFile(t, cookie, "stuck.txt", "data")
	fileID := file["id"].(string)

	// Move to nonexistent directory.
	body := `{"destination":"/nonexistent/"}`
	req, _ := http.NewRequest("PUT", env.ts.URL+"/api/v1/files/"+fileID+"/move", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Category browsing tests
// ---------------------------------------------------------------------------

func TestFile_List_CategoryMedia(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Upload an image and a text file.
	env.uploadFile(t, cookie, "photo.jpg", "imgdata")
	env.uploadFile(t, cookie, "notes.txt", "text")

	// Category=media should return only the image.
	resp := env.get(t, "/api/v1/files?category=media", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 1 {
		t.Errorf("media category: file count = %d, want 1", len(files))
	}
	if len(files) > 0 {
		f := files[0].(map[string]any)
		if f["original_name"] != "photo.jpg" {
			t.Errorf("expected photo.jpg, got %v", f["original_name"])
		}
	}
}

func TestFile_List_CategoryDocuments(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	env.uploadFile(t, cookie, "photo.jpg", "imgdata")
	env.uploadFile(t, cookie, "notes.txt", "text")
	env.uploadFile(t, cookie, "report.pdf", "pdfdata")

	// Category=documents should return text and pdf files.
	resp := env.get(t, "/api/v1/files?category=documents", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 2 {
		t.Errorf("documents category: file count = %d, want 2", len(files))
	}
}

func TestFile_List_CategoryWithPath(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Create a directory and upload files in different locations.
	env.postAuth(t, "/api/v1/directories", `{"path":"/","name":"photos"}`, cookie).Body.Close()

	env.uploadFile(t, cookie, "root.jpg", "imgdata")
	env.uploadFileToPath(t, cookie, "/photos/", "vacation.jpg", "imgdata2")
	env.uploadFileToPath(t, cookie, "/photos/", "notes.txt", "text")

	// Category=media with path=/photos/ should return only vacation.jpg.
	resp := env.get(t, "/api/v1/files?category=media&path=/photos/", cookie)
	data := jsonData(t, resp)

	files := data["files"].([]any)
	if len(files) != 1 {
		t.Errorf("media in /photos/: file count = %d, want 1", len(files))
	}
	if len(files) > 0 {
		f := files[0].(map[string]any)
		if f["original_name"] != "vacation.jpg" {
			t.Errorf("expected vacation.jpg, got %v", f["original_name"])
		}
	}
}

func TestFile_List_InvalidCategory(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	resp := env.get(t, "/api/v1/files?category=invalid", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Admin user management tests
// ---------------------------------------------------------------------------

func TestAdmin_CreateUser(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	body := `{"username":"newuser","password":"password123","email":"new@test.com","role":"user"}`
	resp := env.postAuth(t, "/api/v1/admin/users/", body, cookie)
	data := jsonData(t, resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	user := data["user"].(map[string]any)
	if user["username"] != "newuser" {
		t.Errorf("username = %v, want newuser", user["username"])
	}
	if user["role"] != "user" {
		t.Errorf("role = %v, want user", user["role"])
	}
}

func TestAdmin_CreateUser_NonAdmin(t *testing.T) {
	env := newTestEnv(t)
	adminCookie := env.setupAdmin(t, "admin", "securepass123")

	// Create a regular user.
	body := `{"username":"regular","password":"password123","role":"user"}`
	resp := env.postAuth(t, "/api/v1/admin/users/", body, adminCookie)
	resp.Body.Close()

	// Login as regular user.
	userCookie := env.login(t, "regular", "password123")

	// Try to create a user — should be forbidden.
	body2 := `{"username":"hacker","password":"password123"}`
	resp2 := env.postAuth(t, "/api/v1/admin/users/", body2, userCookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp2.StatusCode)
	}
}

func TestAdmin_ListUsers(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Create another user.
	body := `{"username":"user2","password":"password123","role":"user"}`
	resp := env.postAuth(t, "/api/v1/admin/users/", body, cookie)
	resp.Body.Close()

	resp2 := env.get(t, "/api/v1/admin/users/", cookie)
	data := jsonData(t, resp2)

	users := data["users"].([]any)
	if len(users) != 2 {
		t.Errorf("user count = %d, want 2", len(users))
	}
}

// ---------------------------------------------------------------------------
// SPA fallback tests
// ---------------------------------------------------------------------------

func TestSPA_ServesIndexForUnknownRoutes(t *testing.T) {
	env := newTestEnv(t)

	resp := env.get(t, "/some/nonexistent/route", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("MiniCloud")) {
		t.Error("expected index.html content for SPA fallback")
	}
}

func TestSPA_ServesStaticAssets(t *testing.T) {
	env := newTestEnv(t)

	resp := env.get(t, "/style.css", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("MiniCloud")) {
		t.Error("expected CSS content")
	}
}

// ---------------------------------------------------------------------------
// File ownership / isolation tests
// ---------------------------------------------------------------------------

func TestFile_Isolation_UserCannotAccessOtherUsersFiles(t *testing.T) {
	env := newTestEnv(t)
	adminCookie := env.setupAdmin(t, "admin", "securepass123")

	// Create a regular user.
	body := `{"username":"alice","password":"password123","role":"user"}`
	resp := env.postAuth(t, "/api/v1/admin/users/", body, adminCookie)
	resp.Body.Close()

	aliceCookie := env.login(t, "alice", "password123")

	// Admin uploads a file.
	file := env.uploadFile(t, adminCookie, "secret.txt", "admin secret")
	fileID := file["id"].(string)

	// Alice tries to download it.
	resp2 := env.get(t, "/api/v1/files/"+fileID, aliceCookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp2.StatusCode)
	}

	// Alice tries to delete it.
	resp3 := env.del(t, "/api/v1/files/"+fileID, aliceCookie)
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusForbidden {
		t.Errorf("delete status = %d, want 403", resp3.StatusCode)
	}
}

func TestFile_AdminCanAccessAnyFile(t *testing.T) {
	env := newTestEnv(t)
	adminCookie := env.setupAdmin(t, "admin", "securepass123")

	// Create user + upload as that user.
	body := `{"username":"bob","password":"password123","role":"user"}`
	resp := env.postAuth(t, "/api/v1/admin/users/", body, adminCookie)
	resp.Body.Close()

	bobCookie := env.login(t, "bob", "password123")
	file := env.uploadFile(t, bobCookie, "bob.txt", "bob's data")
	fileID := file["id"].(string)

	// Admin can download Bob's file.
	resp2 := env.get(t, "/api/v1/files/"+fileID, adminCookie)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("admin download: status = %d, want 200", resp2.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestAuth_Setup_MissingFields(t *testing.T) {
	env := newTestEnv(t)
	env.authSvc.InitSetup(context.Background())

	body := `{"token":"something"}`
	resp := env.post(t, "/api/v1/auth/setup", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAuth_Login_NonexistentUser(t *testing.T) {
	env := newTestEnv(t)
	env.setupAdmin(t, "admin", "securepass123")

	body := `{"username":"ghost","password":"password123"}`
	resp := env.post(t, "/api/v1/auth/login", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestFile_Upload_NoFiles(t *testing.T) {
	env := newTestEnv(t)
	cookie := env.setupAdmin(t, "admin", "securepass123")

	// Multipart with no file parts.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.Close()

	req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/files", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRequestID_HeaderPropagated(t *testing.T) {
	env := newTestEnv(t)

	req, _ := http.NewRequest("GET", env.ts.URL+"/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	rid := resp.Header.Get("X-Request-ID")
	if rid == "" {
		t.Error("expected X-Request-ID in response")
	}
}

func TestSecurityHeaders_Present(t *testing.T) {
	env := newTestEnv(t)

	resp := env.get(t, "/healthz", nil)
	defer resp.Body.Close()

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
	}
	for header, want := range checks {
		got := resp.Header.Get(header)
		if got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
