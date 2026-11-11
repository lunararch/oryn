package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config is non-secret desktop configuration. Credentials belong in the OS
// keyring, never in this file.
type Config struct {
	ServerURL       string            `json:"serverUrl"`
	ClientName      string            `json:"clientName"`
	LibraryMappings map[string]string `json:"libraryMappings"`
	PreferredRunner string            `json:"preferredRunner"`
	LaunchArgs      []string          `json:"launchArgs"`
}

// DefaultPath returns the OS configuration path for Oryn.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, "oryn", "config.json"), nil
}

// Load reads a config file. A missing file returns defaults.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{PreferredRunner: "native"}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.PreferredRunner == "" {
		cfg.PreferredRunner = "native"
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes a private config file atomically.
func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("publish config: %w", err)
	}
	return nil
}

// Validate checks basic config invariants.
func (c Config) Validate() error {
	parsed, err := url.Parse(c.ServerURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("server URL must be an http(s) URL")
	}
	if strings.TrimSpace(c.ClientName) == "" || len(c.ClientName) > 100 {
		return fmt.Errorf("client name is required")
	}
	switch c.PreferredRunner {
	case "native", "wine", "proton":
	default:
		return fmt.Errorf("preferred runner must be native, wine, or proton")
	}
	for libraryID, root := range c.LibraryMappings {
		if strings.TrimSpace(libraryID) == "" || strings.TrimSpace(root) == "" {
			return fmt.Errorf("library mappings require an ID and local root")
		}
	}
	return nil
}
