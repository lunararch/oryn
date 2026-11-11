import { useSetFavorite } from "../lib/api/queries";
import type { Game } from "../lib/api/generated";
import { GameCard, type GameView } from "./GameCard";
import { Skeleton } from "./ui/Skeleton";

interface GameGridProps {
  games: Game[];
  view?: GameView;
}

export function GameGrid({ games, view = "grid" }: GameGridProps) {
  const favorite = useSetFavorite();

  return (
    <div
      className={
        view === "grid"
          ? "grid grid-cols-2 gap-4 sm:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5"
          : "flex flex-col gap-3"
      }
    >
      {games.map((game) => (
        <GameCard
          key={game.id}
          game={game}
          view={view}
          favoritePending={favorite.isPending && favorite.variables?.id === game.id}
          onToggleFavorite={(target, next) => favorite.mutate({ id: target.id, favorite: next })}
        />
      ))}
    </div>
  );
}

export function GameGridSkeleton({
  count = 8,
  view = "grid",
}: {
  count?: number;
  view?: GameView;
}) {
  if (view === "list") {
    return (
      <div className="flex flex-col gap-3">
        {Array.from({ length: count }).map((_, index) => (
          <div key={index} className="flex gap-4 rounded-[1.35rem] border border-white/8 p-3">
            <Skeleton className="h-28 w-24 shrink-0" />
            <div className="flex flex-1 flex-col justify-center gap-3">
              <Skeleton className="h-5 w-1/3" />
              <Skeleton className="h-3 w-1/4" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
      {Array.from({ length: count }).map((_, index) => (
        <div key={index} className="overflow-hidden rounded-[1.35rem] border border-white/8">
          <Skeleton className="aspect-[2/3] w-full rounded-none" />
          <div className="space-y-2 p-4">
            <Skeleton className="h-5 w-4/5" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        </div>
      ))}
    </div>
  );
}
