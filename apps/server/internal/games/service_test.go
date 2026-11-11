package games

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func TestListSearchesSortsFiltersAndPaginates(t *testing.T) {
	f := newGamesFixture(t)
	f.insertGame(t, "Zeta", "A space adventure", false, "")
	f.insertGame(t, "Alpha", "A romantic story", true, "")

	result, err := f.service.List(context.Background(), ListParams{
		LibraryID: f.library.ID,
		Query:     "romantic",
		Page:      1,
		PageSize:  10,
		Sort:      "title",
		Order:     "asc",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = %+v, want one game", result)
	}
	if result.Items[0].Title != "Alpha" || !result.Items[0].Favorite {
		t.Errorf("item = %+v", result.Items[0])
	}

	favorite := true
	result, err = f.service.List(context.Background(), ListParams{
		LibraryID: f.library.ID,
		Favorite:  &favorite,
		Page:      1,
		PageSize:  10,
		Sort:      "title",
		Order:     "asc",
	})
	if err != nil {
		t.Fatalf("favorite List() error = %v", err)
	}
	if result.Total != 1 || result.Items[0].Title != "Alpha" {
		t.Errorf("favorite result = %+v", result)
	}

	result, err = f.service.List(context.Background(), ListParams{
		LibraryID: f.library.ID,
		Page:      1,
		PageSize:  1,
		Sort:      "title",
		Order:     "asc",
	})
	if err != nil {
		t.Fatalf("paged List() error = %v", err)
	}
	if result.Total != 2 || len(result.Items) != 1 || result.TotalPages != 2 {
		t.Errorf("paged result = %+v", result)
	}
}

func TestListUsesOverrideTitleAndCompletion(t *testing.T) {
	f := newGamesFixture(t)
	game := f.insertGame(t, "Detected", "", false, "")
	if err := f.queries.UpsertGameOverride(context.Background(), db.UpsertGameOverrideParams{
		GameID:           game.ID,
		Title:            sql.NullString{String: "Override", Valid: true},
		CompletionStatus: sql.NullString{String: "completed", Valid: true},
		UpdatedAt:        time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	result, err := f.service.List(context.Background(), ListParams{
		LibraryID:  f.library.ID,
		Completion: "completed",
		Page:       1,
		PageSize:   10,
		Sort:       "title",
		Order:      "asc",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 || result.Items[0].Title != "Override" {
		t.Errorf("result = %+v", result)
	}
}

func TestGetReturnsResolvedMetadata(t *testing.T) {
	f := newGamesFixture(t)
	game := f.insertGame(t, "Detected", "Detected description", true, "")
	if err := f.queries.UpsertGameOverride(context.Background(), db.UpsertGameOverrideParams{
		GameID:      game.ID,
		Title:       sql.NullString{String: "Custom", Valid: true},
		Description: sql.NullString{String: "Custom description", Valid: true},
		Rating:      sql.NullInt64{Int64: 8, Valid: true},
		UpdatedAt:   time.Now().Unix(),
	}); err != nil {
		t.Fatal(err)
	}

	detail, err := f.service.Get(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if detail.Title != "Custom" || detail.Description != "Custom description" {
		t.Errorf("detail = %+v", detail)
	}
	if detail.Rating == nil || *detail.Rating != 8 {
		t.Errorf("rating = %v", detail.Rating)
	}
	if !detail.Favorite {
		t.Error("favorite = false")
	}
}

func TestUpdateMetadataValidatesAndClears(t *testing.T) {
	f := newGamesFixture(t)
	game := f.insertGame(t, "Detected", "", false, "")

	title := "  Custom Title  "
	rating := 9
	completion := "playing"
	detail, err := f.service.UpdateMetadata(context.Background(), game.ID, MetadataPatch{
		Title:            Field[string]{Set: true, Value: &title},
		Rating:           Field[int]{Set: true, Value: &rating},
		CompletionStatus: Field[string]{Set: true, Value: &completion},
	})
	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}
	if detail.Title != "Custom Title" || detail.Rating == nil || *detail.Rating != 9 {
		t.Errorf("detail = %+v", detail)
	}
	if detail.CompletionStatus != "playing" {
		t.Errorf("completion = %q", detail.CompletionStatus)
	}

	empty := ""
	detail, err = f.service.UpdateMetadata(context.Background(), game.ID, MetadataPatch{
		Title: Field[string]{Set: true, Value: &empty},
	})
	if err != nil {
		t.Fatalf("clear UpdateMetadata() error = %v", err)
	}
	if detail.Title != "Detected" {
		t.Errorf("title = %q, want detected fallback", detail.Title)
	}

	badRating := 11
	if _, err := f.service.UpdateMetadata(context.Background(), game.ID, MetadataPatch{
		Rating: Field[int]{Set: true, Value: &badRating},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}

	badCompletion := "finished"
	if _, err := f.service.UpdateMetadata(context.Background(), game.ID, MetadataPatch{
		CompletionStatus: Field[string]{Set: true, Value: &badCompletion},
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}

func TestFavoritesAreIdempotent(t *testing.T) {
	f := newGamesFixture(t)
	game := f.insertGame(t, "Alpha", "", false, "")

	if err := f.service.SetFavorite(context.Background(), game.ID, true); err != nil {
		t.Fatalf("SetFavorite() error = %v", err)
	}
	if err := f.service.SetFavorite(context.Background(), game.ID, true); err != nil {
		t.Fatalf("second SetFavorite() error = %v", err)
	}
	detail, err := f.service.Get(context.Background(), game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Favorite {
		t.Error("favorite = false")
	}

	if err := f.service.SetFavorite(context.Background(), game.ID, false); err != nil {
		t.Fatalf("unset favorite: %v", err)
	}
	if err := f.service.SetFavorite(context.Background(), game.ID, false); err != nil {
		t.Fatalf("second unset favorite: %v", err)
	}
}

func TestGetMissingGameReturnsNotFound(t *testing.T) {
	f := newGamesFixture(t)
	if _, err := f.service.Get(context.Background(), uuid.Must(uuid.NewV7()).String()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

type gamesFixture struct {
	service *Service
	queries *db.Queries
	library db.Library
}

func newGamesFixture(t *testing.T) *gamesFixture {
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
		t.Fatalf("Ensure() error = %v", err)
	}
	return &gamesFixture{
		service: NewService(s, time.Now),
		queries: q,
		library: lib,
	}
}

func (f *gamesFixture) insertGame(t *testing.T, title, description string, favorite bool, completion string) db.Game {
	t.Helper()
	now := time.Now().Unix()
	game := db.Game{
		ID:                  uuid.Must(uuid.NewV7()).String(),
		LibraryID:           f.library.ID,
		RelativePath:        title,
		FolderName:          title,
		DetectedTitle:       title,
		DetectedDescription: description,
		AvailabilityStatus:  "available",
		FirstSeenAt:         now,
		LastSeenAt:          now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := f.queries.InsertGame(context.Background(), db.InsertGameParams{
		ID:                    game.ID,
		LibraryID:             game.LibraryID,
		RelativePath:          game.RelativePath,
		FolderName:            game.FolderName,
		DetectedTitle:         game.DetectedTitle,
		DetectedDescription:   game.DetectedDescription,
		DetectedDeveloper:     "",
		DetectedPublisher:     "",
		DetectedReleaseDate:   "",
		DetectedVersion:       "",
		DetectionConfidence:   10,
		DetectionEvidence:     "[]",
		AvailabilityStatus:    "available",
		FilesystemFingerprint: title,
		FirstSeenAt:           now,
		LastSeenAt:            now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("InsertGame() error = %v", err)
	}
	if favorite {
		if err := f.queries.SetFavorite(context.Background(), db.SetFavoriteParams{
			GameID:    game.ID,
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("SetFavorite() error = %v", err)
		}
	}
	if completion != "" {
		if err := f.queries.UpsertGameOverride(context.Background(), db.UpsertGameOverrideParams{
			GameID:           game.ID,
			CompletionStatus: sql.NullString{String: completion, Valid: true},
			UpdatedAt:        now,
		}); err != nil {
			t.Fatalf("UpsertGameOverride() error = %v", err)
		}
	}
	return game
}
