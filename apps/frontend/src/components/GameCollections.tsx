import { FolderHeart } from "lucide-react";
import { Link } from "react-router-dom";
import { useGameCollections, useToggleCollectionMembership } from "../lib/api/queries";

export function GameCollections({ gameID }: { gameID: string }) {
  const collections = useGameCollections(gameID);
  const toggle = useToggleCollectionMembership();

  return (
    <section className="surface px-6 py-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="font-display text-paper text-2xl">Collections</h2>
          <p className="text-paper-muted mt-1 text-sm">Put this game on one or more shelves.</p>
        </div>
        <Link to="/collections" className="link-quiet text-xs">
          Manage
        </Link>
      </div>

      {collections.isLoading ? (
        <p className="text-paper-muted mt-4 text-sm">Loading collections…</p>
      ) : collections.isError ? (
        <p className="text-rust-soft mt-4 text-sm">Could not load collections.</p>
      ) : collections.data && collections.data.items.length === 0 ? (
        <div className="text-paper-muted mt-4 flex items-center gap-3 rounded-2xl border border-white/8 bg-white/[0.02] px-4 py-3 text-sm">
          <FolderHeart size={17} className="text-paper-faint shrink-0" />
          <span>
            No collections yet.{" "}
            <Link to="/collections" className="link-quiet">
              Create one
            </Link>
            .
          </span>
        </div>
      ) : (
        <div className="mt-4 flex flex-wrap gap-2">
          {collections.data?.items.map((collection) => (
            <button
              key={collection.id}
              type="button"
              aria-pressed={collection.contains}
              disabled={toggle.isPending}
              onClick={() =>
                toggle.mutate({
                  collectionID: collection.id,
                  gameID,
                  contains: collection.contains,
                })
              }
              className={`rounded-full border px-3 py-1.5 text-xs font-medium transition disabled:opacity-60 ${
                collection.contains
                  ? "border-brass/35 bg-brass/12 text-brass-soft"
                  : "text-paper-muted hover:border-brass/25 hover:text-paper border-white/10"
              }`}
            >
              {collection.name}
              <span className="ml-1.5 text-[0.62rem] opacity-60">· {collection.gameCount}</span>
            </button>
          ))}
        </div>
      )}

      {toggle.isError ? (
        <p className="text-rust-soft mt-3 text-xs" role="alert">
          {toggle.error.message}
        </p>
      ) : null}
    </section>
  );
}
