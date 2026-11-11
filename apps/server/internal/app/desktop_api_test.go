package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestDesktopPairingAndSessionAPI(t *testing.T) {
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
	var games struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	decode(t, rec, &games)
	gameID := games.Items[0].ID

	rec = request(t, handler, http.MethodPost, "/api/v1/clients/pairing-codes", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("pairing code status = %d; body = %s", rec.Code, rec.Body)
	}
	var pairing struct {
		Code string `json:"code"`
	}
	decode(t, rec, &pairing)
	if pairing.Code == "" {
		t.Fatal("pairing code is empty")
	}

	body, _ := json.Marshal(map[string]string{
		"code":     pairing.Code,
		"name":     "Test Desktop",
		"platform": "linux",
	})
	rec = request(t, handler, http.MethodPost, "/api/v1/clients/register", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d; body = %s", rec.Code, rec.Body)
	}
	var registration struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
		Token string `json:"token"`
	}
	decode(t, rec, &registration)
	if registration.Client.ID == "" || registration.Token == "" {
		t.Fatalf("registration = %+v", registration)
	}

	sessionBody, _ := json.Marshal(map[string]string{"gameId": gameID})
	rec = request(t, handler, http.MethodPost, "/api/v1/sessions", sessionBody)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated session status = %d, want 401", rec.Code)
	}

	rec = requestWithToken(t, handler, http.MethodPost, "/api/v1/sessions", sessionBody, registration.Token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start session status = %d; body = %s", rec.Code, rec.Body)
	}
	var session struct {
		ID                       string `json:"id"`
		HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds"`
	}
	decode(t, rec, &session)
	if session.ID == "" || session.HeartbeatIntervalSeconds != 30 {
		t.Fatalf("session = %+v", session)
	}

	rec = requestWithToken(t, handler, http.MethodPost, "/api/v1/sessions/"+session.ID+"/heartbeat", nil, registration.Token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d; body = %s", rec.Code, rec.Body)
	}

	finishBody, _ := json.Marshal(map[string]any{"status": "completed", "exitCode": 0})
	rec = requestWithToken(t, handler, http.MethodPost, "/api/v1/sessions/"+session.ID+"/finish", finishBody, registration.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("finish status = %d; body = %s", rec.Code, rec.Body)
	}

	rec = request(t, handler, http.MethodDelete, "/api/v1/clients/"+registration.Client.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d", rec.Code)
	}
	rec = requestWithToken(t, handler, http.MethodPost, "/api/v1/sessions", sessionBody, registration.Token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d, want 401", rec.Code)
	}
}

func requestWithToken(
	t *testing.T,
	handler http.Handler,
	method, path string,
	body []byte,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
