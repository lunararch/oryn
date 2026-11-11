package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkFindsGamesAndSkipsGameInternals(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, "Game One")
	makeGame(t, root, filepath.Join("Category", "Game Two"))
	writeFile(t, filepath.Join(root, "Category", "Game Two", "game", "nested", "script.rpyc"), "compiled")
	writeFile(t, filepath.Join(root, "not-a-game", "game", "data.bin"), "data")

	result, err := Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 3})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %v", result.Errors)
	}

	paths := make([]string, 0, len(result.Games))
	for _, game := range result.Games {
		paths = append(paths, game.RelPath)
	}
	want := []string{"Category/Game Two", "Game One"}
	if len(paths) != len(want) {
		t.Fatalf("games = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("games[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestWalkSkipsHiddenAndNoiseDirectories(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, filepath.Join(".hidden", "Hidden Game"))
	makeGame(t, root, filepath.Join("System Volume Information", "System Game"))
	makeGame(t, root, filepath.Join("$RECYCLE.BIN", "Recycle Game"))
	makeGame(t, root, filepath.Join("node_modules", "Node Game"))
	makeGame(t, root, "Game")
	writeFile(t, filepath.Join(root, "Game", "saves", "Save Game", "game", "script.rpyc"), "compiled")
	writeFile(t, filepath.Join(root, "Game", "saves", "Save Game", "renpy", "common", "00layout.rpyc"), "runtime")
	writeFile(t, filepath.Join(root, "Game", "saves", "Save Game", "lib", "linux-x86_64", "game"), "binary")

	result, err := Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 4})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Games) != 1 {
		t.Fatalf("games = %v, want only Game", result.Games)
	}
	if result.Games[0].RelPath != "Game" {
		t.Errorf("game = %q, want Game", result.Games[0].RelPath)
	}
}

func TestWalkRespectsMaxDepth(t *testing.T) {
	root := t.TempDir()
	makeGame(t, root, filepath.Join("one", "two", "Deep Game"))

	result, err := Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 2})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Games) != 0 {
		t.Fatalf("games = %v, want none at depth 3 with max 2", result.Games)
	}

	result, err = Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 3})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Games) != 1 {
		t.Fatalf("games = %v, want the depth-3 game", result.Games)
	}
}

func TestWalkDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	makeGame(t, outside, "Outside Game")
	if err := os.Symlink(filepath.Join(outside, "Outside Game"), filepath.Join(root, "Linked Game")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result, err := Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 3})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Games) != 0 {
		t.Fatalf("games = %v, want none through symlink", result.Games)
	}
}

func TestWalkFailsWhenRootIsMissing(t *testing.T) {
	_, err := Walk(context.Background(), WalkOptions{Root: filepath.Join(t.TempDir(), "missing"), MaxDepth: 3})
	if err == nil {
		t.Fatal("Walk() succeeded, want error")
	}
}

func TestWalkCollectsNestedReadErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission test does not work as root")
	}
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o700) })

	result, err := Walk(context.Background(), WalkOptions{Root: root, MaxDepth: 3})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(result.Errors) == 0 {
		t.Fatal("errors = empty, want a permission error")
	}
	if result.Errors[0].Path != "blocked" {
		t.Errorf("error path = %q, want blocked", result.Errors[0].Path)
	}
}

func TestWalkHonoursCancellation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		makeGame(t, root, filepath.Join("games", "Game "+string(rune('A'+i))))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Walk(ctx, WalkOptions{Root: root, MaxDepth: 4})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Walk() error = %v, want context.Canceled", err)
	}
}

func makeGame(t *testing.T, root, rel string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	base := filepath.Base(rel)
	buildName := strings.ReplaceAll(base, " ", "")
	writeFile(t, filepath.Join(dir, "game", "script.rpyc"), "compiled")
	writeFile(t, filepath.Join(dir, "renpy", "common", "00layout.rpyc"), "runtime")
	writeFile(t, filepath.Join(dir, "lib", "linux-x86_64", "game"), "binary")
	writeFile(t, filepath.Join(dir, base+".exe"), "launcher")
	writeFile(t, filepath.Join(dir, "game", "options.rpy"), `define config.name = _("`+base+`")
define build.name = "`+buildName+`"
`)
}
