package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

// Scan statuses stored in scan_runs.
const (
	ScanRunning   = "running"
	ScanCompleted = "completed"
	ScanPartial   = "partial"
	ScanFailed    = "failed"
	ScanCancelled = "cancelled"
)

// ErrScanInProgress is returned when a scan is already running.
var ErrScanInProgress = errors.New("a scan is already running")

// ScanOptions controls one scan run.
type ScanOptions struct {
	AllowMissing bool
	ScanRunID    string
}

// ScanResult summarises one scan run.
type ScanResult struct {
	ScanRunID         string
	Status            string
	Discovered        int
	Added             int
	Updated           int
	Unchanged         int
	Missing           int
	MissingCandidates int
	Errors            int
	Summary           string
}

// ServiceOptions configures the scanner service.
type ServiceOptions struct {
	Store    *store.Store
	MaxDepth int
	Sentinel string
	Now      func() time.Time
	Logger   *slog.Logger
}

// Service coordinates safe discovery and persistence.
type Service struct {
	store    *store.Store
	maxDepth int
	sentinel string
	now      func() time.Time
	logger   *slog.Logger
	mu       sync.Mutex
}

// NewService builds a scanner service.
func NewService(opts ServiceOptions) *Service {
	if opts.MaxDepth < 1 {
		opts.MaxDepth = 3
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Service{
		store:    opts.Store,
		maxDepth: opts.MaxDepth,
		sentinel: opts.Sentinel,
		now:      opts.Now,
		logger:   opts.Logger,
	}
}

// Scan discovers games under the library root and applies the result. Failures
// of the library mount are represented by a failed ScanResult, not by an
// error, because the scan history should record them.
func (s *Service) Scan(ctx context.Context, lib db.Library, opts ScanOptions) (ScanResult, error) {
	if s.store == nil {
		return ScanResult{}, fmt.Errorf("scanner store is not configured")
	}
	if !s.mu.TryLock() {
		return ScanResult{}, ErrScanInProgress
	}
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return ScanResult{Status: ScanCancelled, Summary: "scan cancelled"}, nil
	}

	q := db.New(s.store.DB)
	if _, err := q.GetCurrentScanRun(ctx); err == nil {
		return ScanResult{}, ErrScanInProgress
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ScanResult{}, fmt.Errorf("check current scan: %w", err)
	}

	startedAt := s.now().Unix()
	runID := opts.ScanRunID
	if runID == "" {
		runID = newID()
	}
	if err := q.CreateScanRun(ctx, db.CreateScanRunParams{
		ID:        runID,
		LibraryID: lib.ID,
		StartedAt: startedAt,
	}); err != nil {
		return ScanResult{}, fmt.Errorf("create scan run: %w", err)
	}

	result := ScanResult{ScanRunID: runID, Status: ScanRunning}
	s.logger.Info("scan started", "scan_id", runID, "library_id", lib.ID)

	if err := s.verifyLibrary(lib.RootPath); err != nil {
		result.Status = ScanFailed
		result.Summary = err.Error()
		s.finish(runID, lib, result, result.Summary)
		s.logger.Warn("scan failed before discovery", "scan_id", runID, "error", err)
		return result, nil
	}

	walk, err := Walk(ctx, WalkOptions{Root: lib.RootPath, MaxDepth: s.maxDepth})
	if err != nil {
		result.Status = ScanFailed
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result.Status = ScanCancelled
			result.Summary = "scan cancelled"
		} else {
			result.Summary = "library root is unavailable or unreadable"
			s.logger.Warn("scan walk failed", "scan_id", runID, "error", err)
		}
		s.finish(runID, lib, result, result.Summary)
		return result, nil
	}

	result.Discovered = len(walk.Games)
	result.Errors = len(walk.Errors)
	if ctx.Err() != nil {
		result.Status = ScanCancelled
		result.Summary = "scan cancelled"
		s.finish(runID, lib, result, result.Summary)
		return result, nil
	}

	knownAvailable, err := q.CountAvailableGames(ctx, lib.ID)
	if err != nil {
		return result, fmt.Errorf("count known games: %w", err)
	}
	if result.Discovered == 0 && knownAvailable > 0 {
		result.Status = ScanFailed
		result.Summary = "library root is readable but no games were discovered; existing records preserved"
		s.finish(runID, lib, result, result.Summary)
		return result, nil
	}

	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin scan transaction: %w", err)
	}
	qtx := db.New(tx)
	applyErr := s.apply(ctx, qtx, lib, walk, opts, &result)
	if applyErr != nil {
		_ = tx.Rollback()
		result.Status = ScanFailed
		result.Summary = "could not apply scan results"
		s.finish(runID, lib, result, result.Summary)
		return result, fmt.Errorf("apply scan results: %w", applyErr)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit scan results: %w", err)
	}

	switch {
	case len(walk.Errors) > 0:
		result.Status = ScanPartial
		result.Summary = fmt.Sprintf("scan completed with %d recoverable errors; missing-game reconciliation skipped", len(walk.Errors))
	case result.MissingCandidates > 0 && result.Missing == 0:
		result.Status = ScanPartial
		result.Summary = fmt.Sprintf("missing-game reconciliation skipped: %d games would be marked missing", result.MissingCandidates)
	default:
		result.Status = ScanCompleted
		result.Summary = "scan completed"
	}

	s.finish(runID, lib, result, result.Summary)
	s.logger.Info("scan finished",
		"scan_id", runID,
		"status", result.Status,
		"discovered", result.Discovered,
		"added", result.Added,
		"updated", result.Updated,
		"missing", result.Missing,
		"errors", result.Errors,
	)
	return result, nil
}

func (s *Service) apply(ctx context.Context, q *db.Queries, lib db.Library, walk WalkResult, opts ScanOptions, result *ScanResult) error {
	existing, err := q.ListGamesForLibrary(ctx, lib.ID)
	if err != nil {
		return fmt.Errorf("list existing games: %w", err)
	}

	existingByPath := make(map[string]db.ListGamesForLibraryRow, len(existing))
	knownAvailable := 0
	for _, game := range existing {
		existingByPath[game.RelativePath] = game
		if game.AvailabilityStatus == "available" {
			knownAvailable++
		}
	}

	discoveredPaths := make(map[string]bool, len(walk.Games))
	missing := make([]db.ListGamesForLibraryRow, 0)
	for _, game := range existing {
		if game.AvailabilityStatus == "available" {
			missing = append(missing, game)
		}
	}
	for _, game := range walk.Games {
		discoveredPaths[game.RelPath] = true
	}
	filtered := missing[:0]
	for _, game := range missing {
		if !discoveredPaths[game.RelativePath] {
			filtered = append(filtered, game)
		}
	}
	missing = filtered
	result.MissingCandidates = len(missing)

	missingByFingerprint := make(map[string][]db.ListGamesForLibraryRow)
	for _, game := range missing {
		if game.FilesystemFingerprint != "" {
			missingByFingerprint[game.FilesystemFingerprint] = append(missingByFingerprint[game.FilesystemFingerprint], game)
		}
	}

	cleanScan := len(walk.Errors) == 0
	massMissing := !opts.AllowMissing && len(missing) > max(10, knownAvailable/4)

	now := s.now().Unix()
	seen := make(map[string]bool, len(walk.Games))

	for _, game := range walk.Games {
		if err := ctx.Err(); err != nil {
			return err
		}
		detection := game.Detection
		fingerprint := Fingerprint(detection)
		folderName := filepath.Base(filepath.FromSlash(game.RelPath))
		if game.RelPath == "." {
			folderName = filepath.Base(lib.RootPath)
		}
		evidence, err := json.Marshal(detection.Evidence)
		if err != nil {
			return fmt.Errorf("encode detection evidence: %w", err)
		}

		if current, ok := existingByPath[game.RelPath]; ok {
			seen[current.ID] = true
			unchanged := current.DetectedTitle == detection.Title &&
				current.FilesystemFingerprint == fingerprint &&
				current.AvailabilityStatus == "available"
			if err := q.UpdateGameDetected(ctx, db.UpdateGameDetectedParams{
				FolderName:            folderName,
				DetectedTitle:         detection.Title,
				DetectedDescription:   detection.About,
				DetectedDeveloper:     "",
				DetectedPublisher:     "",
				DetectedReleaseDate:   "",
				DetectedVersion:       detection.Version,
				DetectionConfidence:   int64(detection.Confidence),
				DetectionEvidence:     string(evidence),
				FilesystemFingerprint: fingerprint,
				LastSeenAt:            now,
				UpdatedAt:             now,
				ID:                    current.ID,
			}); err != nil {
				return fmt.Errorf("update game %s: %w", current.ID, err)
			}
			if unchanged {
				result.Unchanged++
			} else {
				result.Updated++
			}
			continue
		}

		if cleanScan {
			if matches := missingByFingerprint[fingerprint]; fingerprint != "" && len(matches) == 1 {
				moved := matches[0]
				seen[moved.ID] = true
				if err := q.UpdateGamePath(ctx, db.UpdateGamePathParams{
					RelativePath: game.RelPath,
					FolderName:   folderName,
					LastSeenAt:   now,
					UpdatedAt:    now,
					ID:           moved.ID,
				}); err != nil {
					return fmt.Errorf("move game %s: %w", moved.ID, err)
				}
				if err := q.UpdateGameDetected(ctx, db.UpdateGameDetectedParams{
					FolderName:            folderName,
					DetectedTitle:         detection.Title,
					DetectedDescription:   detection.About,
					DetectedDeveloper:     "",
					DetectedPublisher:     "",
					DetectedReleaseDate:   "",
					DetectedVersion:       detection.Version,
					DetectionConfidence:   int64(detection.Confidence),
					DetectionEvidence:     string(evidence),
					FilesystemFingerprint: fingerprint,
					LastSeenAt:            now,
					UpdatedAt:             now,
					ID:                    moved.ID,
				}); err != nil {
					return fmt.Errorf("update moved game %s: %w", moved.ID, err)
				}
				delete(missingByFingerprint, fingerprint)
				result.Updated++
				continue
			}
		}

		id := newID()
		if err := q.InsertGame(ctx, db.InsertGameParams{
			ID:                    id,
			LibraryID:             lib.ID,
			RelativePath:          game.RelPath,
			FolderName:            folderName,
			DetectedTitle:         detection.Title,
			DetectedDescription:   detection.About,
			DetectedDeveloper:     "",
			DetectedPublisher:     "",
			DetectedReleaseDate:   "",
			DetectedVersion:       detection.Version,
			DetectionConfidence:   int64(detection.Confidence),
			DetectionEvidence:     string(evidence),
			AvailabilityStatus:    "available",
			FilesystemFingerprint: fingerprint,
			FirstSeenAt:           now,
			LastSeenAt:            now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}); err != nil {
			return fmt.Errorf("insert game %s: %w", game.RelPath, err)
		}
		seen[id] = true
		result.Added++
	}

	for i, walkErr := range walk.Errors {
		if i >= 1000 {
			break
		}
		if err := q.AddScanError(ctx, db.AddScanErrorParams{
			ID:        newID(),
			ScanRunID: result.ScanRunID,
			Path:      walkErr.Path,
			Kind:      walkErr.Kind,
			Message:   walkErr.Message,
			CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("store scan error: %w", err)
		}
	}

	if cleanScan && !massMissing {
		for _, game := range missing {
			if seen[game.ID] {
				continue
			}
			if err := q.SetGameAvailability(ctx, db.SetGameAvailabilityParams{
				AvailabilityStatus: "missing",
				LastSeenAt:         game.LastSeenAt,
				UpdatedAt:          now,
				ID:                 game.ID,
			}); err != nil {
				return fmt.Errorf("mark game %s missing: %w", game.ID, err)
			}
			result.Missing++
		}
	}

	return nil
}

func (s *Service) verifyLibrary(root string) error {
	if root == "" {
		return fmt.Errorf("library root is not configured")
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("library root is unavailable")
	}
	if !info.IsDir() {
		return fmt.Errorf("library root is not a directory")
	}
	f, err := os.Open(root)
	if err != nil {
		return fmt.Errorf("library root is unreadable")
	}
	defer f.Close()
	if _, err := f.Readdirnames(1); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("library root is unreadable")
	}
	if s.sentinel != "" {
		sentinel := s.sentinel
		if !filepath.IsAbs(sentinel) {
			sentinel = filepath.Join(root, sentinel)
		}
		if _, err := os.Stat(sentinel); err != nil {
			return fmt.Errorf("library sentinel is missing")
		}
	}
	return nil
}

func (s *Service) finish(runID string, lib db.Library, result ScanResult, summary string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := db.New(s.store.DB)
	now := s.now().Unix()
	if err := q.FinishScanRun(ctx, db.FinishScanRunParams{
		Status:          result.Status,
		FinishedAt:      sql.NullInt64{Int64: now, Valid: true},
		DiscoveredCount: int64(result.Discovered),
		AddedCount:      int64(result.Added),
		UpdatedCount:    int64(result.Updated),
		UnchangedCount:  int64(result.Unchanged),
		MissingCount:    int64(result.Missing),
		ErrorCount:      int64(result.Errors),
		Summary:         summary,
		ID:              runID,
	}); err != nil {
		s.logger.Error("could not finish scan run", "scan_id", runID, "error", err)
	}
	if result.Status == ScanCompleted {
		if err := q.TouchLibraryScan(ctx, db.TouchLibraryScanParams{
			LastScanAt: sql.NullInt64{Int64: now, Valid: true},
			UpdatedAt:  now,
			ID:         lib.ID,
		}); err != nil {
			s.logger.Error("could not update library scan timestamp", "library_id", lib.ID, "error", err)
		}
	}
}

func newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return id.String()
}
