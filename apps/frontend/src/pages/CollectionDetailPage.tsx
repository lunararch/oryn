import { ArrowLeft, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { CollectionDialog } from "../components/CollectionDialog";
import { GameGrid, GameGridSkeleton } from "../components/GameGrid";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { useCollection, useDeleteCollection, useGames } from "../lib/api/queries";

export function CollectionDetailPage() {
  const { collectionId = "" } = useParams();
  const navigate = useNavigate();
  const collection = useCollection(collectionId);
  const games = useGames({ collection: collectionId, pageSize: 100, sort: "title", order: "asc" });
  const remove = useDeleteCollection();
  const [editing, setEditing] = useState(false);

  if (collection.isError) {
    return (
      <EmptyState
        icon={Pencil}
        title="Collection not found"
        description="This shelf may have been deleted."
        action={
          <Link to="/collections" className="link-quiet text-sm">
            Back to collections
          </Link>
        }
      />
    );
  }

  return (
    <div className="space-y-7">
      <Link
        to="/collections"
        className="text-paper-muted hover:text-paper inline-flex items-center gap-2 text-sm transition"
      >
        <ArrowLeft size={16} />
        Back to collections
      </Link>

      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="eyebrow">Collection</p>
          <h1 className="page-title mt-2">{collection.data?.name ?? "Loading…"}</h1>
          {collection.data?.description ? (
            <p className="text-paper-muted mt-3 max-w-2xl text-sm leading-relaxed">
              {collection.data.description}
            </p>
          ) : null}
          <p className="text-paper-faint mt-2 text-xs">
            {collection.data?.gameCount ?? 0} {collection.data?.gameCount === 1 ? "game" : "games"}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setEditing(true)}>
            <Pencil size={15} />
            Edit
          </Button>
          <Button
            variant="danger"
            disabled={remove.isPending}
            onClick={() => {
              if (
                !window.confirm("Delete this collection? The games themselves are not affected.")
              ) {
                return;
              }
              remove.mutate(collectionId, {
                onSuccess: () => navigate("/collections"),
              });
            }}
          >
            <Trash2 size={15} />
            Delete
          </Button>
        </div>
      </div>

      {collection.isLoading || games.isLoading ? (
        <GameGridSkeleton count={6} />
      ) : games.data && games.data.items.length > 0 ? (
        <GameGrid games={games.data.items} />
      ) : (
        <EmptyState
          icon={Pencil}
          title="This shelf is empty"
          description="Open a game's details page and add it to this collection."
          action={
            <Link to="/library" className="link-quiet text-sm">
              Browse the library
            </Link>
          }
        />
      )}

      {collection.data ? (
        <CollectionDialog
          open={editing}
          onClose={() => setEditing(false)}
          collection={{
            id: collection.data.id,
            name: collection.data.name,
            description: collection.data.description,
          }}
        />
      ) : null}
    </div>
  );
}
