package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveJoinsInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Game", "game"), 0o750); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveExisting(root, "Game")
	if err != nil {
		t.Fatalf("ResolveExisting() error = %v", err)
	}
	want := filepath.Join(root, "Game")
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestResolveRejectsUnsafeRelativePaths(t *testing.T) {
	root := t.TempDir()
	tests := []string{
		"",
		"..",
		"../outside",
		"Game/../../outside",
		"/etc/passwd",
		`C:\Windows\System32`,
		`\\server\share\game`,
		"Game\x00Name",
	}
	for _, rel := range tests {
		t.Run(rel, func(t *testing.T) {
			if _, err := Resolve(root, rel); err == nil {
				t.Fatalf("Resolve(%q) succeeded", rel)
			}
		})
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolveExisting(root, "escape"); err == nil {
		t.Fatal("ResolveExisting() followed a symlink outside the root")
	}
}

func TestResolveAllowsExistingSymlinkInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "real"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolveExisting(root, "link"); err != nil {
		t.Fatalf("ResolveExisting() error = %v", err)
	}
}

func TestResolveMissingTarget(t *testing.T) {
	root := t.TempDir()
	if _, err := ResolveExisting(root, "Missing"); err == nil {
		t.Fatal("ResolveExisting() accepted a missing target")
	}
}

func TestResolveRejectsEmptyRoot(t *testing.T) {
	if _, err := Resolve("", "Game"); err == nil {
		t.Fatal("Resolve() accepted an empty root")
	}
}
