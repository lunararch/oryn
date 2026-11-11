import { Heart } from "lucide-react";
import { Link } from "react-router-dom";
import type { Game } from "../lib/api/generated";
import { Badge } from "./ui/Badge";
import { GamePoster } from "./GamePoster";

export type GameView = "grid" | "list";

interface GameCardProps {
  game: Game;
  view: GameView;
  onToggleFavorite: (game: Game, favorite: boolean) => void;
  favoritePending?: boolean;
}

const completionLabels: Record<string, string> = {
  playing: "Playing",
  completed: "Completed",
  dropped: "Dropped",
  on_hold: "On hold",
};

export function GameCard({ game, view, onToggleFavorite, favoritePending }: GameCardProps) {
  const isMissing = game.availability === "missing";
  const completion = completionLabels[game.completionStatus];

  return (
    <article className={`group relative ${view === "list" ? "game-card-list" : "game-card-grid"}`}>
      <Link
        to={`/games/${game.id}`}
        className={`block rounded-[1.35rem] outline-none focus-visible:ring-2 focus-visible:ring-brass focus-visible:ring-offset-4 focus-visible:ring-offset-ink-950 ${
          view === "list"
            ? "flex items-stretch gap-4 p-3"
            : "flex h-full flex-col overflow-hidden"
        }`}
      >
        <div
          className={
            view === "list"
              ? "w-24 shrink-0 overflow-hidden rounded-xl sm:w-28"
              : "relative aspect-[2/3] w-full overflow-hidden"
          }
        >
          <GamePoster title={game.title} className="h-full w-full" />
          {isMissing ? (
            <span className="absolute left-3 top-3 rounded-full bg-ink-950/85 px-2.5 py-1 text-[0.65rem] font-semibold uppercase tracking-[0.16em] text-rust-soft backdrop-blur">
              Missing
            </span>
          ) : null}
        </div>

        <div
          className={
            view === "list"
              ? "flex min-w-0 flex-1 flex-col justify-center py-1 pr-10"
              : "flex flex-1 flex-col gap-2 px-4 pb-4 pt-4"
          }
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h3 className="font-display text-[1.15rem] leading-tight text-paper transition-colors group-hover:text-brass-soft">
                {game.title}
              </h3>
              {game.developer ? (
                <p className="mt-1 truncate text-sm text-paper-muted">{game.developer}</p>
              ) : null}
            </div>
          </div>

          <div className="mt-auto flex flex-wrap items-center gap-1.5 pt-3">
            {completion ? <Badge tone="brass">{completion}</Badge> : null}
            {game.rating ? <Badge tone="quiet">★ {game.rating}</Badge> : null}
            {game.version ? <Badge tone="quiet">v{game.version}</Badge> : null}
          </div>
        </div>
      </Link>

      <button
        type="button"
        aria-label={game.favorite ? "Remove from favorites" : "Add to favorites"}
        aria-pressed={game.favorite}
        disabled={favoritePending}
        onClick={() => onToggleFavorite(game, !game.favorite)}
        className={`absolute right-3 top-3 grid h-9 w-9 place-items-center rounded-full border border-white/10 bg-ink-950/65 backdrop-blur transition hover:border-brass/50 hover:bg-ink-900 disabled:opacity-60 ${
          game.favorite ? "text-brass" : "text-paper-muted hover:text-paper"
        }`}
      >
        <Heart size={17} strokeWidth={1.8} fill={game.favorite ? "currentColor" : "none"} />
      </button>
    </article>
  );
}
