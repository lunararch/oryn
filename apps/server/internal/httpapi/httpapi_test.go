package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthOK(t *testing.T) {
	handler := New(Options{
		Version: "1.2.3",
		Ping:    func(context.Context) error { return nil },
		LibraryStatus: func(context.Context) LibraryStatus {
			return LibraryAvailable
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if got := rec.Header().Get("X-Request-Id"); got == "" {
		t.Error("X-Request-Id is empty")
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", body.Version)
	}
	if body.Database != "ok" {
		t.Errorf("database = %q, want ok", body.Database)
	}
	if body.Library != "available" {
		t.Errorf("library = %q, want available", body.Library)
	}
}

func TestHealthReportsDatabaseFailure(t *testing.T) {
	handler := New(Options{
		Version: "1.2.3",
		Ping: func(context.Context) error {
			return errors.New("database is locked: /secret/path")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "degraded" {
		t.Errorf("status = %q, want degraded", body.Status)
	}
	if body.Database != "unavailable" {
		t.Errorf("database = %q, want unavailable", body.Database)
	}
}

func TestLibraryUnavailableDoesNotFailProcessHealth(t *testing.T) {
	handler := New(Options{
		Version: "1.2.3",
		Ping:    func(context.Context) error { return nil },
		LibraryStatus: func(context.Context) LibraryStatus {
			return LibraryUnavailable
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Library != "unavailable" {
		t.Errorf("library = %q, want unavailable", body.Library)
	}
}

func TestVersion(t *testing.T) {
	handler := New(Options{Version: "1.2.3"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body versionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", body.Version)
	}
}

func TestNotFoundUsesErrorEnvelope(t *testing.T) {
	handler := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("code = %q, want not_found", body.Error.Code)
	}
}
