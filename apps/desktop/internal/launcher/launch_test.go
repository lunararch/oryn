package launcher

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lunararch/oryn/apps/desktop/internal/serverapi"
)

func TestLaunchRejectsPathOutsideGame(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Game")
	if err := os.MkdirAll(gameDir, 0o750); err != nil {
		t.Fatal(err)
	}
	service := &Service{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	_, err := service.LaunchAndTrack(context.Background(), "game-1", root, "Game", Candidate{RelativePath: "../outside"}, nil)
	if err == nil {
		t.Fatal("LaunchAndTrack() accepted a path outside the game")
	}
}

func TestLaunchAndTrackRunsControlledExecutable(t *testing.T) {
	if _, err := os.Stat("/bin/true"); err != nil {
		t.Skip("POSIX test executable unavailable")
	}
	root := t.TempDir()
	gameDir := filepath.Join(root, "Game")
	if err := os.MkdirAll(gameDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := copyExecutable("/bin/true", filepath.Join(gameDir, "Game")); err != nil {
		t.Fatal(err)
	}

	var started, finished bool
	var finishStatus string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			started = true
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":                       "session-1",
				"gameId":                   "game-1",
				"status":                   "running",
				"heartbeatIntervalSeconds": 30,
				"timeoutSeconds":           120,
			})
		case "/api/v1/sessions/session-1/heartbeat":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/sessions/session-1/finish":
			finished = true
			var body struct {
				Status string `json:"status"`
			}
			_ = decodeJSON(r, &body)
			finishStatus = body.Status
			writeJSON(w, http.StatusOK, map[string]any{
				"id": "session-1", "status": body.Status, "durationSeconds": 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := serverapi.New(server.URL)
	client.SetToken("oryn_test")
	service := &Service{
		API:    client,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	result, err := service.LaunchAndTrack(context.Background(), "game-1", root, "Game", Candidate{
		RelativePath: "Game",
		Kind:         KindBinary,
	}, nil)
	if err != nil {
		t.Fatalf("LaunchAndTrack() error = %v", err)
	}
	if !started || !finished {
		t.Fatalf("started = %v, finished = %v", started, finished)
	}
	if finishStatus != "completed" {
		t.Errorf("finish status = %q", finishStatus)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d", result.ExitCode)
	}
}

func copyExecutable(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o755)
}

func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
