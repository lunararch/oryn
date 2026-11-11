package artwork

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func TestProcessWritesImageVariantsAndRow(t *testing.T) {
	f := newArtworkFixture(t)
	source := filepath.Join(f.gameDir, "cover.png")
	writePNG(t, source, 900, 1350, color.RGBA{R: 120, G: 80, B: 160, A: 255})

	art, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", source, "cover.png", "detected")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}
	if art.ID == "" || art.CacheKey == "" {
		t.Fatalf("artwork = %+v", art)
	}
	if art.Width != 900 || art.Height != 1350 {
		t.Errorf("dimensions = %dx%d", art.Width, art.Height)
	}

	cardPath, _, err := f.service.VariantPath(context.Background(), art.ID, "card")
	if err != nil {
		t.Fatalf("VariantPath(card) error = %v", err)
	}
	if _, err := os.Stat(cardPath); err != nil {
		t.Fatalf("card variant missing: %v", err)
	}
	cardWidth, _ := imageDimensions(t, cardPath)
	if cardWidth > CardWidth {
		t.Errorf("card width = %d, want <= %d", cardWidth, CardWidth)
	}

	detailPath, _, err := f.service.VariantPath(context.Background(), art.ID, "detail")
	if err != nil {
		t.Fatalf("VariantPath(detail) error = %v", err)
	}
	detailWidth, _ := imageDimensions(t, detailPath)
	if detailWidth > DetailWidth {
		t.Errorf("detail width = %d, want <= %d", detailWidth, DetailWidth)
	}
}

func TestProcessRejectsNonImageContent(t *testing.T) {
	f := newArtworkFixture(t)
	source := filepath.Join(f.gameDir, "cover.png")
	if err := os.WriteFile(source, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", source, "cover.png", "detected"); err == nil {
		t.Fatal("ProcessFile() succeeded for non-image content")
	}
}

func TestProcessRejectsOversizedDimensions(t *testing.T) {
	f := newArtworkFixture(t)
	f.service.maxDimension = 100
	source := filepath.Join(f.gameDir, "cover.png")
	writePNG(t, source, 200, 300, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	if _, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", source, "cover.png", "detected"); err == nil {
		t.Fatal("ProcessFile() succeeded for oversized image")
	}
}

func TestSyncGameDetectsCoverAndBanner(t *testing.T) {
	f := newArtworkFixture(t)
	writePNG(t, filepath.Join(f.gameDir, "cover.png"), 600, 900, color.RGBA{R: 120, G: 80, B: 160, A: 255})
	writePNG(t, filepath.Join(f.gameDir, "game", "gui", "main_menu.png"), 1280, 720, color.RGBA{R: 40, G: 50, B: 60, A: 255})

	if err := f.service.SyncLibrary(context.Background(), f.queries, f.library); err != nil {
		t.Fatalf("SyncLibrary() error = %v", err)
	}
	rows, err := f.queries.ListArtworkForGame(context.Background(), f.gameID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("artwork rows = %d, want 2", len(rows))
	}
	kinds := map[string]bool{}
	for _, row := range rows {
		kinds[row.Kind] = true
	}
	if !kinds["cover"] || !kinds["banner"] {
		t.Errorf("kinds = %v", kinds)
	}
}

func TestSyncPreservesUploadedArtwork(t *testing.T) {
	f := newArtworkFixture(t)
	writePNG(t, filepath.Join(f.gameDir, "cover.png"), 600, 900, color.RGBA{R: 120, G: 80, B: 160, A: 255})
	upload, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", filepath.Join(f.gameDir, "cover.png"), "cover.png", "upload")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.service.SyncLibrary(context.Background(), f.queries, f.library); err != nil {
		t.Fatal(err)
	}
	row, err := f.queries.GetArtworkForGame(context.Background(), db.GetArtworkForGameParams{
		GameID: f.gameID,
		Kind:   "cover",
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != upload.ID || row.Origin != "upload" {
		t.Errorf("cover = %+v, want uploaded artwork preserved", row)
	}
}

func TestVariantPathRejectsUnknownVariant(t *testing.T) {
	f := newArtworkFixture(t)
	writePNG(t, filepath.Join(f.gameDir, "cover.png"), 300, 450, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	art, err := f.service.ProcessFile(context.Background(), f.gameID, "cover", filepath.Join(f.gameDir, "cover.png"), "cover.png", "detected")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.service.VariantPath(context.Background(), art.ID, "../cover"); err == nil {
		t.Fatal("VariantPath() accepted an invalid variant")
	}
}

type artworkFixture struct {
	service *Service
	queries *db.Queries
	library db.Library
	gameID  string
	gameDir string
}

func newArtworkFixture(t *testing.T) *artworkFixture {
	t.Helper()
	root := t.TempDir()
	s, err := store.Open(filepath.Join(t.TempDir(), "oryn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	q := db.New(s.DB)
	lib, err := library.Ensure(context.Background(), q, root, "Test", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	gameID := uuid.Must(uuid.NewV7()).String()
	gameDir := filepath.Join(root, "Example Game")
	if err := os.MkdirAll(filepath.Join(gameDir, "game", "gui"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := q.InsertGame(context.Background(), db.InsertGameParams{
		ID:                 gameID,
		LibraryID:          lib.ID,
		RelativePath:       "Example Game",
		FolderName:         "Example Game",
		DetectedTitle:      "Example Game",
		AvailabilityStatus: "available",
		FirstSeenAt:        now,
		LastSeenAt:         now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{
		Store:    s,
		CacheDir: filepath.Join(t.TempDir(), "artwork"),
		Now:      time.Now,
	})
	return &artworkFixture{
		service: service,
		queries: q,
		library: lib,
		gameID:  gameID,
		gameDir: gameDir,
	}
}

func writePNG(t *testing.T, path string, width, height int, fill color.RGBA) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func imageDimensions(t *testing.T, path string) (int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}
