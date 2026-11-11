package scanner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

// ScanState is a snapshot of the manager's current or most recent scan.
type ScanState struct {
	ID         string
	Running    bool
	StartedAt  time.Time
	FinishedAt time.Time
	Result     ScanResult
	Error      string
}

// Manager runs at most one scan at a time and exposes its state to the API.
type Manager struct {
	service *Service
	logger  *slog.Logger
	now     func() time.Time

	mu         sync.Mutex
	state      ScanState
	cancel     context.CancelFunc
	onComplete func(context.Context, db.Library)
}

// NewManager builds a scan manager.
func NewManager(service *Service, logger *slog.Logger, now func() time.Time) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Manager{service: service, logger: logger, now: now}
}

// Start launches a scan in the background.
func (m *Manager) Start(lib db.Library, opts ScanOptions) (string, error) {
	m.mu.Lock()
	if m.state.Running {
		m.mu.Unlock()
		return "", ErrScanInProgress
	}
	id := newID()
	opts.ScanRunID = id
	ctx, cancel := context.WithCancel(context.Background())
	m.state = ScanState{ID: id, Running: true, StartedAt: m.now()}
	m.cancel = cancel
	m.mu.Unlock()

	go func() {
		result, err := m.service.Scan(ctx, lib, opts)
		if err == nil && (result.Status == ScanCompleted || result.Status == ScanPartial) && m.onComplete != nil {
			hookCtx, hookCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			m.onComplete(hookCtx, lib)
			hookCancel()
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.state.Running = false
		m.state.FinishedAt = m.now()
		m.state.Result = result
		if err != nil {
			m.state.Error = "scan failed"
			m.logger.Error("background scan failed", "scan_id", id, "error", err)
		}
		m.cancel = nil
	}()

	return id, nil
}

// SetOnComplete registers a post-scan hook, such as artwork synchronisation.
func (m *Manager) SetOnComplete(fn func(context.Context, db.Library)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onComplete = fn
}

// Current returns the current or most recent scan state.
func (m *Manager) Current() ScanState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Cancel requests cancellation of the running scan.
func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.state.Running || (id != "" && id != m.state.ID) || m.cancel == nil {
		return false
	}
	m.cancel()
	return true
}
