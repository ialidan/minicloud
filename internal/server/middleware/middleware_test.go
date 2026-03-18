package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minicloud/internal/domain"
)

// ---------------------------------------------------------------------------
// RequestID middleware tests
// ---------------------------------------------------------------------------

func TestRequestID_GeneratesID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID in context")
		}
		if len(id) != 16 { // 8 bytes = 16 hex chars
			t.Errorf("request ID length = %d, want 16", len(id))
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should also set the response header.
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
}

func TestRequestID_ReusesExisting(t *testing.T) {
	existingID := "my-custom-request-id"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != existingID {
			t.Errorf("request ID = %q, want %q", id, existingID)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != existingID {
		t.Errorf("response header = %q, want %q", rec.Header().Get("X-Request-ID"), existingID)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	id := GetRequestID(context.Background())
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}

// ---------------------------------------------------------------------------
// RequireAuth middleware tests
// ---------------------------------------------------------------------------

// mockValidator implements SessionValidator for testing.
type mockValidator struct {
	user    *domain.User
	session *domain.Session
	err     error
}

func (m *mockValidator) ValidateSession(_ context.Context, _ string) (*domain.User, *domain.Session, error) {
	return m.user, m.session, m.err
}

func TestRequireAuth_NoCookie(t *testing.T) {
	mw := RequireAuth(&mockValidator{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	mw := RequireAuth(&mockValidator{err: domain.ErrNotFound})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "bad-session"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	user := &domain.User{ID: "u1", Username: "alice", Role: domain.RoleUser, IsActive: true}
	session := &domain.Session{ID: "s1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}

	mw := RequireAuth(&mockValidator{user: user, session: session})

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		u := UserFromContext(r.Context())
		if u == nil || u.ID != "u1" {
			t.Error("expected user in context")
		}
		s := SessionFromContext(r.Context())
		if s == nil || s.ID != "s1" {
			t.Error("expected session in context")
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "valid-session"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

// ---------------------------------------------------------------------------
// RequireAdmin middleware tests
// ---------------------------------------------------------------------------

func TestRequireAdmin_NonAdmin(t *testing.T) {
	user := &domain.User{ID: "u1", Role: domain.RoleUser}
	ctx := context.WithValue(context.Background(), UserKey, user)

	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireAdmin_Admin(t *testing.T) {
	user := &domain.User{ID: "u1", Role: domain.RoleAdmin}
	ctx := context.WithValue(context.Background(), UserKey, user)

	var called bool
	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called for admin")
	}
}

func TestRequireAdmin_NoUser(t *testing.T) {
	handler := RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// ---------------------------------------------------------------------------
// SecureHeaders middleware tests
// ---------------------------------------------------------------------------

func TestSecureHeaders(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "0",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
		"Permissions-Policy":    "camera=(), microphone=(), geolocation=()",
	}

	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Context helper tests
// ---------------------------------------------------------------------------

func TestUserFromContext_Nil(t *testing.T) {
	u := UserFromContext(context.Background())
	if u != nil {
		t.Error("expected nil user from empty context")
	}
}

func TestSessionFromContext_Nil(t *testing.T) {
	s := SessionFromContext(context.Background())
	if s != nil {
		t.Error("expected nil session from empty context")
	}
}
