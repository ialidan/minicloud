package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveness(t *testing.T) {
	h := NewHealth()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Liveness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "alive" {
		t.Errorf("expected status alive, got %s", body["status"])
	}
}

func TestReadiness_NoCheckers(t *testing.T) {
	h := NewHealth()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readiness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReadiness_AllPass(t *testing.T) {
	h := NewHealth()
	h.RegisterChecker("db", func() error { return nil })
	h.RegisterChecker("storage", func() error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readiness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestReadiness_OneFails(t *testing.T) {
	h := NewHealth()
	h.RegisterChecker("db", func() error { return errors.New("connection refused") })
	h.RegisterChecker("storage", func() error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readiness(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "not ready" {
		t.Errorf("expected status 'not ready', got %v", body["status"])
	}
	errs, ok := body["errors"].(map[string]any)
	if !ok {
		t.Fatal("expected errors map in response")
	}
	if errs["db"] != "connection refused" {
		t.Errorf("expected db error, got %v", errs["db"])
	}
}
