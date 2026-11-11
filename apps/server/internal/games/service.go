package games

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

// ErrNotFound is returned when a game does not exist.
var ErrNotFound = errors.New("game not found")

// ErrValidation is returned for invalid user input.
var ErrValidation = errors.New("validation failed")

// Field represents an optional PATCH field. Set distinguishes an omitted field
// from an explicit null.
type Field[T any] struct {
	Set   bool
	Value *T
}

// MetadataPatch contains user-editable fields. Only fields with Set=true are
// changed; a nil Value clears the override.
type MetadataPatch struct {
	Title            Field[string]
	Description      Field[string]
	Developer        Field[string]
	Publisher        Field[string]
	ReleaseDate      Field[string]
	Rating           Field[int]
	Notes            Field[string]
	CompletionStatus Field[string]
}

// ListParams controls a paged library query.
type ListParams struct {
	LibraryID    string
	Query        string
	Page         int
	PageSize     int
	Sort         string
	Order        string
	Favorite     *bool
	Availability string
	Completion   string
}

// Game is the resolved view of a game, with overrides applied.
type Game struct {
	ID               string
	LibraryID        string
	RelativePath     string
	FolderName       string
	Title            string
	Description      string
	Developer        string
	Publisher        string
	ReleaseDate      string
	Version          string
	Availability     string
	CompletionStatus string
	Rating           *int
	Notes            string
	Favorite         bool
	AddedAt          int64
	LastSeenAt       int64
	FirstSeenAt      int64
}

// Page is a paged game list.
type Page struct {
	Items      []Game
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}

// Stats contains library totals.
type Stats struct {
	Total     int
	Available int
	Missing   int
	Favorites int
}

// Service implements library queries and user metadata changes.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// NewService builds a games service.
func NewService(s *store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: s, now: now}
}

// List returns a filtered page of games.
func (s *Service) List(ctx context.Context, params ListParams) (Page, error) {
	if params.LibraryID == "" {
		return Page{}, fmt.Errorf("%w: library is required", ErrValidation)
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 60
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	if params.Sort == "" {
		params.Sort = "title"
	}
	if params.Order == "" {
		if params.Sort == "title" {
			params.Order = "asc"
		} else {
			params.Order = "desc"
		}
	}
	if params.Order != "asc" && params.Order != "desc" {
		return Page{}, fmt.Errorf("%w: invalid sort order", ErrValidation)
	}
	if params.Availability != "" && params.Availability != "available" && params.Availability != "missing" {
		return Page{}, fmt.Errorf("%w: invalid availability", ErrValidation)
	}
	if params.Completion != "" && !validCompletion(params.Completion) {
		return Page{}, fmt.Errorf("%w: invalid completion status", ErrValidation)
	}

	orderBy, ok := map[string]string{
		"title":    "COALESCE(o.title, g.detected_title) COLLATE NOCASE",
		"added":    "g.first_seen_at",
		"updated":  "g.updated_at",
		"lastSeen": "g.last_seen_at",
		"path":     "g.relative_path",
	}[params.Sort]
	if !ok {
		return Page{}, fmt.Errorf("%w: invalid sort field", ErrValidation)
	}

	where, args := buildWhere(params)
	countQuery := `SELECT COUNT(*) FROM games g LEFT JOIN game_overrides o ON o.game_id = g.id WHERE ` + where

	var total int
	if err := s.store.DB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("count games: %w", err)
	}

	offset := (params.Page - 1) * params.PageSize
	query := `
SELECT
    g.id, g.library_id, g.relative_path, g.folder_name,
    COALESCE(o.title, g.detected_title) AS title,
    COALESCE(o.description, g.detected_description) AS description,
    COALESCE(o.developer, g.detected_developer) AS developer,
    COALESCE(o.publisher, g.detected_publisher) AS publisher,
    COALESCE(o.release_date, g.detected_release_date) AS release_date,
    g.detected_version,
    g.availability_status,
    COALESCE(o.completion_status, 'unplayed') AS completion_status,
    o.rating,
    COALESCE(o.notes, '') AS notes,
    EXISTS(SELECT 1 FROM favorites f WHERE f.game_id = g.id) AS is_favorite,
    g.first_seen_at, g.last_seen_at
FROM games g
LEFT JOIN game_overrides o ON o.game_id = g.id
WHERE ` + where + `
ORDER BY ` + orderBy + ` ` + strings.ToUpper(params.Order) + `
LIMIT ? OFFSET ?`
	args = append(args, params.PageSize, offset)

	rows, err := s.store.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return Page{}, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()

	items := make([]Game, 0, params.PageSize)
	for rows.Next() {
		game, err := scanGame(rows)
		if err != nil {
			return Page{}, err
		}
		items = append(items, game)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("list games: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + params.PageSize - 1) / params.PageSize
	}
	return Page{
		Items:      items,
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func buildWhere(params ListParams) (string, []any) {
	clauses := []string{"g.library_id = ?"}
	args := []any{params.LibraryID}

	if query := strings.TrimSpace(params.Query); query != "" {
		pattern := "%" + escapeLike(query) + "%"
		clauses = append(clauses, `(
			COALESCE(o.title, g.detected_title) LIKE ? ESCAPE '\' OR
			COALESCE(o.description, g.detected_description) LIKE ? ESCAPE '\' OR
			g.folder_name LIKE ? ESCAPE '\' OR
			g.detected_developer LIKE ? ESCAPE '\' OR
			g.detected_publisher LIKE ? ESCAPE '\'
		)`)
		for i := 0; i < 5; i++ {
			args = append(args, pattern)
		}
	}
	if params.Favorite != nil {
		if *params.Favorite {
			clauses = append(clauses, `EXISTS(SELECT 1 FROM favorites f WHERE f.game_id = g.id)`)
		} else {
			clauses = append(clauses, `NOT EXISTS(SELECT 1 FROM favorites f WHERE f.game_id = g.id)`)
		}
	}
	if params.Availability != "" {
		clauses = append(clauses, "g.availability_status = ?")
		args = append(args, params.Availability)
	}
	if params.Completion != "" {
		clauses = append(clauses, "COALESCE(o.completion_status, 'unplayed') = ?")
		args = append(args, params.Completion)
	}
	return strings.Join(clauses, " AND "), args
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

// Get returns one game with overrides applied.
func (s *Service) Get(ctx context.Context, id string) (Game, error) {
	row, err := db.New(s.store.DB).GetGame(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	if err != nil {
		return Game{}, fmt.Errorf("get game: %w", err)
	}
	return gameFromRow(row), nil
}

// UpdateMetadata applies a validated metadata patch and returns the resolved game.
func (s *Service) UpdateMetadata(ctx context.Context, id string, patch MetadataPatch) (Game, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Game{}, err
	}
	if err := validatePatch(patch); err != nil {
		return Game{}, err
	}

	q := db.New(s.store.DB)
	existing, err := q.GetGameOverride(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Game{}, fmt.Errorf("get metadata override: %w", err)
	}

	next := db.UpsertGameOverrideParams{GameID: id, UpdatedAt: s.now().Unix()}
	if err == nil {
		next.Title = existing.Title
		next.Description = existing.Description
		next.Developer = existing.Developer
		next.Publisher = existing.Publisher
		next.ReleaseDate = existing.ReleaseDate
		next.Rating = existing.Rating
		next.Notes = existing.Notes
		next.CompletionStatus = existing.CompletionStatus
	}

	next.Title = applyString(patch.Title, next.Title)
	next.Description = applyString(patch.Description, next.Description)
	next.Developer = applyString(patch.Developer, next.Developer)
	next.Publisher = applyString(patch.Publisher, next.Publisher)
	next.ReleaseDate = applyString(patch.ReleaseDate, next.ReleaseDate)
	next.Notes = applyString(patch.Notes, next.Notes)
	if patch.Rating.Set {
		if patch.Rating.Value == nil {
			next.Rating = sql.NullInt64{}
		} else {
			next.Rating = sql.NullInt64{Int64: int64(*patch.Rating.Value), Valid: true}
		}
	}
	if patch.CompletionStatus.Set {
		if patch.CompletionStatus.Value == nil || strings.TrimSpace(*patch.CompletionStatus.Value) == "" {
			next.CompletionStatus = sql.NullString{}
		} else {
			next.CompletionStatus = sql.NullString{String: strings.TrimSpace(*patch.CompletionStatus.Value), Valid: true}
		}
	}

	if allEmpty(next) {
		if err := q.DeleteGameOverride(ctx, id); err != nil {
			return Game{}, fmt.Errorf("delete metadata override: %w", err)
		}
	} else if err := q.UpsertGameOverride(ctx, next); err != nil {
		return Game{}, fmt.Errorf("save metadata override: %w", err)
	}

	return s.Get(ctx, id)
}

// SetFavorite adds or removes a favorite idempotently.
func (s *Service) SetFavorite(ctx context.Context, id string, favorite bool) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	q := db.New(s.store.DB)
	if favorite {
		if err := q.SetFavorite(ctx, db.SetFavoriteParams{GameID: id, CreatedAt: s.now().Unix()}); err != nil {
			return fmt.Errorf("set favorite: %w", err)
		}
		return nil
	}
	if err := q.DeleteFavorite(ctx, id); err != nil {
		return fmt.Errorf("remove favorite: %w", err)
	}
	return nil
}

// GetStats returns library totals.
func (s *Service) GetStats(ctx context.Context, libraryID string) (Stats, error) {
	row, err := db.New(s.store.DB).GetStats(ctx, libraryID)
	if err != nil {
		return Stats{}, fmt.Errorf("get stats: %w", err)
	}
	return Stats{
		Total:     int(row.TotalGames),
		Available: int(row.AvailableGames),
		Missing:   int(row.MissingGames),
		Favorites: int(row.FavoriteGames),
	}, nil
}

func scanGame(rows *sql.Rows) (Game, error) {
	var game Game
	var rating sql.NullInt64
	var favorite bool
	if err := rows.Scan(
		&game.ID, &game.LibraryID, &game.RelativePath, &game.FolderName,
		&game.Title, &game.Description, &game.Developer, &game.Publisher,
		&game.ReleaseDate, &game.Version, &game.Availability, &game.CompletionStatus,
		&rating, &game.Notes, &favorite, &game.FirstSeenAt, &game.LastSeenAt,
	); err != nil {
		return Game{}, fmt.Errorf("scan game: %w", err)
	}
	if rating.Valid {
		value := int(rating.Int64)
		game.Rating = &value
	}
	game.Favorite = favorite
	game.AddedAt = game.FirstSeenAt
	return game, nil
}

func gameFromRow(row db.GetGameRow) Game {
	game := Game{
		ID:               row.ID,
		LibraryID:        row.LibraryID,
		RelativePath:     row.RelativePath,
		FolderName:       row.FolderName,
		Title:            row.DetectedTitle,
		Description:      row.DetectedDescription,
		Developer:        row.DetectedDeveloper,
		Publisher:        row.DetectedPublisher,
		ReleaseDate:      row.DetectedReleaseDate,
		Version:          row.DetectedVersion,
		Availability:     row.AvailabilityStatus,
		CompletionStatus: "unplayed",
		Favorite:         row.IsFavorite,
		AddedAt:          row.FirstSeenAt,
		LastSeenAt:       row.LastSeenAt,
		FirstSeenAt:      row.FirstSeenAt,
	}
	if row.OverrideTitle.Valid {
		game.Title = row.OverrideTitle.String
	}
	if row.OverrideDescription.Valid {
		game.Description = row.OverrideDescription.String
	}
	if row.OverrideDeveloper.Valid {
		game.Developer = row.OverrideDeveloper.String
	}
	if row.OverridePublisher.Valid {
		game.Publisher = row.OverridePublisher.String
	}
	if row.OverrideReleaseDate.Valid {
		game.ReleaseDate = row.OverrideReleaseDate.String
	}
	if row.OverrideRating.Valid {
		value := int(row.OverrideRating.Int64)
		game.Rating = &value
	}
	if row.OverrideNotes.Valid {
		game.Notes = row.OverrideNotes.String
	}
	if row.OverrideCompletionStatus.Valid {
		game.CompletionStatus = row.OverrideCompletionStatus.String
	}
	return game
}

func applyString(field Field[string], current sql.NullString) sql.NullString {
	if !field.Set {
		return current
	}
	if field.Value == nil || strings.TrimSpace(*field.Value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(*field.Value), Valid: true}
}

func allEmpty(p db.UpsertGameOverrideParams) bool {
	return !p.Title.Valid && !p.Description.Valid && !p.Developer.Valid &&
		!p.Publisher.Valid && !p.ReleaseDate.Valid && !p.Rating.Valid &&
		!p.Notes.Valid && !p.CompletionStatus.Valid
}

func validCompletion(value string) bool {
	switch value {
	case "unplayed", "playing", "completed", "dropped", "on_hold":
		return true
	default:
		return false
	}
}

func validatePatch(patch MetadataPatch) error {
	lengths := []struct {
		name  string
		field Field[string]
		max   int
	}{
		{"title", patch.Title, 300},
		{"description", patch.Description, 20000},
		{"developer", patch.Developer, 200},
		{"publisher", patch.Publisher, 200},
		{"release date", patch.ReleaseDate, 50},
		{"notes", patch.Notes, 10000},
	}
	for _, item := range lengths {
		if item.field.Set && item.field.Value != nil && len(*item.field.Value) > item.max {
			return fmt.Errorf("%w: %s is too long", ErrValidation, item.name)
		}
	}
	if patch.Rating.Set && patch.Rating.Value != nil {
		if *patch.Rating.Value < 1 || *patch.Rating.Value > 10 {
			return fmt.Errorf("%w: rating must be between 1 and 10", ErrValidation)
		}
	}
	if patch.CompletionStatus.Set && patch.CompletionStatus.Value != nil {
		value := strings.TrimSpace(*patch.CompletionStatus.Value)
		if value != "" && !validCompletion(value) {
			return fmt.Errorf("%w: invalid completion status", ErrValidation)
		}
	}
	return nil
}
