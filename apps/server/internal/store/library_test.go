package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/library"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

func TestEnsureLibraryCreatesOnce(t *testing.T) {
	s := newMigratedStore(t)
	q := db.New(s.DB)
	root := t.TempDir()
	now := time.Unix(1700000000, 0)

	first, err := library.Ensure(context.Background(), q, root, "", func() time.Time { return now })
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if first.ID == "" {
		t.Fatal("library ID is empty")
	}
	if first.RootPath != root {
		t.Errorf("RootPath = %q, want %q", first.RootPath, root)
	}
	if first.Name != filepath.Base(root) {
		t.Errorf("Name = %q, want %q", first.Name, filepath.Base(root))
	}

	second, err := library.Ensure(context.Background(), q, root, "Custom", func() time.Time { return now.Add(time.Hour) })
	if err != nil {
		t.Fatalf("second Ensure() error = %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("second ID = %q, want %q", second.ID, first.ID)
	}
}

func TestEnsureLibraryRejectsEmptyRoot(t *testing.T) {
	s := newMigratedStore(t)
	_, err := library.Ensure(context.Background(), db.New(s.DB), "", "Library", time.Now)
	if err == nil {
		t.Fatal("Ensure() succeeded, want error")
	}
}

func newMigratedStore(t *testing.T) *Store {
	t.Helper()
	s := openTestStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return s
}
