package app

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestManualSessionAPI(t *testing.T) {
	root := t.TempDir()
	makeAPIGame(t, root, "Alpha", "A story")
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
	gameID := list.Items[0].ID

	ended := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	started := ended.Add(-time.Hour)
	body, _ := json.Marshal(map[string]string{
		"startedAt": started.Format(time.RFC3339),
		"endedAt":   ended.Format(time.RFC3339),
		"notes":     "First session",
	})
	rec = request(t, handler, http.MethodPost, "/api/v1/games/"+gameID+"/sessions", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create session status = %d; body = %s", rec.Code, rec.Body)
	}
	var session struct {
		ID              string `json:"id"`
		DurationSeconds int    `json:"durationSeconds"`
		Notes           string `json:"notes"`
	}
	decode(t, rec, &session)
	if session.DurationSeconds != 3600 || session.Notes != "First session" {
		t.Errorf("session = %+v", session)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/games/"+gameID+"/sessions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d", rec.Code)
	}
	var sessions struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	decode(t, rec, &sessions)
	if sessions.Total != 1 || len(sessions.Items) != 1 {
		t.Fatalf("sessions = %+v", sessions)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/stats", nil)
	var stats struct {
		PlayedGames     int `json:"playedGames"`
		PlaytimeSeconds int `json:"playtimeSeconds"`
	}
	decode(t, rec, &stats)
	if stats.PlayedGames != 1 || stats.PlaytimeSeconds != 3600 {
		t.Errorf("stats = %+v", stats)
	}

	rec = request(t, handler, http.MethodDelete, "/api/v1/sessions/"+session.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete session status = %d", rec.Code)
	}
}
