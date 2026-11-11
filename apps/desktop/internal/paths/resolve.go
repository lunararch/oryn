package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrInvalidPath is returned when a server-relative path is unsafe.
var ErrInvalidPath = errors.New("invalid game path")

var drivePrefix = regexp.MustCompile(`^[A-Za-z]:`)

// Resolve joins a server-relative path to a local root and verifies lexical
// containment. It does not require the target to exist.
func Resolve(root, relative string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("%w: local library root is required", ErrInvalidPath)
	}
	if relative == "" || strings.TrimSpace(relative) == "" {
		return "", fmt.Errorf("%w: path is empty", ErrInvalidPath)
	}
	if strings.ContainsRune(relative, '\x00') {
		return "", fmt.Errorf("%w: path contains a null byte", ErrInvalidPath)
	}
	if strings.Contains(relative, `\`) {
		return "", fmt.Errorf("%w: path must use forward slashes", ErrInvalidPath)
	}
	if drivePrefix.MatchString(relative) || filepath.IsAbs(relative) || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("%w: absolute paths are not allowed", ErrInvalidPath)
	}

	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%w: path contains an invalid segment", ErrInvalidPath)
		}
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: resolve root: %v", ErrInvalidPath, err)
	}
	target := filepath.Join(absRoot, filepath.FromSlash(relative))
	if !within(absRoot, target) {
		return "", fmt.Errorf("%w: target escapes the library root", ErrInvalidPath)
	}
	return target, nil
}

// ResolveExisting resolves a path and rejects missing targets or symlink
// escapes. The final target must remain beneath the configured root.
func ResolveExisting(root, relative string) (string, error) {
	target, err := Resolve(root, relative)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("%w: resolve root symlinks: %v", ErrInvalidPath, err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("%w: target does not exist or cannot be resolved", ErrInvalidPath)
	}
	if !within(resolvedRoot, resolvedTarget) {
		return "", fmt.Errorf("%w: target escapes the library root through a symlink", ErrInvalidPath)
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("%w: target is not accessible", ErrInvalidPath)
	}
	if !info.Mode().IsRegular() && !info.IsDir() {
		return "", fmt.Errorf("%w: target is not a regular file or directory", ErrInvalidPath)
	}
	return resolvedTarget, nil
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
