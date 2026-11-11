package scanner

import (
	"path/filepath"
	"testing"
)

func TestFindArtworkCandidatesRecognizesCommonMenuAndCoverNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "game", "gui", "MainMenu_Background.png"), "menu")
	writeFile(t, filepath.Join(root, "game", "gui", "Oblojka.jpg"), "cover")

	candidates, err := FindArtworkCandidates(root)
	if err != nil {
		t.Fatalf("FindArtworkCandidates() error = %v", err)
	}
	foundBanner := false
	foundCover := false
	for _, candidate := range candidates {
		if candidate.RelPath == "game/gui/MainMenu_Background.png" && candidate.Kind == ArtworkBanner {
			foundBanner = true
		}
		if candidate.RelPath == "game/gui/Oblojka.jpg" && candidate.Kind == ArtworkBanner {
			foundCover = true
		}
	}
	if !foundBanner {
		t.Errorf("MainMenu_Background.png not found as a banner: %+v", candidates)
	}
	if !foundCover {
		t.Errorf("Oblojka.jpg not found as a cover: %+v", candidates)
	}
}
