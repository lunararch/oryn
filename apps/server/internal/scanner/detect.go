package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// DetectionThreshold is the minimum confidence required to index a directory
// as a Ren'Py game.
const DetectionThreshold = 5

// Detection contains the evidence and conservative metadata found for one
// candidate game directory.
type Detection struct {
	IsGame          bool
	Confidence      int
	Evidence        []string
	Title           string
	About           string
	Version         string
	BuildName       string
	SaveDirectory   string
	LaunchBaseNames []string
}

type optionsMetadata struct {
	Name          string
	About         string
	Version       string
	BuildName     string
	SaveDirectory string
}

// Detect inspects a candidate game directory without executing any content.
func Detect(dir string) (Detection, error) {
	detection := Detection{
		Title: TitleFromFolder(filepath.Base(dir)),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return Detection{}, fmt.Errorf("read game directory: %w", err)
	}

	names := make(map[string]bool, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = true
	}

	if names["game"] {
		detection.addEvidence("game directory", 2)
	}

	if names["renpy"] {
		detection.addEvidence("Ren'Py runtime directory", 3)
		if dirExists(filepath.Join(dir, "renpy", "common")) {
			detection.addEvidence("Ren'Py common runtime", 1)
		}
	}

	if names["lib"] {
		if hasRenPyLib(dir) {
			detection.addEvidence("Ren'Py platform libraries", 2)
		}
	}

	if names["game"] {
		rpy, rpyc, err := countScripts(filepath.Join(dir, "game"))
		if err != nil {
			return Detection{}, err
		}
		if rpyc > 0 {
			detection.addEvidence("compiled Ren'Py scripts", 3)
		}
		if rpy > 0 {
			detection.addEvidence("Ren'Py source scripts", 2)
		}
	}

	if hasLauncher(entries) {
		detection.addEvidence("game launcher", 1)
	}
	detection.LaunchBaseNames = launcherBaseNames(entries)

	detection.IsGame = detection.Confidence >= DetectionThreshold
	if !detection.IsGame {
		return detection, nil
	}

	optionsPath := filepath.Join(dir, "game", "options.rpy")
	if data, err := os.ReadFile(optionsPath); err == nil {
		if meta, err := parseOptions(data); err == nil {
			if meta.Name != "" {
				detection.Title = meta.Name
			}
			detection.About = meta.About
			detection.Version = meta.Version
			detection.BuildName = meta.BuildName
			detection.SaveDirectory = meta.SaveDirectory
		}
	}

	return detection, nil
}

func (d *Detection) addEvidence(label string, weight int) {
	d.Evidence = append(d.Evidence, label)
	d.Confidence += weight
}

func countScripts(gameDir string) (rpy, rpyc int, err error) {
	entries, err := os.ReadDir(gameDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read game scripts: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".rpy":
			rpy++
		case ".rpyc":
			rpyc++
		}
	}
	return rpy, rpyc, nil
}

func hasRenPyLib(dir string) bool {
	entries, err := os.ReadDir(filepath.Join(dir, "lib"))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, prefix := range []string{"windows-", "linux-", "mac-", "py2-", "py3-"} {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
	}
	return false
}

func hasLauncher(entries []os.DirEntry) bool {
	return len(launcherBaseNames(entries)) > 0
}

func launcherBaseNames(entries []os.DirEntry) []string {
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".sh" && ext != ".exe" {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		lower := strings.ToLower(base)
		if strings.HasPrefix(lower, "unins") || strings.Contains(lower, "installer") ||
			strings.Contains(lower, "updater") || strings.Contains(lower, "crashreport") ||
			lower == "python" || lower == "pythonw" || lower == "zsync" {
			continue
		}
		names = append(names, strings.ToLower(base))
	}
	return names
}

func parseOptions(data []byte) (optionsMetadata, error) {
	content := strings.TrimPrefix(string(data), "\ufeff")

	var meta optionsMetadata
	meta.Name = extractString(content, `config\.name`)
	meta.Version = extractString(content, `config\.version`)
	meta.BuildName = extractString(content, `build\.name`)
	meta.SaveDirectory = extractString(content, `config\.save_directory`)
	meta.About = extractAbout(content)

	return meta, nil
}

func extractString(content, key string) string {
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(fmt.Sprintf(`(?m)^\s*define\s+`+`%s`+`\s*=\s*_?\(\s*"((?:\\.|[^"\\])*)"\s*\)`, key)),
		regexp.MustCompile(fmt.Sprintf(`(?m)^\s*define\s+`+`%s`+`\s*=\s*"((?:\\.|[^"\\])*)"`, key)),
	} {
		match := pattern.FindStringSubmatch(content)
		if len(match) < 2 {
			continue
		}
		if value, err := strconv.Unquote(`"` + match[1] + `"`); err == nil {
			return strings.TrimSpace(value)
		}
		return strings.TrimSpace(match[1])
	}
	return ""
}

var aboutAssignment = regexp.MustCompile(`(?ms)^\s*define\s+gui\.about\s*=\s*_p?\(\s*"""(.*?)"""\s*\)`)

func extractAbout(content string) string {
	match := aboutAssignment.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	text := match[1]
	text = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(text, "")
	text = unescapeRenPy(text)
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func unescapeRenPy(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}
		switch value[i+1] {
		case 'n':
			builder.WriteByte('\n')
			i++
		case 't':
			builder.WriteByte('\t')
			i++
		case '"', '\\', '\'':
			builder.WriteByte(value[i+1])
			i++
		default:
			builder.WriteByte(value[i])
		}
	}
	return builder.String()
}

// TitleFromFolder produces a readable fallback title while leaving the raw
// folder name untouched in the database.
func TitleFromFolder(name string) string {
	title := strings.TrimSpace(name)
	if title == "" {
		return "Untitled"
	}

	title = stripPackagingSuffixes(title)
	title = stripVersionParts(title)
	title = strings.NewReplacer("_", " ", "-", " ").Replace(title)
	title = splitCamelCase(title)
	title = strings.Join(strings.Fields(title), " ")
	return titleCaseWords(title)
}

func stripPackagingSuffixes(title string) string {
	for {
		lower := strings.ToLower(title)
		changed := false
		for _, suffix := range []string{"-pc", "-win", "-lin", "-linux", "-mac", "-windows", "-32", "-64"} {
			if strings.HasSuffix(lower, suffix) {
				title = title[:len(title)-len(suffix)]
				changed = true
				break
			}
		}
		if !changed {
			return strings.Trim(title, " -_")
		}
	}
}

var (
	attachedVersion = regexp.MustCompile(`([A-Za-z])\d+(?:\.\d+)+[A-Za-z0-9]*`)
	versionSegment  = regexp.MustCompile(`(?i)[-_ ]v?\d+(?:\.\d+)+(?:[a-z0-9-]*)?`)
)

func stripVersionParts(title string) string {
	title = versionSegment.ReplaceAllString(title, " ")
	title = attachedVersion.ReplaceAllString(title, "$1")
	return strings.Trim(title, " -_")
}

func splitCamelCase(value string) string {
	var out strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				out.WriteRune(' ')
			}
		}
		out.WriteRune(r)
	}
	return out.String()
}

func titleCaseWords(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// ArtworkKind identifies the role of an artwork candidate.
type ArtworkKind string

const (
	ArtworkCover  ArtworkKind = "cover"
	ArtworkBanner ArtworkKind = "banner"
)

// ArtworkCandidate is a file that may be suitable for a game's artwork.
type ArtworkCandidate struct {
	Kind     ArtworkKind
	RelPath  string
	Priority int
}

var imageExtensions = []string{".png", ".jpg", ".jpeg", ".webp"}

// FindArtworkCandidates returns possible cover and banner files in priority
// order. It only lists candidates; content validation happens separately.
func FindArtworkCandidates(gameDir string) ([]ArtworkCandidate, error) {
	var candidates []ArtworkCandidate
	addIfPresent := func(kind ArtworkKind, relBase string, priority int) {
		if path, ok := findImage(gameDir, relBase); ok {
			candidates = append(candidates, ArtworkCandidate{Kind: kind, RelPath: path, Priority: priority})
		}
	}

	addIfPresent(ArtworkCover, "cover", 100)
	addIfPresent(ArtworkCover, "game/cover", 90)
	addIfPresent(ArtworkCover, "game/gui/cover", 80)
	addIfPresent(ArtworkBanner, "banner", 75)
	addIfPresent(ArtworkBanner, "game/banner", 70)
	addIfPresent(ArtworkBanner, "game/gui/main_menu", 60)
	addIfPresent(ArtworkBanner, "game/gui/mainmenu", 55)
	addIfPresent(ArtworkBanner, "game/gui/mainmenubackground", 54)
	addIfPresent(ArtworkBanner, "game/gui/mainmenu_background", 54)
	addIfPresent(ArtworkBanner, "game/gui/menu_background", 53)
	addIfPresent(ArtworkBanner, "game/gui/main_menu1", 52)
	addIfPresent(ArtworkBanner, "game/gui/main_menu2", 51)
	addIfPresent(ArtworkBanner, "game/gui/game_menu", 50)
	addIfPresent(ArtworkBanner, "game/gui/oblojka", 46)
	addIfPresent(ArtworkBanner, "game/oblojka", 45)
	addIfPresent(ArtworkCover, "icon", 40)
	addIfPresent(ArtworkCover, "game/gui/window_icon", 30)

	seen := make(map[string]bool, len(candidates))
	out := candidates[:0]
	for _, candidate := range candidates {
		if seen[candidate.RelPath] {
			continue
		}
		seen[candidate.RelPath] = true
		out = append(out, candidate)
	}
	return out, nil
}

func findImage(gameDir, relBase string) (string, bool) {
	dir := filepath.Dir(filepath.FromSlash(relBase))
	base := strings.ToLower(filepath.Base(relBase))
	entries, err := os.ReadDir(filepath.Join(gameDir, dir))
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, ext := range imageExtensions {
			if name == base+ext {
				return pathToSlash(filepath.Join(dir, entry.Name())), true
			}
		}
	}
	return "", false
}

func pathToSlash(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "/")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
