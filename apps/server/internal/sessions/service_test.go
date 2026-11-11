package sessions

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func TestCreateManualSession(t *testing.T) {
	f := newSessionsFixture(t)
	start := time.Now().Add(-90 * time.Minute).Truncate(time.Second)
	end := start.Add(45 * time.Minute)

	created, err := f.service.CreateManual(context.Background(), f.gameID, start, end, "First route")
	if err != nil {
		t.Fatalf("CreateManual() error = %v", err)
	}
	if created.DurationSeconds != 2700 {
		t.Errorf("duration = %d, want 2700", created.DurationSeconds)
	}
	if created.Status != "completed" || created.Source != "manual" {
		t.Errorf("session = %+v", created)
	}
	if created.Notes != "First route" {
		t.Errorf("notes = %q", created.Notes)
	}

	page, err := f.service.ListForGame(context.Background(), f.gameID, 1, 20)
	if err != nil {
		t.Fatalf("ListForGame() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
}

func TestManualSessionValidation(t *testing.T) {
	f := newSessionsFixture(t)
	now := time.Now()
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		notes string
	}{
		{name: "end before start", start: now, end: now.Add(-time.Minute)},
		{name: "zero duration", start: now, end: now},
		{name: "future end", start: now.Add(10 * time.Minute), end: now.Add(20 * time.Minute)},
		{name: "too long", start: now.Add(-8 * 24 * time.Hour), end: now},
		{name: "notes too long", start: now.Add(-time.Hour), end: now, notes: string(make([]byte, 2001))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateManual(context.Background(), f.gameID, tt.start, tt.end, tt.notes)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
		})
	}
}

func TestManualSessionRequiresGame(t *testing.T) {
	f := newSessionsFixture(t)
	if _, err := f.service.CreateManual(context.Background(), uuid.Must(uuid.NewV7()).String(), time.Now().Add(-time.Hour), time.Now(), ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestUpdateAndDeleteManualSession(t *testing.T) {
	f := newSessionsFixture(t)
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)
	created, err := f.service.CreateManual(context.Background(), f.gameID, start, end, "")
	if err != nil {
		t.Fatal(err)
	}

	newStart := start.Add(-15 * time.Minute)
	newEnd := end.Add(15 * time.Minute)
	updated, err := f.service.UpdateManual(context.Background(), created.ID, newStart, newEnd, "Corrected")
	if err != nil {
		t.Fatalf("UpdateManual() error = %v", err)
	}
	if updated.DurationSeconds != 5400 || updated.Notes != "Corrected" {
		t.Errorf("updated = %+v", updated)
	}

	if err := f.service.DeleteManual(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteManual() error = %v", err)
	}
	if _, err := f.service.Get(context.Background(), created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestPlayStats(t *testing.T) {
	f := newSessionsFixture(t)
	start := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	if _, err := f.service.CreateManual(context.Background(), f.gameID, start, start.Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateManual(context.Background(), f.gameID, start.Add(2*time.Hour), start.Add(3*time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	stats, err := f.service.Stats(context.Background(), f.library.ID)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.PlayedGames != 1 || stats.PlaytimeSeconds != 7200 {
		t.Errorf("stats = %+v", stats)
	}
}

type sessionsFixture struct {
	service *Service
	store   *store.Store
	library db.Library
	gameID  string
}

func newSessionsFixture(t *testing.T) *sessionsFixture {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "oryn.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	q := db.New(s.DB)
	lib, err := library.Ensure(context.Background(), q, t.TempDir(), "Test", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	gameID := uuid.Must(uuid.NewV7()).String()
	if err := q.InsertGame(context.Background(), db.InsertGameParams{
		ID:                 gameID,
		LibraryID:          lib.ID,
		RelativePath:       "Game",
		FolderName:         "Game",
		DetectedTitle:      "Game",
		AvailabilityStatus: "available",
		FirstSeenAt:        now,
		LastSeenAt:         now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatal(err)
	}
	return &sessionsFixture{
		service: NewService(s, time.Now),
		store:   s,
		library: lib,
		gameID:  gameID,
	}
}
