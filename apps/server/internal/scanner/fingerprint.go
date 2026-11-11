package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Fingerprint produces a stable identity for move/rename detection. It uses
// build metadata and launcher names that survive a folder rename, and only
// falls back to the display title when no stronger signal exists.
func Fingerprint(d Detection) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(d.BuildName)),
		strings.ToLower(strings.TrimSpace(d.SaveDirectory)),
	}
	names := append([]string(nil), d.LaunchBaseNames...)
	for i := range names {
		names[i] = strings.ToLower(strings.TrimSpace(names[i]))
	}
	sort.Strings(names)
	parts = append(parts, names...)

	var identity []string
	for _, part := range parts {
		if part != "" && part != "|" {
			identity = append(identity, part)
		}
	}
	if len(identity) == 0 {
		title := strings.ToLower(strings.TrimSpace(d.Title))
		if title == "" {
			return ""
		}
		identity = []string{title}
	}

	sum := sha256.Sum256([]byte(strings.Join(identity, "\x00")))
	return hex.EncodeToString(sum[:])
}
