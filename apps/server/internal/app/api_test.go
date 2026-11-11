package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestScanAndLibraryAPI(t *testing.T) {
	root := t.TempDir()
	makeAPIGame(t, root, "Alpha", "A romantic visual novel")
	makeAPIGame(t, root, "Beta", "A mystery visual novel")

	a := newTestApp(t, config.Config{
		DataDir:      t.TempDir(),
		DBPath:       filepath.Join(t.TempDir(), "oryn.db"),
		LibraryRoot:  root,
		ScanMaxDepth: 3,
	})
	handler := a.Handler()

	rec := request(t, handler, http.MethodPost, "/api/v1/scans", []byte(`{}`))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start scan status = %d; body = %s", rec.Code, rec.Body)
	}
	waitForScan(t, handler)

	rec = request(t, handler, http.MethodGet, "/api/v1/games?sort=title&order=asc", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %s", rec.Code, rec.Body)
	}
	var list struct {
		Items      []map[string]any `json:"items"`
		Total      int              `json:"total"`
		TotalPages int              `json:"totalPages"`
	}
	decode(t, rec, &list)
	if list.Total != 2 || len(list.Items) != 2 {
		t.Fatalf("list = %+v", list)
	}
	if list.Items[0]["title"] != "Alpha" {
		t.Errorf("first title = %v", list.Items[0]["title"])
	}
	gameID := list.Items[0]["id"].(string)

	rec = request(t, handler, http.MethodPut, "/api/v1/games/"+gameID+"/favorite", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("favorite status = %d; body = %s", rec.Code, rec.Body)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/libraries", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("libraries status = %d", rec.Code)
	}
	var libraries struct {
		Items []struct {
			GameCount  int    `json:"gameCount"`
			LastScanAt string `json:"lastScanAt"`
		} `json:"items"`
	}
	decode(t, rec, &libraries)
	if len(libraries.Items) != 1 || libraries.Items[0].GameCount != 2 {
		t.Errorf("libraries = %+v", libraries.Items)
	}
	if libraries.Items[0].LastScanAt == "" {
		t.Error("lastScanAt is empty after a completed scan")
	}

	rec = request(t, handler, http.MethodPatch, "/api/v1/games/"+gameID, []byte(`{
		"title": "Alpha Custom",
		"description": "Edited",
		"developer": "Studio",
		"publisher": "",
		"releaseDate": "2025",
		"rating": 9,
		"notes": "Great",
		"completionStatus": "playing"
	}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d; body = %s", rec.Code, rec.Body)
	}
	var detail map[string]any
	decode(t, rec, &detail)
	if detail["title"] != "Alpha Custom" || detail["favorite"] != true {
		t.Errorf("detail = %+v", detail)
	}
	if detail["rating"] != float64(9) || detail["completionStatus"] != "playing" {
		t.Errorf("detail = %+v", detail)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/games?favorite=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("favorite list status = %d", rec.Code)
	}
	decode(t, rec, &list)
	if list.Total != 1 {
		t.Errorf("favorite total = %d, want 1", list.Total)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/stats", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d", rec.Code)
	}
	var stats struct {
		Total     int `json:"total"`
		Available int `json:"available"`
		Favorites int `json:"favorites"`
	}
	decode(t, rec, &stats)
	if stats.Total != 2 || stats.Available != 2 || stats.Favorites != 1 {
		t.Errorf("stats = %+v", stats)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/scans", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan history status = %d", rec.Code)
	}
	var history struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	decode(t, rec, &history)
	if len(history.Items) != 1 || history.Items[0].Status != "completed" {
		t.Errorf("history = %+v", history)
	}
}

func TestInvalidMetadataIsRejected(t *testing.T) {
	root := t.TempDir()
	makeAPIGame(t, root, "Alpha", "")
	a := newTestApp(t, config.Config{
		DataDir:      t.TempDir(),
		DBPath:       filepath.Join(t.TempDir(), "oryn.db"),
		LibraryRoot:  root,
		ScanMaxDepth: 3,
	})
	handler := a.Handler()
	if rec := request(t, handler, http.MethodPost, "/api/v1/scans", []byte(`{}`)); rec.Code != http.StatusAccepted {
		t.Fatalf("scan status = %d", rec.Code)
	}
	waitForScan(t, handler)

	rec := request(t, handler, http.MethodGet, "/api/v1/games", nil)
	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	decode(t, rec, &list)
	if len(list.Items) != 1 {
		t.Fatal("no game")
	}

	rec = request(t, handler, http.MethodPatch, "/api/v1/games/"+list.Items[0].ID, []byte(`{"rating": 12}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body)
	}
}

func TestConcurrentScanConflict(t *testing.T) {
	root := t.TempDir()
	makeAPIGame(t, root, "Alpha", "")
	a := newTestApp(t, config.Config{
		DataDir:      t.TempDir(),
		DBPath:       filepath.Join(t.TempDir(), "oryn.db"),
		LibraryRoot:  root,
		ScanMaxDepth: 3,
	})
	handler := a.Handler()
	if rec := request(t, handler, http.MethodPost, "/api/v1/scans", []byte(`{}`)); rec.Code != http.StatusAccepted {
		t.Fatalf("first scan status = %d", rec.Code)
	}
	if rec := request(t, handler, http.MethodPost, "/api/v1/scans", []byte(`{}`)); rec.Code != http.StatusConflict {
		t.Fatalf("second scan status = %d, want 409; body = %s", rec.Code, rec.Body)
	}
	waitForScan(t, handler)
}

func request(t *testing.T, handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func waitForScan(t *testing.T, handler http.Handler) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec := request(t, handler, http.MethodGet, "/api/v1/scans/current", nil)
		var state struct {
			Running bool   `json:"running"`
			Status  string `json:"status"`
		}
		decode(t, rec, &state)
		if !state.Running && state.Status != "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("scan did not finish")
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
		t.Fatalf("decode: %v; body = %s", err, rec.Body)
	}
}

func makeAPIGame(t *testing.T, root, name, description string) {
	t.Helper()
	dir := filepath.Join(root, name)
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("game/script.rpyc", "compiled")
	write("renpy/common/00layout.rpyc", "runtime")
	write("lib/linux-x86_64/game", "binary")
	write(name+".exe", "launcher")
	write("game/options.rpy", fmt.Sprintf(`define config.name = _("%s")%s
define build.name = "%s"
`, name, func() string {
		if description == "" {
			return ""
		}
		return fmt.Sprintf("\ndefine gui.about = _p(\"\"\"%s\"\"\")", description)
	}(), name))
}
