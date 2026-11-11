package collections

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
	// ErrNotFound is returned when a collection or game does not exist.
	ErrNotFound = errors.New("collection not found")
	// ErrValidation is returned for invalid collection input.
	ErrValidation = errors.New("validation failed")
	// ErrConflict is returned when a collection name already exists.
	ErrConflict = errors.New("collection already exists")
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 2000
)

// Collection is a named group of games.
type Collection struct {
	ID          string
	Name        string
	Description string
	GameCount   int
	CreatedAt   int64
	UpdatedAt   int64
}

// Membership describes a collection and whether a game belongs to it.
type Membership struct {
	Collection
	Contains bool
}

// Service manages collections.
type Service struct {
	store *store.Store
	now   func() time.Time
}

// NewService builds a collection service.
func NewService(s *store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: s, now: now}
}

// Create makes a new collection.
func (s *Service) Create(ctx context.Context, name, description string) (Collection, error) {
	name, description, err := validate(name, description)
	if err != nil {
		return Collection{}, err
	}
	q := db.New(s.store.DB)
	all, err := q.ListCollections(ctx)
	if err != nil {
		return Collection{}, fmt.Errorf("list collections: %w", err)
	}
	for _, existing := range all {
		if strings.EqualFold(existing.Name, name) {
			return Collection{}, ErrConflict
		}
	}
	id := uuid.Must(uuid.NewV7()).String()
	now := s.now().Unix()
	if err := q.CreateCollection(ctx, db.CreateCollectionParams{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Collection{}, ErrConflict
		}
		return Collection{}, fmt.Errorf("create collection: %w", err)
	}
	return s.Get(ctx, id)
}

// Get returns one collection.
func (s *Service) Get(ctx context.Context, id string) (Collection, error) {
	row, err := db.New(s.store.DB).GetCollection(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Collection{}, ErrNotFound
	}
	if err != nil {
		return Collection{}, fmt.Errorf("get collection: %w", err)
	}
	return toCollection(row.ID, row.Name, row.Description, row.GameCount, row.CreatedAt, row.UpdatedAt), nil
}

// List returns all collections alphabetically.
func (s *Service) List(ctx context.Context) ([]Collection, error) {
	rows, err := db.New(s.store.DB).ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	items := make([]Collection, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCollection(row.ID, row.Name, row.Description, row.GameCount, row.CreatedAt, row.UpdatedAt))
	}
	return items, nil
}

// ListForGame returns all collections with a membership flag.
func (s *Service) ListForGame(ctx context.Context, gameID string) ([]Membership, error) {
	rows, err := db.New(s.store.DB).ListCollectionsForGame(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("list collections for game: %w", err)
	}
	items := make([]Membership, 0, len(rows))
	for _, row := range rows {
		items = append(items, Membership{
			Collection: toCollection(row.ID, row.Name, row.Description, row.GameCount, row.CreatedAt, row.UpdatedAt),
			Contains:   row.Contains,
		})
	}
	return items, nil
}

// Update changes a collection's name and description.
func (s *Service) Update(ctx context.Context, id, name, description string) (Collection, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return Collection{}, err
	}
	name, description, err := validate(name, description)
	if err != nil {
		return Collection{}, err
	}
	q := db.New(s.store.DB)
	all, err := q.ListCollections(ctx)
	if err != nil {
		return Collection{}, fmt.Errorf("list collections: %w", err)
	}
	for _, existing := range all {
		if existing.ID != id && strings.EqualFold(existing.Name, name) {
			return Collection{}, ErrConflict
		}
	}
	if err := q.UpdateCollection(ctx, db.UpdateCollectionParams{
		Name:        name,
		Description: description,
		UpdatedAt:   s.now().Unix(),
		ID:          id,
	}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Collection{}, ErrConflict
		}
		return Collection{}, fmt.Errorf("update collection: %w", err)
	}
	return s.Get(ctx, id)
}

// Delete removes a collection and its membership rows.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := db.New(s.store.DB).DeleteCollection(ctx, id); err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}

// AddGame adds a game to a collection idempotently.
func (s *Service) AddGame(ctx context.Context, collectionID, gameID string) error {
	q := db.New(s.store.DB)
	if _, err := q.GetCollection(ctx, collectionID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("get collection: %w", err)
	}
	if _, err := q.GetGame(ctx, gameID); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("get game: %w", err)
	}
	if err := q.AddGameToCollection(ctx, db.AddGameToCollectionParams{
		CollectionID: collectionID,
		GameID:       gameID,
		AddedAt:      s.now().Unix(),
	}); err != nil {
		return fmt.Errorf("add game to collection: %w", err)
	}
	return nil
}

// RemoveGame removes a game from a collection.
func (s *Service) RemoveGame(ctx context.Context, collectionID, gameID string) error {
	if err := db.New(s.store.DB).RemoveGameFromCollection(ctx, db.RemoveGameFromCollectionParams{
		CollectionID: collectionID,
		GameID:       gameID,
	}); err != nil {
		return fmt.Errorf("remove game from collection: %w", err)
	}
	return nil
}

func validate(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || len(name) > maxNameLength {
		return "", "", fmt.Errorf("%w: name must be between 1 and %d characters", ErrValidation, maxNameLength)
	}
	if len(description) > maxDescriptionLength {
		return "", "", fmt.Errorf("%w: description is too long", ErrValidation)
	}
	return name, description, nil
}

func toCollection(id, name, description string, count int64, created, updated int64) Collection {
	return Collection{
		ID:          id,
		Name:        name,
		Description: description,
		GameCount:   int(count),
		CreatedAt:   created,
		UpdatedAt:   updated,
	}
}
