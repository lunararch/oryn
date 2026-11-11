package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTypicalRenPyGame(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Example Game")
	writeFile(t, filepath.Join(game, "game", "script.rpyc"), "compiled")
	writeFile(t, filepath.Join(game, "game", "options.rpy"), `
define config.name = _("Example Game")
define config.version = "1.2.0"
define build.name = "ExampleGame"
`)
	writeFile(t, filepath.Join(game, "renpy", "common", "00layout.rpyc"), "runtime")
	writeFile(t, filepath.Join(game, "lib", "linux-x86_64", "ExampleGame"), "binary")
	writeFile(t, filepath.Join(game, "ExampleGame.sh"), "#!/bin/sh\n")

	detection, err := Detect(game)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detection.IsGame {
		t.Fatalf("IsGame = false, confidence = %d, evidence = %v", detection.Confidence, detection.Evidence)
	}
	if detection.Confidence < DetectionThreshold {
		t.Errorf("confidence = %d, want >= %d", detection.Confidence, DetectionThreshold)
	}
	if detection.Title != "Example Game" {
		t.Errorf("Title = %q, want Example Game", detection.Title)
	}
	if detection.Version != "1.2.0" {
		t.Errorf("Version = %q, want 1.2.0", detection.Version)
	}
	if detection.BuildName != "ExampleGame" {
		t.Errorf("BuildName = %q, want ExampleGame", detection.BuildName)
	}
}

func TestDetectCompiledOnlyGame(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Compiled")
	writeFile(t, filepath.Join(game, "game", "script.rpyc"), "compiled")
	writeFile(t, filepath.Join(game, "renpy", "common", "00layout.rpyc"), "runtime")
	writeFile(t, filepath.Join(game, "lib", "windows-x86_64", "pythonw.exe"), "binary")

	detection, err := Detect(game)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detection.IsGame {
		t.Fatalf("IsGame = false, confidence = %d, evidence = %v", detection.Confidence, detection.Evidence)
	}
	if detection.Title != "Compiled" {
		t.Errorf("Title = %q, want folder fallback Compiled", detection.Title)
	}
}

func TestDetectRejectsFolderThatOnlyHasGameDirectory(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Not A RenPy Game")
	writeFile(t, filepath.Join(game, "game", "data.bin"), "data")
	writeFile(t, filepath.Join(game, "package.json"), "{}")

	detection, err := Detect(game)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection.IsGame {
		t.Fatalf("IsGame = true, confidence = %d, evidence = %v", detection.Confidence, detection.Evidence)
	}
}

func TestDetectRejectsGameWithoutRenPyEvidence(t *testing.T) {
	root := t.TempDir()
	game := filepath.Join(root, "Loose Script")
	writeFile(t, filepath.Join(game, "game", "script.rpy"), "label start:\n")

	detection, err := Detect(game)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection.IsGame {
		t.Fatalf("IsGame = true, confidence = %d, evidence = %v", detection.Confidence, detection.Evidence)
	}
}

func TestTitleFromFolder(t *testing.T) {
	tests := map[string]string{
		"AboveTheClouds-v0.9-pc":                     "Above The Clouds",
		"BeingADIK-0.11.1-pc-lin":                    "Being ADIK",
		"MySistersDeviousPlot-0.4F95-pc":             "My Sisters Devious Plot",
		"APD0.7.3-animation-pc":                      "APD Animation",
		"LISC-0.11.0-pc":                             "LISC",
		"beauty and thug":                            "Beauty And Thug",
		"TheHeadmaster-0.17.2.1-pc":                  "The Headmaster",
		"University_of_Problems-1.9.0-(Extended)-pc": "University Of Problems (Extended)",
	}
	for input, want := range tests {
		if got := TitleFromFolder(input); got != want {
			t.Errorf("TitleFromFolder(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseOptions(t *testing.T) {
	content := "\xef\xbb\xbf" + `
## This is a comment: define config.name = _("Nope")
define config.name = _("A \"Quoted\" Name")
define config.version = "2.0.1"
define build.name = "QuotedName"
define config.save_directory = "Quoted-123"
define gui.about = _p("""
A game about {b}choices{/b} and consequences.

Second paragraph.
""")
`
	meta, err := parseOptions([]byte(content))
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	if meta.Name != `A "Quoted" Name` {
		t.Errorf("Name = %q", meta.Name)
	}
	if meta.Version != "2.0.1" {
		t.Errorf("Version = %q", meta.Version)
	}
	if meta.BuildName != "QuotedName" {
		t.Errorf("BuildName = %q", meta.BuildName)
	}
	if meta.SaveDirectory != "Quoted-123" {
		t.Errorf("SaveDirectory = %q", meta.SaveDirectory)
	}
	if want := "A game about choices and consequences.\n\nSecond paragraph."; meta.About != want {
		t.Errorf("About = %q, want %q", meta.About, want)
	}
}

func TestParseOptionsIgnoresMalformedInput(t *testing.T) {
	meta, err := parseOptions([]byte(`define config.name = _("unterminated`))
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}
	if meta.Name != "" {
		t.Errorf("Name = %q, want empty", meta.Name)
	}
}

func TestFindArtworkCandidates(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cover.png"), "cover")
	writeFile(t, filepath.Join(root, "icon.png"), "icon")
	writeFile(t, filepath.Join(root, "game", "gui", "main_menu.png"), "menu")
	writeFile(t, filepath.Join(root, "game", "gui", "window_icon.png"), "icon")

	candidates, err := FindArtworkCandidates(root)
	if err != nil {
		t.Fatalf("FindArtworkCandidates() error = %v", err)
	}
	if len(candidates) < 3 {
		t.Fatalf("candidates = %v, want at least 3", candidates)
	}
	if candidates[0].RelPath != "cover.png" || candidates[0].Kind != ArtworkCover {
		t.Errorf("first candidate = %+v, want root cover", candidates[0])
	}
	if candidates[1].RelPath != filepath.Join("game", "gui", "main_menu.png") || candidates[1].Kind != ArtworkBanner {
		t.Errorf("second candidate = %+v, want main menu banner", candidates[1])
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
