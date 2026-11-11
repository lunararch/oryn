import { expect, test } from "@playwright/test";

interface MockGame {
  id: string;
  title: string;
  description: string;
  developer: string;
  publisher: string;
  releaseDate: string;
  version: string;
  relativePath: string;
  availability: string;
  completionStatus: string;
  favorite: boolean;
  notes: string;
  addedAt: string;
  lastSeenAt: string;
  playtimeSeconds: number;
}

function makeGame(overrides: Partial<MockGame> = {}): MockGame {
  return {
    id: "game-1",
    title: "Alpha",
    description: "A quiet story.",
    developer: "Paper Lantern",
    publisher: "",
    releaseDate: "2025",
    version: "1.0",
    relativePath: "Alpha",
    availability: "available",
    completionStatus: "unplayed",
    favorite: false,
    notes: "",
    addedAt: "2025-01-02T03:04:05Z",
    lastSeenAt: "2025-01-02T03:04:05Z",
    playtimeSeconds: 3600,
    ...overrides,
  };
}

test("dashboard and web-only capabilities", async ({ page }) => {
  const games = [makeGame(), makeGame({ id: "game-2", title: "Beta", relativePath: "Beta" })];
  await mockAPI(page, () => games);

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Your stories, kept close." })).toBeVisible();
  await expect(page.getByText("Alpha").first()).toBeVisible();
  await expect(page.getByRole("button", { name: /launch/i })).toHaveCount(0);
});

test("searches the library and toggles a favorite", async ({ page }) => {
  const games = [makeGame(), makeGame({ id: "game-2", title: "Beta", relativePath: "Beta" })];
  await mockAPI(page, () => games);

  await page.goto("/library");
  await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();
  await expect(page.getByText("Alpha")).toBeVisible();
  await expect(page.getByText("Beta")).toBeVisible();

  await page.getByPlaceholder(/search titles/i).fill("Beta");
  await expect(page.getByRole("heading", { name: "Alpha" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Beta" })).toBeVisible();

  await page.getByPlaceholder(/search titles/i).fill("");
  await expect(page.getByRole("heading", { name: "Alpha" })).toBeVisible();
  await page.getByRole("button", { name: "Add to favorites" }).first().click();
  await expect(page.getByRole("button", { name: "Remove from favorites" }).first()).toBeVisible();
});

test("edits metadata from the detail page", async ({ page }) => {
  const games = [makeGame()];
  await mockAPI(page, () => games);

  await page.goto("/games/game-1");
  await expect(page.getByRole("heading", { name: "Alpha" })).toBeVisible();
  await page.getByRole("button", { name: "Edit details" }).click();

  const title = page.getByLabel("Title");
  await title.fill("Alpha Custom");
  await page.getByRole("button", { name: "Save details" }).click();

  await expect(page.getByRole("heading", { name: "Alpha Custom" })).toBeVisible();
});

async function mockAPI(
  page: import("@playwright/test").Page,
  getGames: () => MockGame[],
): Promise<void> {
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const games = getGames();

    if (path === "/api/v1/stats") {
      return fulfill(route, 200, {
        total: games.length,
        available: games.length,
        missing: 0,
        favorites: games.filter((game) => game.favorite).length,
        playedGames: 1,
        playtimeSeconds: 3600,
      });
    }
    if (path === "/api/v1/libraries") {
      return fulfill(route, 200, {
        items: [
          {
            id: "library-1",
            name: "Test Library",
            gameCount: games.length,
            lastScanAt: "2025-01-02T03:04:05Z",
          },
        ],
      });
    }
    if (path === "/api/v1/scans/current") {
      return fulfill(route, 200, {
        running: false,
        status: "completed",
        discovered: games.length,
        added: 0,
        updated: 0,
        unchanged: games.length,
        missing: 0,
        errors: 0,
        summary: "scan completed",
      });
    }
    if (path === "/api/v1/scans" && request.method() === "GET") {
      return fulfill(route, 200, { items: [] });
    }
    if (path === "/api/v1/scans" && request.method() === "POST") {
      return fulfill(route, 202, { running: true, status: "running" });
    }
    if (path === "/api/v1/collections") {
      return fulfill(route, 200, { items: [] });
    }
    if (/^\/api\/v1\/games\/[^/]+\/favorite$/.test(path)) {
      const gameID = path.split("/")[4];
      const game = games.find((item) => item.id === gameID);
      if (game) game.favorite = request.method() === "PUT";
      return route.fulfill({ status: 204, body: "" });
    }
    if (/^\/api\/v1\/games\/[^/]+\/sessions$/.test(path)) {
      return fulfill(route, 200, { items: [], page: 1, pageSize: 20, total: 0 });
    }
    if (/^\/api\/v1\/games\/[^/]+\/collections$/.test(path)) {
      return fulfill(route, 200, { items: [] });
    }
    if (/^\/api\/v1\/games\/[^/]+$/.test(path) && request.method() === "PATCH") {
      const gameID = path.split("/").pop();
      const game = games.find((item) => item.id === gameID);
      const patch = request.postDataJSON() as Partial<MockGame>;
      if (game) Object.assign(game, patch);
      return fulfill(route, 200, game);
    }
    if (/^\/api\/v1\/games\/[^/]+$/.test(path)) {
      const gameID = path.split("/").pop();
      return fulfill(
        route,
        200,
        games.find((item) => item.id === gameID),
      );
    }
    if (path === "/api/v1/games") {
      const query = (url.searchParams.get("q") ?? "").toLowerCase();
      const filtered = games.filter(
        (game) =>
          !query ||
          game.title.toLowerCase().includes(query) ||
          game.description.toLowerCase().includes(query),
      );
      return fulfill(route, 200, {
        items: filtered,
        page: 1,
        pageSize: 60,
        total: filtered.length,
        totalPages: 1,
      });
    }

    return fulfill(route, 404, { error: { code: "not_found", message: "not found" } });
  });
}

async function fulfill(
  route: import("@playwright/test").Route,
  status: number,
  body: unknown,
): Promise<void> {
  await route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}
