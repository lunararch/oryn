package collections

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

func TestCollectionCRUD(t *testing.T) {
	f := newCollectionsFixture(t)
	created, err := f.service.Create(context.Background(), "Comfort games", "Rainy day stories")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" || created.Name != "Comfort games" {
		t.Fatalf("created = %+v", created)
	}

	list, err := f.service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].GameCount != 0 {
		t.Fatalf("list = %+v", list)
	}

	updated, err := f.service.Update(context.Background(), created.ID, "Rainy day", "Updated")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Rainy day" || updated.Description != "Updated" {
		t.Errorf("updated = %+v", updated)
	}

	if err := f.service.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := f.service.Get(context.Background(), created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestCollectionValidationAndDuplicates(t *testing.T) {
	f := newCollectionsFixture(t)
	if _, err := f.service.Create(context.Background(), "", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name error = %v", err)
	}
	if _, err := f.service.Create(context.Background(), "Duplicate", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Create(context.Background(), "duplicate", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate error = %v, want ErrConflict", err)
	}
}

func TestCollectionMembership(t *testing.T) {
	f := newCollectionsFixture(t)
	collection, err := f.service.Create(context.Background(), "Favorites", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.AddGame(context.Background(), collection.ID, f.gameID); err != nil {
		t.Fatalf("AddGame() error = %v", err)
	}
	if err := f.service.AddGame(context.Background(), collection.ID, f.gameID); err != nil {
		t.Fatalf("second AddGame() error = %v", err)
	}

	loaded, err := f.service.Get(context.Background(), collection.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GameCount != 1 {
		t.Errorf("gameCount = %d, want 1", loaded.GameCount)
	}

	membership, err := f.service.ListForGame(context.Background(), f.gameID)
	if err != nil {
		t.Fatal(err)
	}
	if len(membership) != 1 || !membership[0].Contains {
		t.Errorf("membership = %+v", membership)
	}

	if err := f.service.RemoveGame(context.Background(), collection.ID, f.gameID); err != nil {
		t.Fatalf("RemoveGame() error = %v", err)
	}
	loaded, err = f.service.Get(context.Background(), collection.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GameCount != 0 {
		t.Errorf("gameCount = %d, want 0", loaded.GameCount)
	}
}

func TestCollectionRequiresExistingGame(t *testing.T) {
	f := newCollectionsFixture(t)
	collection, err := f.service.Create(context.Background(), "Collection", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.AddGame(context.Background(), collection.ID, uuid.Must(uuid.NewV7()).String()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestDeleteCollectionCascadesMembership(t *testing.T) {
	f := newCollectionsFixture(t)
	collection, err := f.service.Create(context.Background(), "Collection", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.AddGame(context.Background(), collection.ID, f.gameID); err != nil {
		t.Fatal(err)
	}
	if err := f.service.Delete(context.Background(), collection.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM collection_games`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("membership rows = %d, want 0", count)
	}
}

type collectionsFixture struct {
	service *Service
	store   *store.Store
	gameID  string
}

func newCollectionsFixture(t *testing.T) *collectionsFixture {
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
	return &collectionsFixture{
		service: NewService(s, time.Now),
		store:   s,
		gameID:  gameID,
	}
}
