import type {
  Game,
  GamesResponse,
  LibrariesResponse,
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
  if (init.body && !headers.has("Content-Type")) {
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
  availability?: string;
  completion?: string;
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

export function getLibraries(): Promise<LibrariesResponse> {
  return apiRequest<LibrariesResponse>("/libraries");
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

export type { Game, ScanRun, ScanState, Stats };
