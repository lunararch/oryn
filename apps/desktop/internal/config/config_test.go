package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Config{
		ServerURL:       "http://pinas:8080",
		ClientName:      "CachyOS Desktop",
		LibraryMappings: map[string]string{"library-1": "/mnt/games/visual-novels"},
		PreferredRunner: "native",
		LaunchArgs:      []string{"--windowed"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("config permissions = %o, want private", info.Mode().Perm())
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ServerURL != cfg.ServerURL || loaded.ClientName != cfg.ClientName {
		t.Errorf("loaded = %+v", loaded)
	}
	if loaded.LibraryMappings["library-1"] != "/mnt/games/visual-novels" {
		t.Errorf("mapping = %v", loaded.LibraryMappings)
	}
}

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PreferredRunner != "native" {
		t.Errorf("runner = %q", cfg.PreferredRunner)
	}
}

func TestValidateRejectsBadValues(t *testing.T) {
	tests := []Config{
		{ServerURL: "ftp://pinas", ClientName: "Desktop", PreferredRunner: "native"},
		{ServerURL: "http://pinas", ClientName: "", PreferredRunner: "native"},
		{ServerURL: "http://pinas", ClientName: "Desktop", PreferredRunner: "unknown"},
		{ServerURL: "http://pinas", ClientName: "Desktop", PreferredRunner: "native", LibraryMappings: map[string]string{"": "/games"}},
	}
	for _, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() accepted %+v", cfg)
		}
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("path = %q", path)
	}
}
