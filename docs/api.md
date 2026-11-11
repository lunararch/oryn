# HTTP API

Base URL: `/api/v1`. Responses are JSON unless noted. Timestamps are UTC RFC 3339 strings.

## Errors

```json
{ "error": { "code": "not_found", "message": "game not found" } }
```

No stack traces, SQL, absolute server paths, or secrets are returned.

## Health and version

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Process, database, and library availability |
| GET | `/version` | Server version |
| GET | `/libraries` | Configured library summary and game count |
| GET | `/stats` | Total, available, missing, and favorite counts |

`health` returns `200` while the process/database are healthy. `library` is `available`, `unavailable`, or `not_configured`. A missing game share does not make the process unhealthy, but it is visible in the body.

## Games

| Method | Path | Purpose |
|---|---|---|
| GET | `/games` | List/search/filter/sort games |
| GET | `/games/{id}` | Game details with resolved overrides |
| PATCH | `/games/{id}` | Replace editable metadata |
| PUT | `/games/{id}/favorite` | Mark favorite (idempotent) |
| DELETE | `/games/{id}/favorite` | Remove favorite (idempotent) |
| POST | `/games/{id}/artwork` | Upload cover/banner (`multipart/form-data`) |
| DELETE | `/games/{id}/artwork/{kind}` | Remove cover/banner; `kind` is `cover` or `banner` |
| GET | `/artwork/{artworkId}/{variant}` | Cached image; `variant` is `card` or `detail` |

### List query parameters

| Parameter | Values | Default |
|---|---|---|
| `q` | text search across title, description, folder, developer, publisher | empty |
| `page` | positive integer | `1` |
| `pageSize` | 1–100 | `60` |
| `sort` | `title`, `added`, `updated`, `lastSeen`, `path` | `title` |
| `order` | `asc`, `desc` | `asc` for title, otherwise `desc` |
| `favorite` | `true`, `false` | unset |
| `played` | `true`, `false` (has a completed/interrupted session) | unset |
| `availability` | `available`, `missing` | unset |
| `completion` | `unplayed`, `playing`, `completed`, `dropped`, `on_hold` | unset |
| `collection` | collection ID | unset |

Response:

```json
{
  "items": [
    {
      "id": "01...",
      "title": "Example",
      "description": "...",
      "developer": "",
      "publisher": "",
      "releaseDate": "",
      "version": "1.0",
      "relativePath": "Example Game",
      "availability": "available",
      "completionStatus": "unplayed",
      "rating": 8,
      "notes": "",
      "favorite": true,
      "coverArtworkId": "01...",
      "bannerArtworkId": "",
      "addedAt": "2026-01-01T12:00:00Z",
      "lastSeenAt": "2026-01-01T12:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 60,
  "total": 1,
  "totalPages": 1
}
```

### Metadata update

`PATCH /games/{id}` accepts the complete editable metadata object:

```json
{
  "title": "Custom title",
  "description": "Custom description",
  "developer": "Studio",
  "publisher": "",
  "releaseDate": "2025",
  "rating": 8,
  "notes": "Private notes",
  "completionStatus": "playing"
}
```

All fields are applied. `null` or an empty string clears the override for that field; the detected value becomes visible again. Rating must be 1–10, or `null` to clear. Completion must be one of the accepted values, or `null`/empty to clear.

The response is the updated game.

### Artwork upload

`POST /games/{id}/artwork` with `multipart/form-data`:

- `kind`: `cover` or `banner` (defaults to `cover`)
- `file`: PNG, JPEG, or WebP; maximum 10 MiB and 8192 px per side

The server validates decoded content, generates variants, stores them outside the game share, and returns the updated game.

## Scans

| Method | Path | Purpose |
|---|---|---|
| POST | `/scans` | Start a background scan |
| GET | `/scans/current` | Current or most recent scan state |
| GET | `/scans` | Recent scan runs (latest 25) |
| GET | `/scans/{id}` | Scan run details |
| POST | `/scans/{id}/cancel` | Request cancellation |

Start request:

```json
{ "allowMissing": false }
```

`allowMissing` bypasses the mass-missing guard. It should require explicit user confirmation in the UI.

Returns `202` with a scan state. A second start while one is running returns `409 scan_in_progress`.

Scan statuses: `running`, `completed`, `partial`, `failed`, `cancelled`.

A failed or partial scan never marks games missing. If a readable root discovers zero games while games are already indexed, the scan fails and preserves the library.

## Play sessions

| Method | Path | Purpose |
|---|---|---|
| GET | `/games/{id}/sessions` | Session history for a game |
| POST | `/games/{id}/sessions` | Record a manual completed session |
| PATCH | `/sessions/{id}` | Correct a manual session |
| DELETE | `/sessions/{id}` | Delete a manual session |

Manual session request:

```json
{
  "startedAt": "2026-01-01T18:00:00Z",
  "endedAt": "2026-01-01T19:30:00Z",
  "notes": "Finished the first route"
}
```

Timestamps must be RFC3339. End must be after start; sessions cannot end in the future, be longer than seven days, or start more than five minutes in the future. Only `source=manual` sessions can be edited or deleted.

Session response:

```json
{
  "id": "01...",
  "gameId": "01...",
  "startedAt": "2026-01-01T18:00:00Z",
  "endedAt": "2026-01-01T19:30:00Z",
  "durationSeconds": 5400,
  "status": "completed",
  "source": "manual",
  "notes": "Finished the first route"
}
```

Playtime totals include `completed` and `interrupted` sessions. Running sessions are not yet part of the web API; desktop tracking arrives with the desktop client.

## Collections

| Method | Path | Purpose |
|---|---|---|
| GET | `/collections` | List collections with game counts |
| POST | `/collections` | Create a collection |
| GET | `/collections/{id}` | Collection details |
| PATCH | `/collections/{id}` | Rename/update description |
| DELETE | `/collections/{id}` | Delete the collection (games are untouched) |
| PUT | `/collections/{id}/games/{gameId}` | Add game (idempotent) |
| DELETE | `/collections/{id}/games/{gameId}` | Remove game |
| GET | `/games/{id}/collections` | All collections with `contains` flags |

Collection names are unique case-insensitively and limited to 100 characters; descriptions are limited to 2000 characters. Collections do not move or modify game folders.

## Authentication

There is no user authentication yet. Oryn is designed for a trusted home LAN and must not be exposed directly to the internet. Desktop pairing tokens and revocable client credentials are planned before desktop session endpoints are added.
