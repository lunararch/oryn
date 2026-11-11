package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kind identifies the type of a launch candidate.
type Kind string

const (
	KindBinary        Kind = "binary"
	KindScript        Kind = "script"
	KindWindowsBinary Kind = "windows-executable"
)

// Candidate is a launch option found inside a game directory.
type Candidate struct {
	RelativePath string
	Kind         Kind
	Arch         string
	Confidence   int
}

var installerMarkers = []string{
	"unins", "installer", "setup", "updater", "update", "crashreport", "crashpad",
	"python", "pythonw", "zsync", "dxwebsetup", "vcredist", "directx", "7za",
	"uninstall", "register", "config",
}

// Detect finds launch candidates for the given platform without executing any
// game content.
func Detect(gameDir, goos string) ([]Candidate, error) {
	entries, err := os.ReadDir(gameDir)
	if err != nil {
		return nil, fmt.Errorf("read game directory: %w", err)
	}
	switch goos {
	case "linux":
		return detectLinux(gameDir, entries), nil
	case "windows":
		return detectWindows(gameDir, entries), nil
	default:
		return nil, fmt.Errorf("unsupported platform %q", goos)
	}
}

func detectLinux(gameDir string, entries []os.DirEntry) []Candidate {
	var candidates []Candidate
	lib, err := os.ReadDir(filepath.Join(gameDir, "lib"))
	if err == nil {
		for _, platform := range lib {
			if !platform.IsDir() {
				continue
			}
			name := strings.ToLower(platform.Name())
			if !strings.HasPrefix(name, "linux-") && !strings.HasPrefix(name, "py3-linux-") {
				continue
			}
			arch := strings.TrimPrefix(strings.TrimPrefix(name, "py3-"), "linux-")
			files, err := os.ReadDir(filepath.Join(gameDir, "lib", platform.Name()))
			if err != nil {
				continue
			}
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				path := filepath.Join(gameDir, "lib", platform.Name(), file.Name())
				if !isELF(path) || !isExecutable(path) {
					continue
				}
				candidates = append(candidates, Candidate{
					RelativePath: filepath.ToSlash(filepath.Join("lib", platform.Name(), file.Name())),
					Kind:         KindBinary,
					Arch:         arch,
					Confidence:   100,
				})
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(gameDir, entry.Name())
		lower := strings.ToLower(entry.Name())
		switch {
		case strings.HasSuffix(lower, ".sh"):
			if isExecutable(path) {
				candidates = append(candidates, Candidate{
					RelativePath: filepath.ToSlash(entry.Name()),
					Kind:         KindScript,
					Arch:         "",
					Confidence:   60,
				})
			}
		case strings.HasSuffix(lower, ".exe"):
			continue
		default:
			if isExecutable(path) && isELF(path) {
				confidence := 90
				if strings.EqualFold(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), filepath.Base(gameDir)) {
					confidence = 95
				}
				candidates = append(candidates, Candidate{
					RelativePath: filepath.ToSlash(entry.Name()),
					Kind:         KindBinary,
					Arch:         "",
					Confidence:   confidence,
				})
			}
		}
	}

	sortCandidates(candidates)
	return candidates
}

func detectWindows(gameDir string, entries []os.DirEntry) []Candidate {
	base := strings.ToLower(filepath.Base(gameDir))
	var candidates []Candidate
	has64 := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if isInstaller(lower) || is32BitName(lower) {
			continue
		}
		has64 = true
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if isInstaller(lower) {
			continue
		}
		if is32BitName(lower) && has64 {
			continue
		}
		stem := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		confidence := 80
		if stem == base || strings.HasPrefix(stem, base) {
			confidence = 100
		}
		arch := "x86_64"
		if is32BitName(lower) {
			arch = "x86"
			confidence -= 10
		}
		candidates = append(candidates, Candidate{
			RelativePath: filepath.ToSlash(entry.Name()),
			Kind:         KindWindowsBinary,
			Arch:         arch,
			Confidence:   confidence,
		})
	}

	sortCandidates(candidates)
	return candidates
}

func sortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence > candidates[j].Confidence
		}
		return candidates[i].RelativePath < candidates[j].RelativePath
	})
}

func isInstaller(name string) bool {
	for _, marker := range installerMarkers {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func is32BitName(name string) bool {
	return strings.Contains(name, "-32.exe") || strings.Contains(name, "_32.exe") ||
		strings.Contains(name, "x86.exe") || strings.Contains(name, "win32.exe")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

func isELF(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var magic [4]byte
	if _, err := file.Read(magic[:]); err != nil {
		return false
	}
	return magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
}
