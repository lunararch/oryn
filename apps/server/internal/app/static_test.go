package app

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestServesSPAWhenWebDirConfigured(t *testing.T) {
	webDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><title>Oryn SPA</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte("console.log('oryn')"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, config.Config{
		DataDir:      t.TempDir(),
		DBPath:       filepath.Join(t.TempDir(), "oryn.db"),
		LibraryRoot:  t.TempDir(),
		WebDir:       webDir,
		ScanMaxDepth: 3,
	})
	handler := a.Handler()

	rec := request(t, handler, http.MethodGet, "/library", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("SPA status = %d; body = %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "Oryn SPA") {
		t.Errorf("SPA body = %q", rec.Body)
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy is empty")
	}

	rec = request(t, handler, http.MethodGet, "/assets/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("asset status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("asset Cache-Control = %q", got)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("health Content-Type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestRejectsMissingWebIndex(t *testing.T) {
	webDir := t.TempDir()
	_, err := New(config.Config{
		DataDir:      t.TempDir(),
		DBPath:       filepath.Join(t.TempDir(), "oryn.db"),
		LibraryRoot:  t.TempDir(),
		WebDir:       webDir,
		ScanMaxDepth: 3,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), "test")
	if err == nil {
		t.Fatal("New() succeeded with a missing index.html")
	}
}
