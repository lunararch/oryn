package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeNotConfigured(t *testing.T) {
	probe := Probe{Timeout: time.Second}
	if got := probe.Status(context.Background()); got != StatusNotConfigured {
		t.Fatalf("Status() = %q, want %q", got, StatusNotConfigured)
	}
}

func TestProbeAvailableRoot(t *testing.T) {
	root := t.TempDir()
	probe := Probe{Root: root, Timeout: time.Second}
	if got := probe.Status(context.Background()); got != StatusAvailable {
		t.Fatalf("Status() = %q, want %q", got, StatusAvailable)
	}
}

func TestProbeMissingRoot(t *testing.T) {
	probe := Probe{Root: filepath.Join(t.TempDir(), "missing"), Timeout: time.Second}
	if got := probe.Status(context.Background()); got != StatusUnavailable {
		t.Fatalf("Status() = %q, want %q", got, StatusUnavailable)
	}
}

func TestProbeRequiresSentinel(t *testing.T) {
	root := t.TempDir()
	probe := Probe{Root: root, Sentinel: "marker.txt", Timeout: time.Second}
	if got := probe.Status(context.Background()); got != StatusUnavailable {
		t.Fatalf("Status() without sentinel = %q, want %q", got, StatusUnavailable)
	}

	if err := os.WriteFile(filepath.Join(root, "marker.txt"), []byte("present"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if got := probe.Status(context.Background()); got != StatusAvailable {
		t.Fatalf("Status() with sentinel = %q, want %q", got, StatusAvailable)
	}
}

func TestProbeRootIsAFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	probe := Probe{Root: root, Timeout: time.Second}
	if got := probe.Status(context.Background()); got != StatusUnavailable {
		t.Fatalf("Status() = %q, want %q", got, StatusUnavailable)
	}
}
