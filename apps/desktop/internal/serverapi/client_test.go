package serverapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAndSessionLifecycle(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/clients/register":
			writeJSON(w, http.StatusCreated, map[string]any{
				"client": map[string]string{"id": "client-1", "name": "Desktop", "platform": "linux"},
				"token":  "oryn_test",
			})
		case "/api/v1/sessions":
			sawAuth = r.Header.Get("Authorization") == "Bearer oryn_test"
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":                       "session-1",
				"gameId":                   "game-1",
				"status":                   "running",
				"source":                   "desktop",
				"startedAt":                "2026-01-01T00:00:00Z",
				"durationSeconds":          0,
				"heartbeatIntervalSeconds": 30,
				"timeoutSeconds":           120,
			})
		case "/api/v1/sessions/session-1/heartbeat":
			if r.Method != http.MethodPost {
				t.Errorf("heartbeat method = %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/sessions/session-1/finish":
			var body struct {
				Status   string `json:"status"`
				ExitCode *int   `json:"exitCode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode finish: %v", err)
			}
			if body.Status != "completed" || body.ExitCode == nil || *body.ExitCode != 0 {
				t.Errorf("finish body = %+v", body)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"id": "session-1", "status": "completed", "durationSeconds": 10,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(server.URL)
	registration, err := client.Register(context.Background(), "ABCD-EFGH", "Desktop", "linux")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registration.Token != "oryn_test" || registration.Client.ID != "client-1" {
		t.Fatalf("registration = %+v", registration)
	}
	client.SetToken(registration.Token)

	session, err := client.StartSession(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if !sawAuth {
		t.Error("session request did not carry the bearer token")
	}
	if session.ID != "session-1" || session.HeartbeatIntervalSeconds != 30 {
		t.Errorf("session = %+v", session)
	}
	if err := client.Heartbeat(context.Background(), session.ID); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	exit := 0
	finished, err := client.Finish(context.Background(), session.ID, "completed", &exit)
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if finished.Status != "completed" {
		t.Errorf("finished = %+v", finished)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]string{"code": "session_already_running", "message": "already running"},
		})
	}))
	defer server.Close()

	client := New(server.URL)
	client.SetToken("oryn_test")
	_, err := client.StartSession(context.Background(), "game-1")
	if err == nil {
		t.Fatal("StartSession() succeeded")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != http.StatusConflict || apiErr.Code != "session_already_running" {
		t.Fatalf("error = %#v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
