package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestHandlerHealthReportsLibrary(t *testing.T) {
	root := t.TempDir()
	a := New(config.Config{LibraryRoot: root}, slog.New(slog.NewTextHandler(io.Discard, nil)), "test-version")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		Version string `json:"version"`
		Library string `json:"library"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Version != "test-version" {
		t.Errorf("version = %q", body.Version)
	}
	if body.Library != "available" {
		t.Errorf("library = %q, want available", body.Library)
	}
}
