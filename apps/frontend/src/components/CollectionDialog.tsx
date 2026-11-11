import { useState, type FormEvent } from "react";
import { useCreateCollection, useUpdateCollection } from "../lib/api/queries";
import { Button } from "./ui/Button";
import { Dialog } from "./ui/Dialog";
import { Input } from "./ui/Input";

interface CollectionDialogProps {
  open: boolean;
  onClose: () => void;
  collection?: { id: string; name: string; description: string };
}

export function CollectionDialog({ open, onClose, collection }: CollectionDialogProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={collection ? "Edit collection" : "New collection"}
      description="Collections are stored in Oryn and never move game folders."
    >
      {open ? <CollectionForm collection={collection} onClose={onClose} /> : null}
    </Dialog>
  );
}

function CollectionForm({
  collection,
  onClose,
}: {
  collection?: { id: string; name: string; description: string };
  onClose: () => void;
}) {
  const create = useCreateCollection();
  const update = useUpdateCollection();
  const [name, setName] = useState(collection?.name ?? "");
  const [description, setDescription] = useState(collection?.description ?? "");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (collection) {
      update.mutate({ id: collection.id, name, description }, { onSuccess: () => onClose() });
    } else {
      create.mutate({ name, description }, { onSuccess: () => onClose() });
    }
  }

  const pending = create.isPending || update.isPending;
  const error = create.error ?? update.error;

  return (
    <form onSubmit={submit} className="space-y-5">
      <label className="block">
        <span className="text-paper-muted mb-1.5 block text-xs font-medium tracking-[0.13em] uppercase">
          Name
        </span>
        <Input
          value={name}
          onChange={(event) => setName(event.target.value)}
          maxLength={100}
          placeholder="Rainy day stories"
          autoFocus
          required
        />
      </label>
      <label className="block">
        <span className="text-paper-muted mb-1.5 block text-xs font-medium tracking-[0.13em] uppercase">
          Description
        </span>
        <textarea
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          maxLength={2000}
          rows={4}
          className="field resize-y"
          placeholder="What belongs on this shelf?"
        />
      </label>

      {error ? (
        <p
          className="border-rust/25 bg-rust/10 text-rust-soft rounded-xl border px-3.5 py-2.5 text-sm"
          role="alert"
        >
          {error.message}
        </p>
      ) : null}

      <div className="flex justify-end gap-2 border-t border-white/8 pt-4">
        <Button type="button" variant="ghost" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={pending || !name.trim()}>
          {pending ? "Saving…" : collection ? "Save collection" : "Create collection"}
        </Button>
      </div>
    </form>
  );
}
