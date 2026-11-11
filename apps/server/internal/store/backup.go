package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Backup writes a consistent SQLite backup using VACUUM INTO. It refuses to
// overwrite an existing file.
func (s *Store) Backup(ctx context.Context, destination string) error {
	if destination == "" {
		return fmt.Errorf("backup destination is required")
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve backup destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	if _, err := os.Stat(abs); err == nil {
		return fmt.Errorf("backup destination already exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check backup destination: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `VACUUM INTO ?`, abs); err != nil {
		return fmt.Errorf("write database backup: %w", err)
	}
	return nil
}
