package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WalkOptions controls a filesystem discovery pass.
type WalkOptions struct {
	Root     string
	MaxDepth int
}

// DiscoveredGame is a candidate directory and its detection result.
type DiscoveredGame struct {
	RelPath   string
	Detection Detection
}

// WalkError describes a recoverable error encountered below the root.
type WalkError struct {
	Path    string
	Kind    string
	Message string
}

// WalkResult contains all discovered candidates and recoverable errors.
type WalkResult struct {
	Games  []DiscoveredGame
	Errors []WalkError
}

var skippedDirectoryNames = map[string]bool{
	".git":                      true,
	".svn":                      true,
	"__macosx":                  true,
	"$recycle.bin":              true,
	"system volume information": true,
	"node_modules":              true,
}

// Walk recursively discovers likely Ren'Py games without executing content or
// following directory symlinks. A game directory is not descended into after
// it is identified.
func Walk(ctx context.Context, opts WalkOptions) (WalkResult, error) {
	if opts.Root == "" {
		return WalkResult{}, fmt.Errorf("library root is required")
	}
	if opts.MaxDepth < 1 {
		opts.MaxDepth = 3
	}

	info, err := os.Stat(opts.Root)
	if err != nil {
		return WalkResult{}, fmt.Errorf("stat library root: %w", err)
	}
	if !info.IsDir() {
		return WalkResult{}, fmt.Errorf("library root is not a directory")
	}

	var result WalkResult
	if err := walkDir(ctx, opts.Root, ".", 0, opts.MaxDepth, &result); err != nil {
		return WalkResult{}, err
	}
	return result, nil
}

func walkDir(ctx context.Context, absRoot, rel string, depth, maxDepth int, result *WalkResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	absPath := absRoot
	if rel != "." {
		absPath = filepath.Join(absRoot, filepath.FromSlash(rel))
	}

	detection, err := Detect(absPath)
	if err != nil {
		if depth == 0 {
			return fmt.Errorf("detect library root: %w", err)
		}
		result.Errors = append(result.Errors, WalkError{
			Path:    rel,
			Kind:    "detect",
			Message: err.Error(),
		})
		return nil
	}
	if detection.IsGame {
		result.Games = append(result.Games, DiscoveredGame{RelPath: rel, Detection: detection})
		return nil
	}

	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		if depth == 0 {
			return fmt.Errorf("read library root: %w", err)
		}
		result.Errors = append(result.Errors, WalkError{
			Path:    rel,
			Kind:    "read_dir",
			Message: err.Error(),
		})
		return nil
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if shouldSkipDirectory(name) {
			continue
		}
		childRel := name
		if rel != "." {
			childRel = rel + "/" + name
		}
		if err := walkDir(ctx, absRoot, childRel, depth+1, maxDepth, result); err != nil {
			return err
		}
	}
	return nil
}

func shouldSkipDirectory(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return skippedDirectoryNames[strings.ToLower(name)]
}
