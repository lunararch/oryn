# Architecture

Oryn is a self-hosted visual-novel library. The Raspberry Pi server indexes the game share and owns shared library data; the browser can browse and organize it; a Wails desktop client will launch games locally in a later milestone.

## Component ownership

| Responsibility | Owner |
|---|---|
| Library index, detected metadata | Server |
| Favorites, overrides, notes, ratings, completion | Server |
| SQLite and migrations | Server |
| Scanning the mounted game share | Server |
| Artwork validation, thumbnails, cache | Server |
| Web browsing and management | Shared React frontend |
| Launching native games | Desktop client (planned) |
| Monitoring local game processes | Desktop client (planned) |
| Authoritative play-session history | Server (planned schema/API) |

The game share is the source of truth for game files. SQLite is the source of truth for Oryn data. Clients never open SQLite directly.

## Server request flow

```text
HTTP handler (internal/httpapi)
    ↓
Application/service (internal/games, internal/scanner, internal/artwork)
    ↓
SQLite via sqlc-generated queries (internal/store/db)
    ↓
Mounted game share (read-only)
```

`internal/app` owns configuration, database startup, service wiring, HTTP lifecycle, static SPA serving, and the post-scan artwork hook.

## Implemented today

- Go server with Chi, structured logging, graceful shutdown, health/version endpoints.
- SQLite with embedded Goose migrations and sqlc-generated queries.
- Safe recursive scanner with confidence-based Ren'Py detection, conservative metadata extraction, move/rename detection, and mount safeguards.
- Idempotent scans with history, cancellation, partial-error handling, and mass-missing protection.
- Artwork discovery, validation, cached card/detail variants, and user uploads.
- React web library with dashboard, search/filter/sort, grid/list views, details, favorites, metadata editing, scan history, and artwork controls.
- Collections with create/edit/delete and membership toggles.
- Manual play sessions, playtime totals, recently played, and session history.
- Desktop client pairing, revocable bearer tokens, authenticated session start/heartbeat/finish, and stale-session recovery.
- Desktop core module: safe path mapping, launch candidate detection, shell-free process execution, and session reporting.
- Docker Compose deployment for ARM64 with a read-only game mount and an optional sentinel.
- SQLite `VACUUM INTO` backup command.
- Playwright browser tests using a mocked API.

Not implemented yet: the Wails 3 desktop shell, Wine/Proton, multi-user support, external metadata providers, and offline desktop cache. The Wails CLI could not be built in the development environment because GTK4/`webkitgtk-6.0` development files are not installed; the desktop core is tested independently and is ready to bind to a Wails service layer.

## Scanning safety

The scanner treats the game share as untrusted and read-only.

- Detection uses multiple signals: `game/`, `.rpy`/`.rpyc`, `renpy/`, `lib/<platform>/`, and launcher files. A directory with only `game/` is not enough.
- A detected game directory is not descended into; `saves/`, caches, and runtime internals are never scanned.
- Directory symlinks are not followed.
- Metadata is parsed with conservative line patterns from `game/options.rpy`; no Ren'Py or Python code is executed.
- A scan records `running`, `completed`, `partial`, `failed`, or `cancelled`.
- A missing/unreadable root fails the scan without changing game availability.
- If a readable root yields zero games while the database has known games, the scan fails and preserves records.
- Missing games are only reconciled after a clean scan with no traversal errors.
- If more than 25% of known available games (and more than 10) would be marked missing, reconciliation is skipped unless the caller explicitly allows it.
- Renames/moves are matched only when a stable fingerprint has exactly one missing candidate. Otherwise a new record is created and the old one remains missing.
- User metadata lives in `game_overrides` and is never overwritten by scans. Uploaded artwork is never replaced by detected artwork.

## Data model

Current tables:

  - `game_overrides`
  - `favorites`
  - `artwork`
  - `scan_runs`
  - `scan_errors`
  - `settings`
  - `play_sessions`
  - `collections`, `collection_games`
  - `desktop_clients`, `pairing_codes`

Future migrations will add richer genre/tag data. Do not add those tables until the behavior is implemented.

Paths are stored relative to the library root, using `/` separators. Absolute server paths are deployment details and are not exposed by the API.

## API and contracts

- API prefix: `/api/v1`.
- Go types in `apps/server/internal/api` are the response contract.
- TypeScript types in `apps/frontend/src/lib/api/generated.ts` are generated with `tygo`.
- Errors use `{"error":{"code":"...","message":"..."}}`.
- Timestamps are UTC RFC 3339 strings in API responses; the database stores Unix seconds.
- Lists are paginated and bounded.

Live scan updates currently use polling (`/scans/current`). SSE or WebSockets should only be introduced when polling is measurably insufficient.

## Artwork

- Candidate discovery covers root covers, `game/cover.*`, GUI covers, main-menu/banner names, menu backgrounds, `Oblojka`, icons, and window icons.
- Files are validated by decoded content, format, dimensions, pixel count, and size. SVG and arbitrary file types are rejected.
- Card (`480px`) and detail (`1400px`) variants are generated into the application data directory, never into the game share.
- Cache filenames are content-hashed and served with immutable cache headers.
- Uploading artwork replaces the current selection and gives it a new ID, avoiding stale browser caches.
- Removing artwork deletes the row/cache; the next clean scan may detect artwork again.

## Security boundaries

- The server never executes game content.
- The API is intended for a trusted LAN. It has no user authentication yet; do not expose it to the public internet.
- Desktop pairing/token authentication is planned before desktop session or client-management operations are added.
- CORS is same-origin by default; the Vite development proxy is the only local exception.
- Path containment checks are applied to library paths, artwork sources, and the SPA static root.
- Uploads are size-limited and content-validated.
