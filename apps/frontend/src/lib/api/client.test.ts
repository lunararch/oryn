import { afterEach, describe, expect, it, vi } from "vitest";
import { apiRequest, listGames, uploadArtwork } from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("apiRequest", () => {
  it("sends requests under the API prefix and parses JSON", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ total: 2, items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiRequest<{ total: number; items: unknown[] }>("/games");
    expect(result.total).toBe(2);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/games",
      expect.objectContaining({ headers: expect.any(Headers) }),
    );
  });

  it("maps API errors to ApiError with status and code", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: "not_found", message: "Game not found" } }), {
          status: 404,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );

    await expect(apiRequest("/games/missing")).rejects.toMatchObject({
      name: "ApiError",
      status: 404,
      code: "not_found",
      message: "Game not found",
    });
  });
});

describe("listGames", () => {
  it("serializes filters and pagination", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [], total: 0, page: 1, pageSize: 24, totalPages: 0 }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await listGames({
      q: "moon light",
      favorite: true,
      page: 2,
      pageSize: 24,
      sort: "title",
      order: "asc",
      completion: "completed",
    });

    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain("/api/v1/games?");
    expect(url).toContain("q=moon+light");
    expect(url).toContain("favorite=true");
    expect(url).toContain("page=2");
    expect(url).toContain("pageSize=24");
    expect(url).toContain("sort=title");
    expect(url).toContain("order=asc");
    expect(url).toContain("completion=completed");
  });
});

describe("uploadArtwork", () => {
  it("sends multipart form data without a JSON content type", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "game-1", title: "Game" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const file = new File(["image"], "cover.png", { type: "image/png" });
    await uploadArtwork("game-1", "cover", file);

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.method).toBe("POST");
    expect(init.body).toBeInstanceOf(FormData);
    const headers = new Headers(init.headers);
    expect(headers.get("Content-Type")).toBeNull();
  });
});
