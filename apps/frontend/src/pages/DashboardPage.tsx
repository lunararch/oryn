import { ArrowRight, Clock3, Heart, LibraryBig, ScanLine } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { GameGrid, GameGridSkeleton } from "../components/GameGrid";
import { GamePoster } from "../components/GamePoster";
import { ScanStatus } from "../components/ScanStatus";
import { EmptyState } from "../components/ui/EmptyState";
import { useGames, useLibraries, useStats } from "../lib/api/queries";
import { formatDate, formatNumber } from "../lib/format";

export function DashboardPage() {
  const stats = useStats();
  const libraries = useLibraries();
  const recent = useGames({ sort: "added", order: "desc", pageSize: 6 });
  const favorites = useGames({ favorite: true, sort: "title", order: "asc", pageSize: 6 });

  const library = libraries.data?.items[0];
  const total = stats.data?.total ?? 0;
  const isEmpty = stats.isSuccess && total === 0;

  return (
    <div className="space-y-10">
      <section className="relative overflow-hidden rounded-[2rem] border border-white/8 bg-ink-900/55 px-6 py-9 sm:px-10 sm:py-12">
        <div className="pointer-events-none absolute -right-28 -top-40 h-[28rem] w-[28rem] rounded-full bg-brass/10 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-52 left-1/3 h-96 w-96 rounded-full bg-rust/8 blur-3xl" />
        <div className="relative max-w-3xl">
          <p className="eyebrow">A library that feels like yours</p>
          <h1 className="page-title mt-3">
            Your stories,
            <br />
            <span className="text-brass-soft">kept close.</span>
          </h1>
          <p className="mt-5 max-w-xl text-[0.98rem] leading-relaxed text-paper-muted">
            Oryn quietly indexes the visual novels on your shelf, remembers what you loved, and
            launches the next chapter from the machine that can actually play it.
          </p>
          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link
              to="/library"
              className="inline-flex h-11 items-center gap-2 rounded-full border border-brass/35 bg-brass px-5 text-sm font-medium text-ink-950 transition hover:border-brass-soft hover:bg-brass-soft"
            >
              Browse library
              <ArrowRight size={16} />
            </Link>
            <Link
              to="/settings"
              className="inline-flex h-11 items-center gap-2 rounded-full border border-white/12 bg-white/[0.04] px-5 text-sm font-medium text-paper transition hover:border-brass/30 hover:bg-white/[0.07]"
            >
              Library settings
            </Link>
          </div>
        </div>

        <div className="relative mt-10 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Stat label="Games" value={stats.data?.total ?? 0} icon={LibraryBig} />
          <Stat label="Favorites" value={stats.data?.favorites ?? 0} icon={Heart} />
          <Stat label="Missing" value={stats.data?.missing ?? 0} icon={ScanLine} />
          <Stat
            label="Last scan"
            value={library?.lastScanAt ? formatDate(library.lastScanAt) : "Never"}
            icon={Clock3}
            text
          />
        </div>
      </section>

      {stats.isError ? (
        <div className="surface border-rust/25 px-5 py-4 text-sm text-rust-soft" role="alert">
          Oryn cannot reach the server. Check that it is running, then refresh this page.
        </div>
      ) : null}

      {isEmpty ? (
        <EmptyState
          icon={LibraryBig}
          title="Your shelf is waiting"
          description="Run the first scan to discover Ren'Py games in the configured library folder. Oryn only reads game files; it never changes them."
          action={
            <Link
              to="/settings"
              className="inline-flex h-11 items-center gap-2 rounded-full border border-brass/35 bg-brass px-5 text-sm font-medium text-ink-950 transition hover:bg-brass-soft"
            >
              <ScanLine size={16} />
              Scan the library
            </Link>
          }
        />
      ) : null}

      {!isEmpty ? (
        <>
          <ScanStatus />

          <section>
            <SectionHeading
              eyebrow="Fresh on the shelf"
              title="Recently added"
              action={
                <Link to="/library?sort=added&order=desc" className="link-quiet text-sm">
                  View all
                </Link>
              }
            />
            {recent.isLoading ? (
              <GameGridSkeleton count={4} />
            ) : recent.isError ? (
              <div className="surface px-5 py-4 text-sm text-rust-soft">Could not load games.</div>
            ) : (
              <GameGrid games={recent.data?.items ?? []} />
            )}
          </section>

          <section>
            <SectionHeading
              eyebrow="Keep coming back"
              title="Favorites"
              action={
                <Link to="/library?favorite=true" className="link-quiet text-sm">
                  View favorites
                </Link>
              }
            />
            {favorites.isLoading ? (
              <GameGridSkeleton count={4} />
            ) : favorites.data && favorites.data.items.length > 0 ? (
              <GameGrid games={favorites.data.items} />
            ) : (
              <div className="surface flex items-center gap-5 px-5 py-5">
                <div className="hidden w-20 shrink-0 sm:block">
                  <GamePoster title="Your next favorite" className="aspect-[2/3] rounded-xl" />
                </div>
                <div>
                  <p className="font-display text-xl text-paper">Nothing pinned yet</p>
                  <p className="mt-1 max-w-md text-sm text-paper-muted">
                    Tap the heart on any game to keep it close. Favorites are shared with every
                    Oryn client.
                  </p>
                  <Link to="/library" className="link-quiet mt-3 inline-flex text-sm">
                    Find something to love
                  </Link>
                </div>
              </div>
            )}
          </section>
        </>
      ) : null}
    </div>
  );
}

function SectionHeading({
  eyebrow,
  title,
  action,
}: {
  eyebrow: string;
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="mb-5 flex items-end justify-between gap-4">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h2 className="mt-1.5 font-display text-3xl text-paper">{title}</h2>
      </div>
      {action}
    </div>
  );
}

function Stat({
  label,
  value,
  icon: Icon,
  text = false,
}: {
  label: string;
  value: number | string;
  icon: typeof LibraryBig;
  text?: boolean;
}) {
  return (
    <div className="rounded-2xl border border-white/8 bg-ink-950/35 px-4 py-3.5">
      <div className="flex items-center gap-2 text-paper-faint">
        <Icon size={14} strokeWidth={1.8} />
        <span className="text-[0.67rem] font-medium uppercase tracking-[0.16em]">{label}</span>
      </div>
      <p className={`mt-2 text-paper ${text ? "text-sm" : "text-2xl font-semibold"}`}>
        {text ? value : formatNumber(Number(value))}
      </p>
    </div>
  );
}
