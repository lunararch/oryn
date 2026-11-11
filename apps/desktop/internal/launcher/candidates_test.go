package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLinuxPrefersNativeBinary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "lib", "linux-x86_64", "ExampleGame"), "\x7fELFbinary", 0o755)
	writeFile(t, filepath.Join(root, "ExampleGame.sh"), "#!/bin/sh\n", 0o755)
	writeFile(t, filepath.Join(root, "ExampleGame.exe"), "MZwindows", 0o644)

	candidates, err := Detect(root, "linux")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}
	if candidates[0].RelativePath != "lib/linux-x86_64/ExampleGame" || candidates[0].Kind != KindBinary {
		t.Errorf("first candidate = %+v", candidates[0])
	}
	for _, candidate := range candidates {
		if candidate.Kind == KindWindowsBinary {
			t.Errorf("Linux detection included a Windows candidate: %+v", candidate)
		}
	}
}

func TestDetectWindowsPrefers64BitAndSkipsInstallers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ExampleGame.exe"), "MZ64", 0o644)
	writeFile(t, filepath.Join(root, "ExampleGame-32.exe"), "MZ32", 0o644)
	writeFile(t, filepath.Join(root, "unins000.exe"), "MZ", 0o644)
	writeFile(t, filepath.Join(root, "python.exe"), "MZ", 0o644)
	writeFile(t, filepath.Join(root, "updater.exe"), "MZ", 0o644)

	candidates, err := Detect(root, "windows")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want only ExampleGame.exe", candidates)
	}
	if candidates[0].RelativePath != "ExampleGame.exe" || candidates[0].Arch != "x86_64" {
		t.Errorf("candidate = %+v", candidates[0])
	}
}

func TestDetectLinuxRejectsNonExecutableScript(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ExampleGame.sh"), "#!/bin/sh\n", 0o644)
	candidates, err := Detect(root, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestDetectLinuxFindsRootELF(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Game"), "\x7fELFbinary", 0o755)
	candidates, err := Detect(root, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].RelativePath != "Game" {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func TestDetectDoesNotTreatArbitraryExecutableAsGame(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Game"), "not an elf", 0o755)
	candidates, err := Detect(root, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v", candidates)
	}
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
