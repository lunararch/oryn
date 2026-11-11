package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCreatesRestorableDatabase(t *testing.T) {
	s := newMigratedStore(t)
	ctx := context.Background()
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM libraries`).Scan(new(int)); err != nil {
		t.Fatalf("source database is not readable: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "backups", "oryn-backup.db")
	if err := s.Backup(ctx, dest); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	restored, err := Open(dest)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer restored.Close()
	var tables int
	if err := restored.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'games'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 1 {
		t.Errorf("games table count = %d, want 1", tables)
	}
}

func TestBackupRefusesToOverwrite(t *testing.T) {
	s := newMigratedStore(t)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := os.WriteFile(dest, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Backup(context.Background(), dest); err == nil {
		t.Fatal("Backup() overwrote an existing file")
	}
}
