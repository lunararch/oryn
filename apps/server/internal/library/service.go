package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

// Ensure returns the library registered for root, creating it when needed.
func Ensure(ctx context.Context, q *db.Queries, root, name string, now func() time.Time) (db.Library, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return db.Library{}, fmt.Errorf("library root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return db.Library{}, fmt.Errorf("resolve library root: %w", err)
	}
	if now == nil {
		now = time.Now
	}

	existing, err := q.GetLibraryByRoot(ctx, absRoot)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.Library{}, fmt.Errorf("look up library: %w", err)
	}

	if strings.TrimSpace(name) == "" {
		name = filepath.Base(absRoot)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return db.Library{}, fmt.Errorf("generate library ID: %w", err)
	}
	ts := now().Unix()
	row := db.Library{
		ID:        id.String(),
		Name:      name,
		RootPath:  absRoot,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	if err := q.CreateLibrary(ctx, db.CreateLibraryParams{
		ID:        row.ID,
		Name:      row.Name,
		RootPath:  row.RootPath,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}); err != nil {
		// Another startup may have created it concurrently.
		if existing, lookupErr := q.GetLibraryByRoot(ctx, absRoot); lookupErr == nil {
			return existing, nil
		}
		return db.Library{}, fmt.Errorf("create library: %w", err)
	}
	return row, nil
}
