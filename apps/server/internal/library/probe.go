package library

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Status describes whether a configured library root is currently usable.
type Status string

const (
	StatusAvailable     Status = "available"
	StatusUnavailable   Status = "unavailable"
	StatusNotConfigured Status = "not_configured"
)

// Probe checks that the library root and optional sentinel are present. It is
// intentionally shallow: health checks must not walk a network share.
type Probe struct {
	Root     string
	Sentinel string
	Timeout  time.Duration
}

// Status returns the current library availability. A missing root, a root that
// is not a directory, or a missing sentinel is reported as unavailable.
func (p Probe) Status(ctx context.Context) Status {
	if p.Root == "" {
		return StatusNotConfigured
	}
	if p.Timeout <= 0 {
		p.Timeout = time.Second
	}

	probeCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	info, err := statContext(probeCtx, p.Root)
	if err != nil || !info.IsDir() {
		return StatusUnavailable
	}

	if p.Sentinel != "" {
		sentinel := p.Sentinel
		if !filepath.IsAbs(sentinel) {
			sentinel = filepath.Join(p.Root, sentinel)
		}
		if _, err := statContext(probeCtx, sentinel); err != nil {
			return StatusUnavailable
		}
	}

	return StatusAvailable
}

func statContext(ctx context.Context, path string) (os.FileInfo, error) {
	type result struct {
		info os.FileInfo
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		info, err := os.Stat(path)
		ch <- result{info: info, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		return result.info, result.err
	}
}
