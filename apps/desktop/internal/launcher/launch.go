package launcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lunararch/oryn/apps/desktop/internal/paths"
	"github.com/lunararch/oryn/apps/desktop/internal/process"
	"github.com/lunararch/oryn/apps/desktop/internal/serverapi"
)

// Service launches games and reports their sessions.
type Service struct {
	API    *serverapi.Client
	Logger *slog.Logger
}

// RunResult describes a completed launch.
type RunResult struct {
	SessionID string
	ExitCode  int
	StartedAt time.Time
	EndedAt   time.Time
}

// LaunchAndTrack validates the candidate, starts a server session, launches
// the process, renews the lease, and reports its exit.
func (s *Service) LaunchAndTrack(ctx context.Context, gameID, libraryRoot, gameRelative string, candidate Candidate, args []string) (RunResult, error) {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	gameDir, err := paths.ResolveExisting(libraryRoot, gameRelative)
	if err != nil {
		return RunResult{}, err
	}
	info, err := os.Stat(gameDir)
	if err != nil || !info.IsDir() {
		return RunResult{}, fmt.Errorf("game path is not a directory")
	}
	if candidate.RelativePath == "" {
		return RunResult{}, fmt.Errorf("launch candidate is required")
	}
	if isInstaller(filepath.Base(candidate.RelativePath)) {
		return RunResult{}, fmt.Errorf("refusing to launch an installer or helper executable")
	}
	executable, err := paths.ResolveExisting(gameDir, candidate.RelativePath)
	if err != nil {
		return RunResult{}, err
	}
	if err := validateExecutable(executable); err != nil {
		return RunResult{}, err
	}
	if s.API == nil {
		return RunResult{}, fmt.Errorf("server API is not configured")
	}

	session, err := s.API.StartSession(ctx, gameID)
	if err != nil {
		return RunResult{}, fmt.Errorf("start play session: %w", err)
	}
	proc, err := process.Start(process.Spec{
		Path: executable,
		Args: args,
		Dir:  gameDir,
		Env:  os.Environ(),
	})
	if err != nil {
		finishErr := s.finish(session.ID, "failed", nil)
		if finishErr != nil {
			logger.Warn("could not report failed launch", "session_id", session.ID, "error", finishErr)
		}
		return RunResult{}, fmt.Errorf("launch game: %w", err)
	}

	interval := time.Duration(session.HeartbeatIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case result := <-proc.Done():
			status := "completed"
			exit := result.ExitCode
			if result.Err != nil || result.ExitCode != 0 {
				status = "failed"
			}
			if err := s.finish(session.ID, status, &exit); err != nil {
				return RunResult{SessionID: session.ID, ExitCode: exit, StartedAt: result.StartedAt, EndedAt: result.EndedAt}, err
			}
			return RunResult{SessionID: session.ID, ExitCode: exit, StartedAt: result.StartedAt, EndedAt: result.EndedAt}, nil
		case <-ticker.C:
			heartbeatCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.API.Heartbeat(heartbeatCtx, session.ID); err != nil {
				logger.Warn("session heartbeat failed", "session_id", session.ID, "error", err)
			}
			cancel()
		case <-ctx.Done():
			if err := proc.Kill(); err != nil {
				logger.Warn("could not stop game process", "pid", proc.PID(), "error", err)
			}
			finishErr := s.finish(session.ID, "failed", nil)
			if finishErr != nil {
				logger.Warn("could not report cancelled session", "session_id", session.ID, "error", finishErr)
			}
			return RunResult{}, ctx.Err()
		}
	}
}

func (s *Service) finish(sessionID, status string, exitCode *int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.API.Finish(ctx, sessionID, status, exitCode)
	return err
}

func validateExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("launch target is not a regular file")
	}
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(filepath.Ext(path), ".exe") {
			return fmt.Errorf("launch target is not a Windows executable")
		}
		return nil
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("launch target is not executable")
	}
	return nil
}
