package sessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

var (
	// ErrNotFound is returned when a session does not exist.
	ErrNotFound = errors.New("session not found")
	// ErrValidation is returned for invalid manual session input.
	ErrValidation = errors.New("validation failed")
)

const (
	maxManualDuration = 7 * 24 * time.Hour
	futureTolerance   = 5 * time.Minute
	maxNotesLength    = 2000
)

// Session is a play session with normalized timestamps.
type Session struct {
	ID              string
	GameID          string
	StartedAt       int64
	EndedAt         int64
	DurationSeconds int
	Status          string
	Source          string
	ExitCode        *int
	Notes           string
}

// Page is a paged session list.
type Page struct {
	Items    []Session
	Page     int
	PageSize int
	Total    int
}

// Stats contains playtime totals for a library.
type Stats struct {
	PlayedGames     int
	PlaytimeSeconds int
}

// Service manages manual play sessions.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// NewService builds a session service.
func NewService(s *store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: s, now: now}
}

// CreateManual records a completed manual session.
func (s *Service) CreateManual(ctx context.Context, gameID string, started, ended time.Time, notes string) (Session, error) {
	q := db.New(s.store.DB)
	game, err := q.GetGame(ctx, gameID)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("get game: %w", err)
	}
	if err := validateTimes(started, ended, notes, s.now()); err != nil {
		return Session{}, err
	}

	id := uuid.Must(uuid.NewV7()).String()
	now := s.now().Unix()
	if err := q.InsertPlaySession(ctx, db.InsertPlaySessionParams{
		ID:              id,
		GameID:          game.ID,
		StartedAt:       started.Unix(),
		EndedAt:         sql.NullInt64{Int64: ended.Unix(), Valid: true},
		DurationSeconds: int64(ended.Sub(started) / time.Second),
		Status:          "completed",
		Source:          "manual",
		Notes:           strings.TrimSpace(notes),
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return s.Get(ctx, id)
}

// Get returns one session.
func (s *Service) Get(ctx context.Context, id string) (Session, error) {
	row, err := db.New(s.store.DB).GetPlaySession(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return sessionFromRow(row), nil
}

// ListForGame returns sessions newest first.
func (s *Service) ListForGame(ctx context.Context, gameID string, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	q := db.New(s.store.DB)
	total, err := q.CountPlaySessionsForGame(ctx, gameID)
	if err != nil {
		return Page{}, fmt.Errorf("count sessions: %w", err)
	}
	rows, err := q.ListPlaySessionsForGame(ctx, db.ListPlaySessionsForGameParams{
		GameID: gameID,
		Limit:  int64(pageSize),
		Offset: int64((page - 1) * pageSize),
	})
	if err != nil {
		return Page{}, fmt.Errorf("list sessions: %w", err)
	}
	items := make([]Session, 0, len(rows))
	for _, row := range rows {
		items = append(items, sessionFromRow(row))
	}
	return Page{Items: items, Page: page, PageSize: pageSize, Total: int(total)}, nil
}

// UpdateManual corrects a manual session.
func (s *Service) UpdateManual(ctx context.Context, id string, started, ended time.Time, notes string) (Session, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return Session{}, err
	}
	if current.Source != "manual" {
		return Session{}, fmt.Errorf("%w: only manual sessions can be edited", ErrValidation)
	}
	if err := validateTimes(started, ended, notes, s.now()); err != nil {
		return Session{}, err
	}
	now := s.now().Unix()
	if err := db.New(s.store.DB).UpdatePlaySession(ctx, db.UpdatePlaySessionParams{
		StartedAt:       started.Unix(),
		EndedAt:         sql.NullInt64{Int64: ended.Unix(), Valid: true},
		DurationSeconds: int64(ended.Sub(started) / time.Second),
		Status:          "completed",
		Notes:           strings.TrimSpace(notes),
		UpdatedAt:       now,
		ID:              id,
	}); err != nil {
		return Session{}, fmt.Errorf("update session: %w", err)
	}
	return s.Get(ctx, id)
}

// DeleteManual removes a manual session.
func (s *Service) DeleteManual(ctx context.Context, id string) error {
	current, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Source != "manual" {
		return fmt.Errorf("%w: only manual sessions can be deleted", ErrValidation)
	}
	if err := db.New(s.store.DB).DeletePlaySession(ctx, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Stats returns playtime totals for a library.
func (s *Service) Stats(ctx context.Context, libraryID string) (Stats, error) {
	row, err := db.New(s.store.DB).GetPlayStats(ctx, libraryID)
	if err != nil {
		return Stats{}, fmt.Errorf("get play stats: %w", err)
	}
	return Stats{PlayedGames: int(row.PlayedGames), PlaytimeSeconds: int(row.TotalSeconds)}, nil
}

func validateTimes(started, ended time.Time, notes string, now time.Time) error {
	if started.IsZero() || ended.IsZero() {
		return fmt.Errorf("%w: start and end are required", ErrValidation)
	}
	if !ended.After(started) {
		return fmt.Errorf("%w: end must be after start", ErrValidation)
	}
	if started.After(now.Add(futureTolerance)) {
		return fmt.Errorf("%w: session cannot start in the future", ErrValidation)
	}
	if ended.After(now.Add(futureTolerance)) {
		return fmt.Errorf("%w: session cannot end in the future", ErrValidation)
	}
	if ended.Sub(started) > maxManualDuration {
		return fmt.Errorf("%w: session is longer than 7 days", ErrValidation)
	}
	if len(notes) > maxNotesLength {
		return fmt.Errorf("%w: notes are too long", ErrValidation)
	}
	return nil
}

func sessionFromRow(row db.PlaySession) Session {
	session := Session{
		ID:              row.ID,
		GameID:          row.GameID,
		StartedAt:       row.StartedAt,
		DurationSeconds: int(row.DurationSeconds),
		Status:          row.Status,
		Source:          row.Source,
		Notes:           row.Notes,
	}
	if row.EndedAt.Valid {
		session.EndedAt = row.EndedAt.Int64
	}
	if row.ExitCode.Valid {
		value := int(row.ExitCode.Int64)
		session.ExitCode = &value
	}
	return session
}
