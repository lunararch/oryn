import { FolderHeart, Plus } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { Skeleton } from "../components/ui/Skeleton";
import { useCollections } from "../lib/api/queries";
import { CollectionDialog } from "../components/CollectionDialog";

export function CollectionsPage() {
  const collections = useCollections();
  const [creating, setCreating] = useState(false);

  return (
    <div className="space-y-7">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">Your own shelves</p>
          <h1 className="page-title mt-2">Collections</h1>
          <p className="text-paper-muted mt-3 max-w-xl text-sm leading-relaxed">
            Group games by mood, route, or anything else. A game can live on several shelves at
            once.
          </p>
        </div>
        <Button variant="primary" onClick={() => setCreating(true)}>
          <Plus size={16} />
          New collection
        </Button>
      </div>

      {collections.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-40" />
          ))}
        </div>
      ) : collections.isError ? (
        <div className="surface border-rust/25 text-rust-soft px-5 py-4 text-sm" role="alert">
          Could not load collections.
        </div>
      ) : collections.data && collections.data.items.length === 0 ? (
        <EmptyState
          icon={FolderHeart}
          title="No collections yet"
          description="Make a shelf for rainy-day stories, favorite routes, or the games you keep meaning to start."
          action={<Button onClick={() => setCreating(true)}>Create the first collection</Button>}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {collections.data?.items.map((collection) => (
            <Link
              key={collection.id}
              to={`/collections/${collection.id}`}
              className="surface group hover:border-brass/25 hover:bg-ink-850/85 flex flex-col justify-between px-5 py-5 transition"
            >
              <div>
                <p className="text-brass/80 text-[0.67rem] font-semibold tracking-[0.15em] uppercase">
                  {collection.gameCount} {collection.gameCount === 1 ? "game" : "games"}
                </p>
                <h2 className="font-display text-paper group-hover:text-brass-soft mt-2 text-2xl transition-colors">
                  {collection.name}
                </h2>
                <p className="text-paper-muted mt-2 line-clamp-3 text-sm leading-relaxed">
                  {collection.description || "No description yet."}
                </p>
              </div>
              <span className="text-paper-faint mt-5 text-xs">Open shelf →</span>
            </Link>
          ))}
        </div>
      )}

      <CollectionDialog open={creating} onClose={() => setCreating(false)} />
    </div>
  );
}
