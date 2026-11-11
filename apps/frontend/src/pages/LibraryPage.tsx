import { LayoutGrid, List, LibraryBig, Search, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { GameGrid, GameGridSkeleton } from "../components/GameGrid";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { useGames } from "../lib/api/queries";
import type { GameView } from "../components/GameCard";
import { useDebouncedValue } from "../lib/useDebouncedValue";
import { formatNumber } from "../lib/format";

const sortOptions = [
  { value: "title:asc", label: "Title A–Z" },
  { value: "title:desc", label: "Title Z–A" },
  { value: "added:desc", label: "Recently added" },
  { value: "lastSeen:desc", label: "Recently seen" },
];

const filters = [
  { key: "all", label: "All" },
  { key: "favorites", label: "Favorites" },
  { key: "unplayed", label: "Unplayed" },
  { key: "playing", label: "Playing" },
  { key: "completed", label: "Completed" },
  { key: "missing", label: "Missing" },
] as const;

export function LibraryPage() {
  const [params, setParams] = useSearchParams();
  const q = params.get("q") ?? "";
  const sort = params.get("sort") ?? "title";
  const order = params.get("order") ?? "asc";
  const page = Math.max(1, Number(params.get("page") ?? 1) || 1);
  const favorite = params.get("favorite") === "true";
  const completion = params.get("completion") ?? "";
  const availability = params.get("availability") ?? "";

  const [search, setSearch] = useState(q);
  const debouncedSearch = useDebouncedValue(search, 280);
  const [view, setView] = useState<GameView>(() => {
    const stored = window.localStorage.getItem("oryn.library.view");
    return stored === "list" ? "list" : "grid";
  });

  useEffect(() => {
    if (debouncedSearch === q) return;
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous);
        if (debouncedSearch) next.set("q", debouncedSearch);
        else next.delete("q");
        next.delete("page");
        return next;
      },
      { replace: true },
    );
  }, [debouncedSearch, q, setParams]);

  useEffect(() => {
    window.localStorage.setItem("oryn.library.view", view);
  }, [view]);

  const activeFilter = useMemo(() => {
    if (favorite) return "favorites";
    if (availability === "missing") return "missing";
    if (completion) return completion;
    return "all";
  }, [availability, completion, favorite]);

  const games = useGames({
    q: q || undefined,
    favorite: favorite || undefined,
    completion: completion || undefined,
    availability: availability || undefined,
    sort,
    order,
    page,
    pageSize: 60,
  });

  function chooseFilter(key: string) {
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      next.delete("favorite");
      next.delete("completion");
      next.delete("availability");
      next.delete("page");
      if (key === "favorites") next.set("favorite", "true");
      else if (key === "missing") next.set("availability", "missing");
      else if (key !== "all") next.set("completion", key);
      return next;
    });
  }

  function changeSort(value: string) {
    const [nextSort, nextOrder] = value.split(":");
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      next.set("sort", nextSort);
      next.set("order", nextOrder);
      next.delete("page");
      return next;
    });
  }

  function clearFilters() {
    setSearch("");
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      for (const key of ["q", "favorite", "completion", "availability", "page"]) next.delete(key);
      return next;
    });
  }

  const total = games.data?.total ?? 0;
  const totalPages = games.data?.totalPages ?? 0;
  const hasFilters = Boolean(q || favorite || completion || availability);

  return (
    <div className="space-y-7">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">The shelf</p>
          <h1 className="page-title mt-2">Library</h1>
          <p className="mt-3 max-w-xl text-sm leading-relaxed text-paper-muted">
            Every game Oryn has found, with the details you choose to keep.
          </p>
        </div>
        <div className="flex items-baseline gap-2 text-sm text-paper-muted">
          <span className="font-display text-3xl text-paper">{formatNumber(total)}</span>
          {total === 1 ? "game" : "games"}
        </div>
      </div>

      <div className="surface p-3">
        <div className="flex flex-col gap-3 xl:flex-row xl:items-center">
          <label className="relative min-w-0 flex-1">
            <span className="sr-only">Search games</span>
            <Search
              size={17}
              className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-paper-faint"
            />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search titles, descriptions, developers…"
              className="pl-10"
            />
            {search ? (
              <button
                type="button"
                onClick={() => setSearch("")}
                aria-label="Clear search"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-paper-faint transition hover:text-paper"
              >
                <X size={15} />
              </button>
            ) : null}
          </label>

          <div className="flex flex-wrap items-center gap-2">
            <Select
              aria-label="Sort games"
              value={`${sort}:${order}`}
              onChange={(event) => changeSort(event.target.value)}
              className="w-auto min-w-40"
            >
              {sortOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
            <div
              className="flex rounded-full border border-white/10 bg-ink-950/40 p-1"
              role="group"
              aria-label="View mode"
            >
              <button
                type="button"
                aria-label="Grid view"
                aria-pressed={view === "grid"}
                onClick={() => setView("grid")}
                className={`grid h-8 w-9 place-items-center rounded-full transition ${
                  view === "grid" ? "bg-brass/15 text-brass-soft" : "text-paper-muted hover:text-paper"
                }`}
              >
                <LayoutGrid size={15} />
              </button>
              <button
                type="button"
                aria-label="List view"
                aria-pressed={view === "list"}
                onClick={() => setView("list")}
                className={`grid h-8 w-9 place-items-center rounded-full transition ${
                  view === "list" ? "bg-brass/15 text-brass-soft" : "text-paper-muted hover:text-paper"
                }`}
              >
                <List size={15} />
              </button>
            </div>
          </div>
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-white/7 pt-3">
          {filters.map((filter) => (
            <button
              key={filter.key}
              type="button"
              onClick={() => chooseFilter(filter.key)}
              aria-pressed={activeFilter === filter.key}
              className={`rounded-full border px-3 py-1.5 text-xs font-medium transition ${
                activeFilter === filter.key
                  ? "border-brass/35 bg-brass/12 text-brass-soft"
                  : "border-white/9 text-paper-muted hover:border-white/16 hover:text-paper"
              }`}
            >
              {filter.label}
            </button>
          ))}
          {hasFilters ? (
            <button
              type="button"
              onClick={clearFilters}
              className="ml-auto flex items-center gap-1.5 text-xs text-paper-faint transition hover:text-paper"
            >
              <X size={13} />
              Clear filters
            </button>
          ) : null}
        </div>
      </div>

      {games.isError ? (
        <div className="surface flex flex-wrap items-center justify-between gap-4 px-5 py-4">
          <div>
            <p className="font-medium text-rust-soft">Could not load the library</p>
            <p className="mt-1 text-sm text-paper-muted">{games.error.message}</p>
          </div>
          <Button onClick={() => void games.refetch()}>Try again</Button>
        </div>
      ) : games.isLoading ? (
        <GameGridSkeleton count={10} view={view} />
      ) : games.data && games.data.items.length === 0 ? (
        hasFilters ? (
          <EmptyState
            icon={Search}
            title="No games match those filters"
            description="Try a different search or clear the filters to see the whole shelf."
            action={<Button onClick={clearFilters}>Clear filters</Button>}
          />
        ) : (
          <EmptyState
            icon={LibraryBig}
            title="No games found yet"
            description="Run a scan from the dashboard or settings to discover the visual novels in your library folder."
          />
        )
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            {q ? <Badge tone="quiet">Search · {q}</Badge> : null}
            {favorite ? <Badge tone="brass">Favorites</Badge> : null}
            {completion ? <Badge tone="brass">{completion.replace("_", " ")}</Badge> : null}
            {availability ? <Badge tone="danger">{availability}</Badge> : null}
          </div>
          <GameGrid games={games.data?.items ?? []} view={view} />

          {totalPages > 1 ? (
            <nav className="flex items-center justify-between gap-3 pt-2" aria-label="Pagination">
              <Button
                variant="secondary"
                disabled={page <= 1}
                onClick={() =>
                  setParams((previous) => {
                    const next = new URLSearchParams(previous);
                    next.set("page", String(page - 1));
                    return next;
                  })
                }
              >
                Previous
              </Button>
              <span className="text-sm text-paper-muted">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="secondary"
                disabled={page >= totalPages}
                onClick={() =>
                  setParams((previous) => {
                    const next = new URLSearchParams(previous);
                    next.set("page", String(page + 1));
                    return next;
                  })
                }
              >
                Next
              </Button>
            </nav>
          ) : null}
        </>
      )}
    </div>
  );
}
