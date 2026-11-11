package artwork

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/lunararch/oryn/apps/server/internal/scanner"
	"github.com/lunararch/oryn/apps/server/internal/store"
	"github.com/lunararch/oryn/apps/server/internal/store/db"
)

const (
	// CardWidth is the maximum width of a grid/list thumbnail.
	CardWidth = 480
	// DetailWidth is the maximum width of a detail-page image.
	DetailWidth = 1400

	defaultMaxFileBytes = 10 << 20
	defaultMaxDimension = 8192
	defaultMaxPixels    = 40_000_000
)

// ErrNotFound is returned when artwork or a variant does not exist.
var ErrNotFound = errors.New("artwork not found")

// Options configures the artwork service.
type Options struct {
	Store        *store.Store
	CacheDir     string
	Now          func() time.Time
	Logger       *slog.Logger
	MaxFileBytes int64
	MaxDimension int
	MaxPixels    int
}

// Service validates artwork, generates cached variants, and records selections.
type Service struct {
	store        *store.Store
	cacheDir     string
	now          func() time.Time
	logger       *slog.Logger
	maxFileBytes int64
	maxDimension int
	maxPixels    int
}

// NewService builds an artwork service.
func NewService(opts Options) *Service {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = defaultMaxFileBytes
	}
	if opts.MaxDimension <= 0 {
		opts.MaxDimension = defaultMaxDimension
	}
	if opts.MaxPixels <= 0 {
		opts.MaxPixels = defaultMaxPixels
	}
	return &Service{
		store:        opts.Store,
		cacheDir:     opts.CacheDir,
		now:          opts.Now,
		logger:       opts.Logger,
		maxFileBytes: opts.MaxFileBytes,
		maxDimension: opts.MaxDimension,
		maxPixels:    opts.MaxPixels,
	}
}

// ProcessFile validates sourcePath and writes card/detail variants to the cache.
func (s *Service) ProcessFile(ctx context.Context, gameID, kind, sourcePath, sourceRel, origin string) (db.Artwork, error) {
	if kind != "cover" && kind != "banner" {
		return db.Artwork{}, fmt.Errorf("invalid artwork kind")
	}
	if origin != "detected" && origin != "upload" {
		return db.Artwork{}, fmt.Errorf("invalid artwork origin")
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return db.Artwork{}, fmt.Errorf("stat artwork: %w", err)
	}
	if !info.Mode().IsRegular() {
		return db.Artwork{}, fmt.Errorf("artwork is not a regular file")
	}
	if info.Size() > s.maxFileBytes {
		return db.Artwork{}, fmt.Errorf("artwork exceeds the size limit")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return db.Artwork{}, fmt.Errorf("read artwork: %w", err)
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return db.Artwork{}, fmt.Errorf("decode artwork header: %w", err)
	}
	switch format {
	case "png", "jpeg", "webp":
	default:
		return db.Artwork{}, fmt.Errorf("unsupported artwork format")
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return db.Artwork{}, fmt.Errorf("invalid artwork dimensions")
	}
	if cfg.Width > s.maxDimension || cfg.Height > s.maxDimension {
		return db.Artwork{}, fmt.Errorf("artwork dimensions exceed the limit")
	}
	if int64(cfg.Width)*int64(cfg.Height) > int64(s.maxPixels) {
		return db.Artwork{}, fmt.Errorf("artwork pixel count exceeds the limit")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return db.Artwork{}, fmt.Errorf("decode artwork: %w", err)
	}

	sum := sha256.Sum256(append([]byte(gameID+"|"+kind+"|"), data...))
	cacheKey := hex.EncodeToString(sum[:16])

	q := db.New(s.store.DB)
	existing, lookupErr := q.GetArtworkForGame(ctx, db.GetArtworkForGameParams{GameID: gameID, Kind: kind})
	hadExisting := lookupErr == nil
	if hadExisting && existing.CacheKey == cacheKey {
		if s.variantExists(existing.CacheKey, "card") && s.variantExists(existing.CacheKey, "detail") {
			return existing, nil
		}
	}

	if err := os.MkdirAll(s.cacheDir, 0o750); err != nil {
		return db.Artwork{}, fmt.Errorf("create artwork cache: %w", err)
	}
	if err := s.writeVariant(img, CardWidth, s.variantPath(cacheKey, "card")); err != nil {
		return db.Artwork{}, err
	}
	if err := s.writeVariant(img, DetailWidth, s.variantPath(cacheKey, "detail")); err != nil {
		return db.Artwork{}, err
	}

	id := uuid.Must(uuid.NewV7()).String()
	now := s.now().Unix()
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return db.Artwork{}, fmt.Errorf("begin artwork transaction: %w", err)
	}
	qtx := db.New(tx)
	if hadExisting {
		// The row is replaced so a changed image gets a new immutable URL.
		if err := qtx.DeleteArtwork(ctx, db.DeleteArtworkParams{GameID: gameID, Kind: kind}); err != nil {
			_ = tx.Rollback()
			return db.Artwork{}, fmt.Errorf("replace artwork: %w", err)
		}
	}
	source := sql.NullString{}
	if sourceRel != "" {
		source = sql.NullString{String: filepath.ToSlash(sourceRel), Valid: true}
	}
	if err := qtx.UpsertArtwork(ctx, db.UpsertArtworkParams{
		ID:            id,
		GameID:        gameID,
		Kind:          kind,
		Origin:        origin,
		SourceRelPath: source,
		CacheKey:      cacheKey,
		MimeType:      "image/jpeg",
		Width:         int64(cfg.Width),
		Height:        int64(cfg.Height),
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		_ = tx.Rollback()
		return db.Artwork{}, fmt.Errorf("save artwork: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return db.Artwork{}, fmt.Errorf("commit artwork: %w", err)
	}

	if hadExisting && existing.CacheKey != cacheKey {
		s.removeVariant(existing.CacheKey, "card")
		s.removeVariant(existing.CacheKey, "detail")
	}

	row, err := q.GetArtwork(ctx, id)
	if err != nil {
		return db.Artwork{}, fmt.Errorf("read saved artwork: %w", err)
	}
	return row, nil
}

// SyncLibrary discovers artwork for every available game in the library.
func (s *Service) SyncLibrary(ctx context.Context, q *db.Queries, lib db.Library) error {
	games, err := q.ListGamesForLibrary(ctx, lib.ID)
	if err != nil {
		return fmt.Errorf("list games for artwork: %w", err)
	}
	for _, game := range games {
		if game.AvailabilityStatus != "available" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.SyncGame(ctx, lib.RootPath, game); err != nil {
			s.logger.Warn("could not sync game artwork", "game_id", game.ID, "error", err)
		}
	}
	return nil
}

// SyncGame discovers and processes artwork for one game. User uploads are not
// replaced by detected files.
func (s *Service) SyncGame(ctx context.Context, libraryRoot string, game db.ListGamesForLibraryRow) error {
	gameDir := filepath.Join(libraryRoot, filepath.FromSlash(game.RelativePath))
	if !withinRoot(libraryRoot, gameDir) {
		return fmt.Errorf("game path escapes the library root")
	}
	candidates, err := scanner.FindArtworkCandidates(gameDir)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	q := db.New(s.store.DB)
	existingRows, err := q.ListArtworkForGame(ctx, game.ID)
	if err != nil {
		return fmt.Errorf("list existing artwork: %w", err)
	}
	existing := make(map[string]db.Artwork, len(existingRows))
	for _, row := range existingRows {
		existing[row.Kind] = row
	}

	processed := make(map[string]bool)
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if processed[string(candidate.Kind)] {
			continue
		}
		row, ok := existing[string(candidate.Kind)]
		if ok && row.Origin == "upload" {
			processed[string(candidate.Kind)] = true
			continue
		}
		sourcePath := filepath.Join(gameDir, filepath.FromSlash(candidate.RelPath))
		if !withinRoot(gameDir, sourcePath) {
			continue
		}
		if _, err := s.ProcessFile(ctx, game.ID, string(candidate.Kind), sourcePath, candidate.RelPath, "detected"); err != nil {
			s.logger.Debug("artwork candidate rejected", "game_id", game.ID, "path", candidate.RelPath, "error", err)
			continue
		}
		processed[string(candidate.Kind)] = true
	}

	for kind, row := range existing {
		if row.Origin == "detected" && !processed[kind] {
			if err := s.deleteArtwork(ctx, game.ID, kind); err != nil {
				return err
			}
		}
	}
	return nil
}

// VariantPath returns the cache path and content type for a variant.
func (s *Service) VariantPath(ctx context.Context, artworkID, variant string) (string, string, error) {
	if variant != "card" && variant != "detail" {
		return "", "", fmt.Errorf("invalid artwork variant")
	}
	row, err := db.New(s.store.DB).GetArtwork(ctx, artworkID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("get artwork: %w", err)
	}
	path := s.variantPath(row.CacheKey, variant)
	if _, err := os.Stat(path); err != nil {
		return "", "", ErrNotFound
	}
	return path, "image/jpeg", nil
}

func (s *Service) deleteArtwork(ctx context.Context, gameID, kind string) error {
	q := db.New(s.store.DB)
	row, err := q.GetArtworkForGame(ctx, db.GetArtworkForGameParams{GameID: gameID, Kind: kind})
	if err != nil {
		return nil
	}
	if err := q.DeleteArtwork(ctx, db.DeleteArtworkParams{GameID: gameID, Kind: kind}); err != nil {
		return fmt.Errorf("delete artwork: %w", err)
	}
	s.removeVariant(row.CacheKey, "card")
	s.removeVariant(row.CacheKey, "detail")
	return nil
}

func (s *Service) variantPath(cacheKey, variant string) string {
	return filepath.Join(s.cacheDir, cacheKey+"-"+variant+".jpg")
}

func (s *Service) variantExists(cacheKey, variant string) bool {
	info, err := os.Stat(s.variantPath(cacheKey, variant))
	return err == nil && info.Mode().IsRegular()
}

func (s *Service) removeVariant(cacheKey, variant string) {
	_ = os.Remove(s.variantPath(cacheKey, variant))
}

func (s *Service) writeVariant(src image.Image, maxWidth int, path string) error {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 1 || height < 1 {
		return fmt.Errorf("invalid source image")
	}
	targetWidth := min(maxWidth, width)
	targetHeight := int(float64(height) * float64(targetWidth) / float64(width))
	if targetHeight < 1 {
		targetHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	background := &image.Uniform{C: color.RGBA{R: 18, G: 17, B: 15, A: 255}}
	draw.Draw(dst, dst.Bounds(), background, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	temp, err := os.CreateTemp(filepath.Dir(path), ".artwork-*.jpg")
	if err != nil {
		return fmt.Errorf("create artwork variant: %w", err)
	}
	tempName := temp.Name()
	if err := jpeg.Encode(temp, dst, &jpeg.Options{Quality: 86}); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempName)
		return fmt.Errorf("encode artwork variant: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("close artwork variant: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return fmt.Errorf("publish artwork variant: %w", err)
	}
	return nil
}

func withinRoot(root, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
