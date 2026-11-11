package artwork

import (
	"context"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreUploadReplacesDetectedArtwork(t *testing.T) {
	f := newArtworkFixture(t)
	source := filepath.Join(f.gameDir, "cover.png")
	writePNG(t, source, 600, 900, color.RGBA{R: 20, G: 40, B: 60, A: 255})
	detected, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", source, "cover.png", "detected")
	if err != nil {
		t.Fatal(err)
	}

	uploadPath := filepath.Join(t.TempDir(), "replacement.png")
	writePNG(t, uploadPath, 500, 750, color.RGBA{R: 180, G: 60, B: 40, A: 255})
	file, err := os.Open(uploadPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	uploaded, err := f.service.StoreUpload(context.Background(), f.gameID, "cover", file)
	if err != nil {
		t.Fatalf("StoreUpload() error = %v", err)
	}
	if uploaded.Origin != "upload" {
		t.Errorf("origin = %q, want upload", uploaded.Origin)
	}
	if uploaded.ID == detected.ID {
		t.Error("upload reused the detected artwork ID")
	}

	row, err := f.queries.GetArtworkForGame(context.Background(), getArtworkParams(f.gameID, "cover"))
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != uploaded.ID || row.Origin != "upload" {
		t.Errorf("row = %+v", row)
	}
}

func TestDeleteRemovesArtworkAndVariants(t *testing.T) {
	f := newArtworkFixture(t)
	source := filepath.Join(f.gameDir, "cover.png")
	writePNG(t, source, 300, 450, color.RGBA{R: 20, G: 40, B: 60, A: 255})
	art, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", source, "cover.png", "detected")
	if err != nil {
		t.Fatal(err)
	}
	card, _, err := f.service.VariantPath(context.Background(), art.ID, "card")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.service.Delete(context.Background(), f.gameID, "cover"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(card); !os.IsNotExist(err) {
		t.Errorf("card variant still exists: %v", err)
	}
	if _, _, err := f.service.VariantPath(context.Background(), art.ID, "card"); err == nil {
		t.Error("VariantPath() still finds deleted artwork")
	}
}
