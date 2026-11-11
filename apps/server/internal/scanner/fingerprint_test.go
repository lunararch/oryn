package scanner

import "testing"

func TestFingerprintIsStableAcrossFolderTitleAndVersion(t *testing.T) {
	base := Detection{
		Title:           "Old Title",
		Version:         "1.0",
		BuildName:       "ExampleGame",
		SaveDirectory:   "ExampleGame-123",
		LaunchBaseNames: []string{"examplegame"},
	}
	renamed := base
	renamed.Title = "New Title"
	renamed.Version = "2.0"

	if Fingerprint(base) != Fingerprint(renamed) {
		t.Fatal("fingerprint changed when only title/version changed")
	}
	if Fingerprint(base) == "" {
		t.Fatal("fingerprint is empty")
	}
}

func TestFingerprintDiffersForDifferentGames(t *testing.T) {
	a := Detection{BuildName: "GameA", SaveDirectory: "SaveA", LaunchBaseNames: []string{"gamea"}}
	b := Detection{BuildName: "GameB", SaveDirectory: "SaveB", LaunchBaseNames: []string{"gameb"}}
	if Fingerprint(a) == Fingerprint(b) {
		t.Fatal("different games produced the same fingerprint")
	}
}

func TestFingerprintFallsBackToTitle(t *testing.T) {
	a := Detection{Title: "A Game"}
	b := Detection{Title: "A Game"}
	if Fingerprint(a) == "" || Fingerprint(a) != Fingerprint(b) {
		t.Fatalf("fallback fingerprints = %q, %q", Fingerprint(a), Fingerprint(b))
	}
}
