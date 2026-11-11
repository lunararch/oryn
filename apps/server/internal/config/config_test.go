package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:8080")
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./data")
	}
	if cfg.DBPath != filepath.Join("./data", "oryn.db") {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, filepath.Join("./data", "oryn.db"))
	}
	if cfg.ScanInterval != 0 {
		t.Errorf("ScanInterval = %v, want 0", cfg.ScanInterval)
	}
	if cfg.ScanMaxDepth != 3 {
		t.Errorf("ScanMaxDepth = %d, want 3", cfg.ScanMaxDepth)
	}
	if cfg.ScanWorkers != 2 {
		t.Errorf("ScanWorkers = %d, want 2", cfg.ScanWorkers)
	}
	if cfg.LogLevel.String() != "INFO" {
		t.Errorf("LogLevel = %s, want INFO", cfg.LogLevel)
	}
}

func TestLoadOverrides(t *testing.T) {
	values := map[string]string{
		"ORYN_HTTP_ADDR":        "0.0.0.0:9000",
		"ORYN_DATA_DIR":         "/srv/oryn",
		"ORYN_DB_PATH":          "/srv/oryn/library.db",
		"ORYN_LIBRARY_ROOT":     "/games",
		"ORYN_LIBRARY_SENTINEL": "games",
		"ORYN_SCAN_INTERVAL":    "6h",
		"ORYN_SCAN_MAX_DEPTH":   "5",
		"ORYN_SCAN_WORKERS":     "4",
		"ORYN_LOG_LEVEL":        "debug",
	}
	cfg, err := Load(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != "0.0.0.0:9000" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.DBPath != "/srv/oryn/library.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.ScanInterval != 6*time.Hour {
		t.Errorf("ScanInterval = %v", cfg.ScanInterval)
	}
	if cfg.ScanMaxDepth != 5 {
		t.Errorf("ScanMaxDepth = %d", cfg.ScanMaxDepth)
	}
	if cfg.ScanWorkers != 4 {
		t.Errorf("ScanWorkers = %d", cfg.ScanWorkers)
	}
	if cfg.LogLevel.String() != "DEBUG" {
		t.Errorf("LogLevel = %s", cfg.LogLevel)
	}
	if got, want := cfg.SentinelPath(), "/games/games"; got != want {
		t.Errorf("SentinelPath() = %q, want %q", got, want)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "invalid interval", key: "ORYN_SCAN_INTERVAL", value: "soon"},
		{name: "negative interval", key: "ORYN_SCAN_INTERVAL", value: "-1h"},
		{name: "depth too low", key: "ORYN_SCAN_MAX_DEPTH", value: "0"},
		{name: "depth too high", key: "ORYN_SCAN_MAX_DEPTH", value: "100"},
		{name: "workers too low", key: "ORYN_SCAN_WORKERS", value: "0"},
		{name: "invalid log level", key: "ORYN_LOG_LEVEL", value: "verbose"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return ""
			})
			if err == nil {
				t.Fatalf("Load() succeeded, want error for %s=%q", tt.key, tt.value)
			}
		})
	}
}

func TestLoadRejectsDataInsideLibraryRoot(t *testing.T) {
	library := t.TempDir()
	_, err := Load(func(key string) string {
		switch key {
		case "ORYN_LIBRARY_ROOT":
			return library
		case "ORYN_DATA_DIR":
			return filepath.Join(library, "oryn-data")
		case "ORYN_DB_PATH":
			return filepath.Join(library, "oryn-data", "oryn.db")
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("Load() succeeded, want error when application data is inside the game share")
	}
}
