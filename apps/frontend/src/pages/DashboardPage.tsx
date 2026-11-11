import { ArrowRight, Clock3, Heart, LibraryBig, Timer } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { GameGrid, GameGridSkeleton } from "../components/GameGrid";
import { GamePoster } from "../components/GamePoster";
import { ScanStatus } from "../components/ScanStatus";
import { EmptyState } from "../components/ui/EmptyState";
import { useGames, useLibraries, useStats } from "../lib/api/queries";
import { formatDate, formatNumber } from "../lib/format";
import { formatDuration } from "../lib/duration";

export function DashboardPage() {
  const stats = useStats();
  const libraries = useLibraries();
  const recent = useGames({ sort: "added", order: "desc", pageSize: 6 });
  const favorites = useGames({ favorite: true, sort: "title", order: "asc", pageSize: 6 });
  const recentlyPlayed = useGames({ played: true, sort: "lastPlayed", order: "desc", pageSize: 6 });

  const library = libraries.data?.items[0];
  const total = stats.data?.total ?? 0;
  const isEmpty = stats.isSuccess && total === 0;

  return (
    <div className="space-y-10">
      <section className="bg-ink-900/55 relative overflow-hidden rounded-[2rem] border border-white/8 px-6 py-9 sm:px-10 sm:py-12">
        <div className="bg-brass/10 pointer-events-none absolute -top-40 -right-28 h-[28rem] w-[28rem] rounded-full blur-3xl" />
        <div className="bg-rust/8 pointer-events-none absolute -bottom-52 left-1/3 h-96 w-96 rounded-full blur-3xl" />
        <div className="relative max-w-3xl">
          <p className="eyebrow">A library that feels like yours</p>
          <h1 className="page-title mt-3">
            Your stories,
            <br />
            <span className="text-brass-soft">kept close.</span>
          </h1>
          <p className="text-paper-muted mt-5 max-w-xl text-[0.98rem] leading-relaxed">
            Oryn quietly indexes the visual novels on your shelf, remembers what you loved, and
            launches the next chapter from the machine that can actually play it.
          </p>
          <div className="mt-7 flex flex-wrap items-center gap-3">
            <Link
              to="/library"
              className="border-brass/35 bg-brass text-ink-950 hover:border-brass-soft hover:bg-brass-soft inline-flex h-11 items-center gap-2 rounded-full border px-5 text-sm font-medium transition"
            >
              Browse library
              <ArrowRight size={16} />
            </Link>
            <Link
              to="/settings"
              className="text-paper hover:border-brass/30 inline-flex h-11 items-center gap-2 rounded-full border border-white/12 bg-white/[0.04] px-5 text-sm font-medium transition hover:bg-white/[0.07]"
            >
              Library settings
            </Link>
          </div>
        </div>

        <div className="relative mt-10 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Stat label="Games" value={stats.data?.total ?? 0} icon={LibraryBig} />
          <Stat label="Favorites" value={stats.data?.favorites ?? 0} icon={Heart} />
          <Stat
            label="Playtime"
            value={formatDuration(stats.data?.playtimeSeconds ?? 0)}
            icon={Timer}
            text
          />
          <Stat
            label="Last scan"
            value={library?.lastScanAt ? formatDate(library.lastScanAt) : "Never"}
            icon={Clock3}
            text
          />
        </div>
      </section>

      {stats.isError ? (
        <div className="surface border-rust/25 text-rust-soft px-5 py-4 text-sm" role="alert">
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
              className="border-brass/35 bg-brass text-ink-950 hover:bg-brass-soft inline-flex h-11 items-center gap-2 rounded-full border px-5 text-sm font-medium transition"
            >
              <Timer size={16} />
              Scan the library
            </Link>
          }
        />
      ) : null}

      {!isEmpty ? (
        <>
          <ScanStatus />

          {recentlyPlayed.data && recentlyPlayed.data.items.length > 0 ? (
            <section>
              <SectionHeading
                eyebrow="Pick up where you left off"
                title="Recently played"
                action={
                  <Link to="/library?sort=lastPlayed&order=desc" className="link-quiet text-sm">
                    View library
                  </Link>
                }
              />
              <GameGrid games={recentlyPlayed.data.items} />
            </section>
          ) : null}

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
              <div className="surface text-rust-soft px-5 py-4 text-sm">Could not load games.</div>
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
                  <p className="font-display text-paper text-xl">Nothing pinned yet</p>
                  <p className="text-paper-muted mt-1 max-w-md text-sm">
                    Tap the heart on any game to keep it close. Favorites are shared with every Oryn
                    client.
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
        <h2 className="font-display text-paper mt-1.5 text-3xl">{title}</h2>
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
    <div className="bg-ink-950/35 rounded-2xl border border-white/8 px-4 py-3.5">
      <div className="text-paper-faint flex items-center gap-2">
        <Icon size={14} strokeWidth={1.8} />
        <span className="text-[0.67rem] font-medium tracking-[0.16em] uppercase">{label}</span>
      </div>
      <p className={`text-paper mt-2 ${text ? "text-sm" : "text-2xl font-semibold"}`}>
        {text ? value : formatNumber(Number(value))}
      </p>
    </div>
  );
}
