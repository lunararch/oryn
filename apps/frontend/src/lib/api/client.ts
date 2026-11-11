import type {
  Collection,
  CollectionMembershipsResponse,
  CollectionsResponse,
  Game,
  GamesResponse,
  LibrariesResponse,
  PlaySession,
  PlaySessionsResponse,
  ScanRun,
  ScanRunsResponse,
  ScanState,
  Stats,
  UpdateGameRequest,
} from "./generated";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (typeof init.body === "string" && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!response.ok) {
    let code = "request_failed";
    let message = `Request failed with status ${response.status}`;
    try {
      const body = (await response.json()) as { error?: { code?: string; message?: string } };
      if (body.error?.code) code = body.error.code;
      if (body.error?.message) message = body.error.message;
    } catch {
      // Keep the safe fallback message.
    }
    throw new ApiError(response.status, code, message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export interface ListGamesParams {
  q?: string;
  page?: number;
  pageSize?: number;
  sort?: string;
  order?: string;
  favorite?: boolean;
  played?: boolean;
  availability?: string;
  completion?: string;
  collection?: string;
}

export function listGames(params: ListGamesParams = {}): Promise<GamesResponse> {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "" || value === null) continue;
    search.set(key, String(value));
  }
  const query = search.toString();
  return apiRequest<GamesResponse>(`/games${query ? `?${query}` : ""}`);
}

export function getGame(id: string): Promise<Game> {
  return apiRequest<Game>(`/games/${encodeURIComponent(id)}`);
}

export function updateGame(id: string, body: UpdateGameRequest): Promise<Game> {
  return apiRequest<Game>(`/games/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function setFavorite(id: string, favorite: boolean): Promise<void> {
  return apiRequest<void>(`/games/${encodeURIComponent(id)}/favorite`, {
    method: favorite ? "PUT" : "DELETE",
  });
}

export function getStats(): Promise<Stats> {
  return apiRequest<Stats>("/stats");
}

export function uploadArtwork(gameID: string, kind: "cover" | "banner", file: File): Promise<Game> {
  const body = new FormData();
  body.append("kind", kind);
  body.append("file", file);
  return apiRequest<Game>(`/games/${encodeURIComponent(gameID)}/artwork`, {
    method: "POST",
    body,
  });
}

export function deleteArtwork(gameID: string, kind: "cover" | "banner"): Promise<Game> {
  return apiRequest<Game>(`/games/${encodeURIComponent(gameID)}/artwork/${kind}`, {
    method: "DELETE",
  });
}

export function getLibraries(): Promise<LibrariesResponse> {
  return apiRequest<LibrariesResponse>("/libraries");
}

export interface SessionInput {
  startedAt: string;
  endedAt: string;
  notes: string;
}

export function listSessions(
  gameID: string,
  page = 1,
  pageSize = 20,
): Promise<PlaySessionsResponse> {
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  return apiRequest<PlaySessionsResponse>(
    `/games/${encodeURIComponent(gameID)}/sessions?${search.toString()}`,
  );
}

export function createSession(gameID: string, input: SessionInput): Promise<PlaySession> {
  return apiRequest<PlaySession>(`/games/${encodeURIComponent(gameID)}/sessions`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateSession(sessionID: string, input: SessionInput): Promise<PlaySession> {
  return apiRequest<PlaySession>(`/sessions/${encodeURIComponent(sessionID)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export function deleteSession(sessionID: string): Promise<void> {
  return apiRequest<void>(`/sessions/${encodeURIComponent(sessionID)}`, {
    method: "DELETE",
  });
}

export function listCollections(): Promise<CollectionsResponse> {
  return apiRequest<CollectionsResponse>("/collections");
}

export function getCollection(id: string): Promise<Collection> {
  return apiRequest<Collection>(`/collections/${encodeURIComponent(id)}`);
}

export function createCollection(name: string, description: string): Promise<Collection> {
  return apiRequest<Collection>("/collections", {
    method: "POST",
    body: JSON.stringify({ name, description }),
  });
}

export function updateCollection(
  id: string,
  name: string,
  description: string,
): Promise<Collection> {
  return apiRequest<Collection>(`/collections/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify({ name, description }),
  });
}

export function deleteCollection(id: string): Promise<void> {
  return apiRequest<void>(`/collections/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listCollectionsForGame(gameID: string): Promise<CollectionMembershipsResponse> {
  return apiRequest<CollectionMembershipsResponse>(
    `/games/${encodeURIComponent(gameID)}/collections`,
  );
}

export function addGameToCollection(collectionID: string, gameID: string): Promise<void> {
  return apiRequest<void>(
    `/collections/${encodeURIComponent(collectionID)}/games/${encodeURIComponent(gameID)}`,
    { method: "PUT" },
  );
}

export function removeGameFromCollection(collectionID: string, gameID: string): Promise<void> {
  return apiRequest<void>(
    `/collections/${encodeURIComponent(collectionID)}/games/${encodeURIComponent(gameID)}`,
    { method: "DELETE" },
  );
}

export function startScan(allowMissing = false): Promise<ScanState> {
  return apiRequest<ScanState>("/scans", {
    method: "POST",
    body: JSON.stringify({ allowMissing }),
  });
}

export function getCurrentScan(): Promise<ScanState> {
  return apiRequest<ScanState>("/scans/current");
}

export function listScans(): Promise<ScanRunsResponse> {
  return apiRequest<ScanRunsResponse>("/scans");
}

export type { Collection, Game, ScanRun, ScanState, Stats };
