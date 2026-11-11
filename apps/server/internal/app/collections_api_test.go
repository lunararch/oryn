package app

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestCollectionsAPI(t *testing.T) {
	root := t.TempDir()
	makeAPIGame(t, root, "Alpha", "")
	makeAPIGame(t, root, "Beta", "")
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

	body, _ := json.Marshal(map[string]string{"name": "Comfort", "description": "Rainy days"})
	rec = request(t, handler, http.MethodPost, "/api/v1/collections", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create collection status = %d; body = %s", rec.Code, rec.Body)
	}
	var collection struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		GameCount int    `json:"gameCount"`
	}
	decode(t, rec, &collection)
	if collection.ID == "" || collection.Name != "Comfort" {
		t.Fatalf("collection = %+v", collection)
	}

	rec = request(t, handler, http.MethodPut, "/api/v1/collections/"+collection.ID+"/games/"+gameID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("add game status = %d; body = %s", rec.Code, rec.Body)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/collections", nil)
	var list struct {
		Items []struct {
			GameCount int `json:"gameCount"`
		} `json:"items"`
	}
	decode(t, rec, &list)
	if len(list.Items) != 1 || list.Items[0].GameCount != 1 {
		t.Errorf("collections = %+v", list.Items)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/games/"+gameID+"/collections", nil)
	var membership struct {
		Items []struct {
			Contains bool `json:"contains"`
		} `json:"items"`
	}
	decode(t, rec, &membership)
	if len(membership.Items) != 1 || !membership.Items[0].Contains {
		t.Errorf("membership = %+v", membership.Items)
	}

	rec = request(t, handler, http.MethodGet, "/api/v1/games?collection="+collection.ID, nil)
	var filtered struct {
		Total int `json:"total"`
	}
	decode(t, rec, &filtered)
	if filtered.Total != 1 {
		t.Errorf("filtered total = %d, want 1", filtered.Total)
	}

	rec = request(t, handler, http.MethodDelete, "/api/v1/collections/"+collection.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete collection status = %d", rec.Code)
	}
}
