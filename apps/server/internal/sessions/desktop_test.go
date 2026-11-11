package sessions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDesktopSessionLifecycle(t *testing.T) {
	f := newSessionsFixture(t)
	clientID := uuid.Must(uuid.NewV7()).String()

	started, err := f.service.StartDesktop(context.Background(), clientID, f.gameID)
	if err != nil {
		t.Fatalf("StartDesktop() error = %v", err)
	}
	if started.Status != "running" || started.Source != "desktop" {
		t.Fatalf("session = %+v", started)
	}

	if _, err := f.service.StartDesktop(context.Background(), clientID, f.gameID); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second StartDesktop() error = %v, want ErrAlreadyRunning", err)
	}

	if err := f.service.Heartbeat(context.Background(), clientID, started.ID); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}

	finished, err := f.service.FinishDesktop(context.Background(), clientID, started.ID, "completed", intPtr(0))
	if err != nil {
		t.Fatalf("FinishDesktop() error = %v", err)
	}
	if finished.Status != "completed" {
		t.Errorf("status = %q", finished.Status)
	}
	if finished.DurationSeconds < 0 {
		t.Errorf("duration = %d", finished.DurationSeconds)
	}
	if finished.ExitCode == nil || *finished.ExitCode != 0 {
		t.Errorf("exit code = %v", finished.ExitCode)
	}
}

func TestDesktopSessionOwnership(t *testing.T) {
	f := newSessionsFixture(t)
	clientID := uuid.Must(uuid.NewV7()).String()
	other := uuid.Must(uuid.NewV7()).String()
	started, err := f.service.StartDesktop(context.Background(), clientID, f.gameID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.Heartbeat(context.Background(), other, started.ID); err == nil {
		t.Fatal("Heartbeat() succeeded for another client")
	}
	if _, err := f.service.FinishDesktop(context.Background(), other, started.ID, "completed", nil); err == nil {
		t.Fatal("FinishDesktop() succeeded for another client")
	}
}

func TestRecoverStaleSessions(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	f := newSessionsFixture(t)
	f.service.now = func() time.Time { return now }
	clientID := uuid.Must(uuid.NewV7()).String()
	started, err := f.service.StartDesktop(context.Background(), clientID, f.gameID)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a heartbeat ten minutes ago, then jump the clock forward.
	oldHeartbeat := now.Add(-10 * time.Minute).Unix()
	if _, err := f.store.DB.Exec(`UPDATE play_sessions SET started_at = ?, last_heartbeat_at = ? WHERE id = ?`,
		now.Add(-15*time.Minute).Unix(), oldHeartbeat, started.ID); err != nil {
		t.Fatal(err)
	}
	f.service.now = func() time.Time { return now }

	recovered, err := f.service.RecoverStale(context.Background(), 2*time.Minute)
	if err != nil {
		t.Fatalf("RecoverStale() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	session, err := f.service.Get(context.Background(), started.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != "interrupted" {
		t.Errorf("status = %q, want interrupted", session.Status)
	}
	if session.EndedAt != oldHeartbeat {
		t.Errorf("endedAt = %d, want last heartbeat %d", session.EndedAt, oldHeartbeat)
	}
	if session.DurationSeconds != 300 {
		t.Errorf("duration = %d, want 300", session.DurationSeconds)
	}
}

func TestStartDesktopRequiresGame(t *testing.T) {
	f := newSessionsFixture(t)
	_, err := f.service.StartDesktop(context.Background(), uuid.Must(uuid.NewV7()).String(), uuid.Must(uuid.NewV7()).String())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func intPtr(value int) *int {
	return &value
}
