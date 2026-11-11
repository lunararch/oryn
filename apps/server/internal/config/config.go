package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the server configuration loaded from the environment.
type Config struct {
	HTTPAddr        string
	DataDir         string
	DBPath          string
	LibraryRoot     string
	LibrarySentinel string
	WebDir          string
	ScanInterval    time.Duration
	ScanMaxDepth    int
	ScanWorkers     int
	LogLevel        slog.Level
}

// Load reads configuration from getenv. It does not touch the filesystem.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := Config{
		HTTPAddr:     "127.0.0.1:8080",
		DataDir:      "./data",
		ScanInterval: 0,
		ScanMaxDepth: 3,
		ScanWorkers:  2,
		LogLevel:     slog.LevelInfo,
	}

	if v := strings.TrimSpace(getenv("ORYN_HTTP_ADDR")); v != "" {
		cfg.HTTPAddr = v
	}
	if v := strings.TrimSpace(getenv("ORYN_DATA_DIR")); v != "" {
		cfg.DataDir = v
	}
	cfg.DBPath = filepath.Join(cfg.DataDir, "oryn.db")
	if v := strings.TrimSpace(getenv("ORYN_DB_PATH")); v != "" {
		cfg.DBPath = v
	}
	cfg.LibraryRoot = strings.TrimSpace(getenv("ORYN_LIBRARY_ROOT"))
	cfg.LibrarySentinel = strings.TrimSpace(getenv("ORYN_LIBRARY_SENTINEL"))
	cfg.WebDir = strings.TrimSpace(getenv("ORYN_WEB_DIR"))

	if v := strings.TrimSpace(getenv("ORYN_SCAN_INTERVAL")); v != "" {
		interval, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("ORYN_SCAN_INTERVAL: %w", err)
		}
		if interval < 0 {
			return Config{}, fmt.Errorf("ORYN_SCAN_INTERVAL must not be negative")
		}
		cfg.ScanInterval = interval
	}

	if v := strings.TrimSpace(getenv("ORYN_SCAN_MAX_DEPTH")); v != "" {
		depth, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("ORYN_SCAN_MAX_DEPTH: %w", err)
		}
		if depth < 1 || depth > 20 {
			return Config{}, fmt.Errorf("ORYN_SCAN_MAX_DEPTH must be between 1 and 20")
		}
		cfg.ScanMaxDepth = depth
	}

	if v := strings.TrimSpace(getenv("ORYN_SCAN_WORKERS")); v != "" {
		workers, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("ORYN_SCAN_WORKERS: %w", err)
		}
		if workers < 1 || workers > 16 {
			return Config{}, fmt.Errorf("ORYN_SCAN_WORKERS must be between 1 and 16")
		}
		cfg.ScanWorkers = workers
	}

	if v := strings.TrimSpace(getenv("ORYN_LOG_LEVEL")); v != "" {
		level, err := parseLogLevel(v)
		if err != nil {
			return Config{}, err
		}
		cfg.LogLevel = level
	}

	if cfg.LibraryRoot != "" {
		inside, err := pathWithin(cfg.LibraryRoot, cfg.DBPath)
		if err != nil {
			return Config{}, fmt.Errorf("resolve data paths: %w", err)
		}
		if inside {
			return Config{}, fmt.Errorf("database path must be outside the game library root")
		}
	}

	return cfg, nil
}

// LoadFromEnv reads configuration from the process environment.
func LoadFromEnv() (Config, error) {
	return Load(os.Getenv)
}

// SentinelPath returns the configured sentinel path, resolved relative to the
// library root when it is not absolute. An empty string means no sentinel.
func (c Config) SentinelPath() string {
	if c.LibrarySentinel == "" {
		return ""
	}
	if filepath.IsAbs(c.LibrarySentinel) {
		return filepath.Clean(c.LibrarySentinel)
	}
	return filepath.Join(c.LibraryRoot, c.LibrarySentinel)
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("ORYN_LOG_LEVEL must be one of debug, info, warn, error")
	}
}

func pathWithin(root, target string) (bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(filepath.Clean(absRoot), filepath.Clean(absTarget))
	if err != nil {
		return false, err
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))), nil
}
