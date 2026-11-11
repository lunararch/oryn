package scanner

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func TestScanIndexesGames(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	makeGame(t, f.root, "Beta")

	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Status != ScanCompleted {
		t.Fatalf("status = %q, summary = %q", result.Status, result.Summary)
	}
	if result.Discovered != 2 || result.Added != 2 || result.Updated != 0 {
		t.Fatalf("counts = %+v, want discovered 2, added 2", result)
	}

	games, err := f.queries.ListGamesForLibrary(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("ListGamesForLibrary() error = %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("games = %d, want 2", len(games))
	}
	if games[0].DetectedTitle != "Alpha" || games[1].DetectedTitle != "Beta" {
		t.Errorf("titles = %q, %q", games[0].DetectedTitle, games[1].DetectedTitle)
	}
	for _, game := range games {
		if game.AvailabilityStatus != "available" {
			t.Errorf("%s availability = %q", game.DetectedTitle, game.AvailabilityStatus)
		}
		if game.FilesystemFingerprint == "" {
			t.Errorf("%s fingerprint is empty", game.DetectedTitle)
		}
	}
}

func TestScanIsIdempotent(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	makeGame(t, f.root, "Beta")

	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Added != 0 {
		t.Errorf("added = %d, want 0", result.Added)
	}
	if result.Unchanged != 2 {
		t.Errorf("unchanged = %d, want 2", result.Unchanged)
	}

	count, err := f.queries.CountGames(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("CountGames() error = %v", err)
	}
	if count != 2 {
		t.Errorf("games = %d, want 2", count)
	}
}

func TestScanAddsNewGames(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	makeGame(t, f.root, "Beta")
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Added != 1 {
		t.Errorf("added = %d, want 1", result.Added)
	}
}

func TestScanMarksMissingGame(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	makeGame(t, f.root, "Beta")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	if err := os.RemoveAll(filepath.Join(f.root, "Beta")); err != nil {
		t.Fatalf("remove game: %v", err)
	}
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Status != ScanCompleted || result.Missing != 1 {
		t.Fatalf("result = %+v, want completed with 1 missing", result)
	}

	games, err := f.queries.ListGamesForLibrary(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("ListGamesForLibrary() error = %v", err)
	}
	for _, game := range games {
		if game.DetectedTitle == "Beta" && game.AvailabilityStatus != "missing" {
			t.Errorf("Beta availability = %q, want missing", game.AvailabilityStatus)
		}
	}
}

func TestScanDoesNotMarkMissingWhenLibraryLooksEmpty(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	makeGame(t, f.root, "Beta")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	if err := os.RemoveAll(filepath.Join(f.root, "Alpha")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(f.root, "Beta")); err != nil {
		t.Fatal(err)
	}
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Status != ScanFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.Missing != 0 {
		t.Errorf("missing = %d, want 0", result.Missing)
	}

	count, err := f.queries.CountAvailableGames(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("CountAvailableGames() error = %v", err)
	}
	if count != 2 {
		t.Errorf("available games = %d, want 2", count)
	}
}

func TestScanGuardsAgainstMassMissing(t *testing.T) {
	f := newScanFixture(t)
	for i := 0; i < 12; i++ {
		makeGame(t, f.root, "Game "+string(rune('A'+i)))
	}
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	for i := 0; i < 11; i++ {
		if err := os.RemoveAll(filepath.Join(f.root, "Game "+string(rune('A'+i)))); err != nil {
			t.Fatal(err)
		}
	}
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Status != ScanPartial {
		t.Fatalf("status = %q, want partial", result.Status)
	}
	if result.Missing != 0 {
		t.Errorf("missing = %d, want 0", result.Missing)
	}
	if result.MissingCandidates != 11 {
		t.Errorf("missing candidates = %d, want 11", result.MissingCandidates)
	}

	count, err := f.queries.CountAvailableGames(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("CountAvailableGames() error = %v", err)
	}
	if count != 12 {
		t.Errorf("available games = %d, want 12", count)
	}
}

func TestScanAllowsMassMissingWhenExplicitlyRequested(t *testing.T) {
	f := newScanFixture(t)
	for i := 0; i < 12; i++ {
		makeGame(t, f.root, "Game "+string(rune('A'+i)))
	}
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}
	for i := 0; i < 11; i++ {
		if err := os.RemoveAll(filepath.Join(f.root, "Game "+string(rune('A'+i)))); err != nil {
			t.Fatal(err)
		}
	}

	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{AllowMissing: true})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Status != ScanCompleted || result.Missing != 11 {
		t.Fatalf("result = %+v, want completed with 11 missing", result)
	}
}

func TestScanFailsSafelyWhenRootIsUnavailable(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	unavailable := f.root + "-unavailable"
	if err := os.Rename(f.root, unavailable); err != nil {
		t.Fatalf("rename root: %v", err)
	}
	t.Cleanup(func() { _ = os.Rename(unavailable, f.root) })

	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Status != ScanFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	count, err := f.queries.CountAvailableGames(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("CountAvailableGames() error = %v", err)
	}
	if count != 1 {
		t.Errorf("available games = %d, want 1", count)
	}
}

func TestScanRequiresSentinel(t *testing.T) {
	f := newScanFixture(t)
	f.service.sentinel = "library.marker"
	makeGame(t, f.root, "Alpha")

	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Status != ScanFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}

	writeFile(t, filepath.Join(f.root, "library.marker"), "present")
	result, err = f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Status != ScanCompleted {
		t.Fatalf("status = %q, summary = %q", result.Status, result.Summary)
	}
}

func TestScanPreservesMetadataOverride(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	games, err := f.queries.ListGamesForLibrary(context.Background(), f.library.ID)
	if err != nil {
		t.Fatal(err)
	}
	gameID := games[0].ID
	if err := f.queries.UpsertGameOverride(context.Background(), db.UpsertGameOverrideParams{
		GameID:    gameID,
		Title:     sql.NullString{String: "Custom Title", Valid: true},
		UpdatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("UpsertGameOverride() error = %v", err)
	}

	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	game, err := f.queries.GetGame(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetGame() error = %v", err)
	}
	if !game.OverrideTitle.Valid || game.OverrideTitle.String != "Custom Title" {
		t.Errorf("override title = %+v, want Custom Title", game.OverrideTitle)
	}
}

func TestScanDetectsRenamedGame(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Old Name")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	games, err := f.queries.ListGamesForLibrary(context.Background(), f.library.ID)
	if err != nil {
		t.Fatal(err)
	}
	originalID := games[0].ID
	if err := f.queries.UpsertGameOverride(context.Background(), db.UpsertGameOverrideParams{
		GameID:    originalID,
		Title:     sql.NullString{String: "Custom Title", Valid: true},
		UpdatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(filepath.Join(f.root, "Old Name"), filepath.Join(f.root, "New Name")); err != nil {
		t.Fatalf("rename game: %v", err)
	}
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("second Scan() error = %v", err)
	}
	if result.Added != 0 {
		t.Errorf("added = %d, want 0 for a rename", result.Added)
	}

	games, err = f.queries.ListGamesForLibrary(context.Background(), f.library.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}
	if games[0].ID != originalID {
		t.Errorf("ID = %q, want original %q", games[0].ID, originalID)
	}
	if games[0].RelativePath != "New Name" {
		t.Errorf("path = %q, want New Name", games[0].RelativePath)
	}
	game, err := f.queries.GetGame(context.Background(), originalID)
	if err != nil {
		t.Fatal(err)
	}
	if !game.OverrideTitle.Valid || game.OverrideTitle.String != "Custom Title" {
		t.Errorf("override = %+v, want preserved", game.OverrideTitle)
	}
}

func TestScanCancellationDoesNotReconcile(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	if _, err := f.service.Scan(context.Background(), f.library, ScanOptions{}); err != nil {
		t.Fatalf("first Scan() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := f.service.Scan(ctx, f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("cancelled Scan() error = %v", err)
	}
	if result.Status != ScanCancelled {
		t.Fatalf("status = %q, want cancelled", result.Status)
	}
}

func TestScanRecordsHistory(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	result, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	run, err := f.queries.GetScanRun(context.Background(), result.ScanRunID)
	if err != nil {
		t.Fatalf("GetScanRun() error = %v", err)
	}
	if run.Status != ScanCompleted || !run.FinishedAt.Valid {
		t.Errorf("run = %+v", run)
	}
	if run.DiscoveredCount != 1 || run.AddedCount != 1 {
		t.Errorf("run counts = %+v", run)
	}
}

func TestScanRejectsConcurrentRun(t *testing.T) {
	f := newScanFixture(t)
	makeGame(t, f.root, "Alpha")
	f.service.mu.Lock()
	defer f.service.mu.Unlock()

	_, err := f.service.Scan(context.Background(), f.library, ScanOptions{})
	if !errors.Is(err, ErrScanInProgress) {
		t.Fatalf("Scan() error = %v, want ErrScanInProgress", err)
	}
}

type scanFixture struct {
	service *Service
	store   *store.Store
	queries *db.Queries
	library db.Library
	root    string
}

func newScanFixture(t *testing.T) *scanFixture {
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
	lib, err := library.Ensure(context.Background(), q, root, "Test Library", time.Now)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	service := NewService(ServiceOptions{
		Store:    s,
		MaxDepth: 3,
		Now:      time.Now,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return &scanFixture{service: service, store: s, queries: q, library: lib, root: root}
}
