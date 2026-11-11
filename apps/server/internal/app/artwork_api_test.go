package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/lunararch/oryn/apps/server/internal/config"
)

func TestArtworkUploadAndDelete(t *testing.T) {
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
			ID             string `json:"id"`
			CoverArtworkID string `json:"coverArtworkId"`
		} `json:"items"`
	}
	decode(t, rec, &list)
	if len(list.Items) != 1 || list.Items[0].CoverArtworkID == "" {
		t.Fatalf("initial cover missing: %+v", list.Items)
	}
	gameID := list.Items[0].ID
	oldCover := list.Items[0].CoverArtworkID

	body, contentType := multipartImage(t, "cover", "replacement.png", 500, 750)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/games/"+gameID+"/artwork", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d; body = %s", rec.Code, rec.Body)
	}
	var detail struct {
		CoverArtworkID string `json:"coverArtworkId"`
	}
	decode(t, rec, &detail)
	if detail.CoverArtworkID == "" || detail.CoverArtworkID == oldCover {
		t.Errorf("cover artwork ID = %q, want a replacement", detail.CoverArtworkID)
	}

	rec = request(t, handler, http.MethodDelete, "/api/v1/games/"+gameID+"/artwork/cover", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete artwork status = %d; body = %s", rec.Code, rec.Body)
	}
	var afterDelete struct {
		CoverArtworkID string `json:"coverArtworkId"`
	}
	decode(t, rec, &afterDelete)
	if afterDelete.CoverArtworkID != "" {
		t.Errorf("cover artwork ID = %q, want empty", afterDelete.CoverArtworkID)
	}
}

func multipartImage(t *testing.T, kind, filename string, width, height int) (*bytes.Buffer, string) {
	t.Helper()
	var imageBytes bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: uint8(x % 255), B: uint8(y % 255), A: 255})
		}
	}
	if err := png.Encode(&imageBytes, img); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("kind", kind); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(imageBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}
