import {
  ArrowLeft,
  CalendarDays,
  CheckCircle2,
  FolderOpen,
  Heart,
  Pencil,
  TriangleAlert,
} from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArtworkControls } from "../components/ArtworkControls";
import { GamePoster } from "../components/GamePoster";
import { MetadataDialog } from "../components/MetadataDialog";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { Skeleton } from "../components/ui/Skeleton";
import { artworkUrl } from "../lib/api/artwork";
import { useGame, useSetFavorite } from "../lib/api/queries";
import { formatDate } from "../lib/format";

const completionLabels: Record<string, string> = {
  unplayed: "Unplayed",
  playing: "Playing",
  completed: "Completed",
  on_hold: "On hold",
  dropped: "Dropped",
};

export function GameDetailPage() {
  const { gameId = "" } = useParams();
  const game = useGame(gameId);
  const favorite = useSetFavorite();
  const [editing, setEditing] = useState(false);
  const [coverFailed, setCoverFailed] = useState(false);
  const [bannerFailed, setBannerFailed] = useState(false);

  if (game.isLoading) {
    return <DetailSkeleton />;
  }

  if (game.isError) {
    return (
      <EmptyState
        icon={TriangleAlert}
        title="Game not found"
        description="This game may have been removed from the library, or the server is unavailable."
        action={
          <Link to="/library" className="link-quiet text-sm">
            Back to the library
          </Link>
        }
      />
    );
  }

  if (!game.data) return null;
  const entry = game.data;
  const isMissing = entry.availability === "missing";
  const cover = artworkUrl(entry.coverArtworkId, "detail");
  const banner =
    artworkUrl(entry.bannerArtworkId, "detail") ?? artworkUrl(entry.coverArtworkId, "detail");

  return (
    <div className="space-y-8">
      <Link
        to="/library"
        className="text-paper-muted hover:text-paper inline-flex items-center gap-2 text-sm transition"
      >
        <ArrowLeft size={16} />
        Back to library
      </Link>

      {isMissing ? (
        <div className="border-rust/25 bg-rust/8 text-rust-soft flex items-start gap-3 rounded-2xl border px-4 py-3.5 text-sm">
          <TriangleAlert size={18} className="mt-0.5 shrink-0" />
          <p>
            Oryn could not find this folder during the last clean scan. The record and your metadata
            are preserved; reconnect the share and scan again, or edit the details if the game
            moved.
          </p>
        </div>
      ) : null}

      <section className="surface overflow-hidden">
        <div className="relative">
          <div className="absolute inset-0 opacity-60">
            <GamePoster title={entry.title} className="h-full w-full" showText={false} />
            {banner && !bannerFailed ? (
              <img
                src={banner}
                alt=""
                onError={() => setBannerFailed(true)}
                className="absolute inset-0 h-full w-full object-cover"
              />
            ) : null}
          </div>
          <div className="from-ink-950 via-ink-950/80 to-ink-950/30 absolute inset-0 bg-gradient-to-t" />
          <div className="relative grid gap-7 px-5 py-7 sm:px-8 sm:py-9 md:grid-cols-[13rem_minmax(0,1fr)] md:gap-9">
            <div className="mx-auto w-40 sm:w-48 md:mx-0 md:w-full">
              <div className="relative aspect-[2/3] w-full overflow-hidden rounded-2xl border border-white/10 shadow-[0_30px_70px_-25px_rgba(0,0,0,0.9)]">
                <GamePoster title={entry.title} className="h-full w-full" />
                {cover && !coverFailed ? (
                  <img
                    src={cover}
                    alt={`Cover artwork for ${entry.title}`}
                    onError={() => setCoverFailed(true)}
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                ) : null}
              </div>
            </div>

            <div className="flex min-w-0 flex-col justify-end">
              <div className="flex flex-wrap items-center gap-2">
                <Badge tone="brass">{completionLabels[entry.completionStatus] ?? "Unplayed"}</Badge>
                {entry.rating ? <Badge tone="quiet">★ {entry.rating} / 10</Badge> : null}
                {entry.version ? <Badge tone="quiet">Version {entry.version}</Badge> : null}
                {isMissing ? <Badge tone="danger">Missing</Badge> : null}
              </div>
              <h1 className="font-display text-paper mt-4 text-4xl leading-[1.02] tracking-[-0.02em] sm:text-5xl">
                {entry.title}
              </h1>
              <p className="text-paper-muted mt-2">
                {[entry.developer, entry.publisher].filter(Boolean).join(" · ") ||
                  "Developer not detected"}
              </p>

              <div className="mt-6 flex flex-wrap items-center gap-2.5">
                <Button
                  variant={entry.favorite ? "primary" : "secondary"}
                  onClick={() => favorite.mutate({ id: entry.id, favorite: !entry.favorite })}
                  disabled={favorite.isPending}
                >
                  <Heart
                    size={16}
                    fill={entry.favorite ? "currentColor" : "none"}
                    className={entry.favorite ? "text-ink-950" : "text-brass"}
                  />
                  {entry.favorite ? "In favorites" : "Add to favorites"}
                </Button>
                <Button variant="secondary" onClick={() => setEditing(true)}>
                  <Pencil size={15} />
                  Edit details
                </Button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1.7fr)_minmax(18rem,1fr)]">
        <div className="space-y-6">
          <section className="surface px-6 py-6">
            <h2 className="font-display text-paper text-2xl">About this story</h2>
            <p className="text-paper-muted mt-4 text-[0.96rem] leading-7 whitespace-pre-line">
              {entry.description || "No description has been detected or written yet."}
            </p>
          </section>

          {entry.notes ? (
            <section className="surface px-6 py-6">
              <h2 className="font-display text-paper text-2xl">Your notes</h2>
              <p className="text-paper-muted mt-4 text-[0.96rem] leading-7 whitespace-pre-line">
                {entry.notes}
              </p>
            </section>
          ) : null}

          <section className="surface px-6 py-6">
            <h2 className="font-display text-paper text-2xl">Play</h2>
            <div className="mt-4 flex items-start gap-4 rounded-2xl border border-white/8 bg-white/[0.025] p-4">
              <div className="border-brass/20 bg-brass/8 text-brass grid h-10 w-10 shrink-0 place-items-center rounded-full border">
                <CheckCircle2 size={18} />
              </div>
              <div>
                <p className="text-paper text-sm font-medium">
                  Launching arrives with the desktop app
                </p>
                <p className="text-paper-muted mt-1 text-sm leading-relaxed">
                  The browser can browse and organize the shelf, but games run on your computer. The
                  Wails desktop client will add path mapping, local launch, and playtime tracking in
                  a later milestone.
                </p>
              </div>
            </div>
          </section>
        </div>

        <aside className="space-y-0">
          <div className="surface h-fit px-6 py-6">
            <h2 className="font-display text-paper text-2xl">Details</h2>
            <dl className="mt-5 space-y-4 text-sm">
              <Detail icon={FolderOpen} label="Library path" value={entry.relativePath} mono />
              <Detail icon={CalendarDays} label="Added" value={formatDate(entry.addedAt)} />
              <Detail label="Last seen" value={formatDate(entry.lastSeenAt)} />
              <Detail label="Developer" value={entry.developer || "Not detected"} />
              <Detail label="Publisher" value={entry.publisher || "Not detected"} />
              <Detail label="Release" value={entry.releaseDate || "Not detected"} />
              <Detail label="Availability" value={isMissing ? "Missing" : "Available"} />
            </dl>
          </div>
          <ArtworkControls game={entry} />
        </aside>
      </div>

      <MetadataDialog game={entry} open={editing} onClose={() => setEditing(false)} />
    </div>
  );
}

function Detail({
  icon: Icon,
  label,
  value,
  mono = false,
}: {
  icon?: typeof FolderOpen;
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div>
      <dt className="text-paper-faint flex items-center gap-2 text-[0.68rem] font-medium tracking-[0.14em] uppercase">
        {Icon ? <Icon size={13} /> : null}
        {label}
      </dt>
      <dd className={`text-paper mt-1 break-words ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-8">
      <Skeleton className="h-5 w-32" />
      <div className="surface grid gap-8 p-8 md:grid-cols-[13rem_minmax(0,1fr)]">
        <Skeleton className="aspect-[2/3] w-full" />
        <div className="flex flex-col justify-end gap-3">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-12 w-2/3" />
          <Skeleton className="h-4 w-1/3" />
        </div>
      </div>
      <div className="grid gap-6 lg:grid-cols-[1.7fr_1fr]">
        <Skeleton className="h-64" />
        <Skeleton className="h-64" />
      </div>
    </div>
  );
}
