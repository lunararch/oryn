import { useState, type FormEvent, type ReactNode } from "react";
import type { Game, UpdateGameRequest } from "../lib/api/generated";
import { useUpdateGame } from "../lib/api/queries";
import { Button } from "./ui/Button";
import { Dialog } from "./ui/Dialog";
import { Input } from "./ui/Input";
import { Select } from "./ui/Select";

interface MetadataDialogProps {
  game: Game;
  open: boolean;
  onClose: () => void;
}

const completionOptions = [
  { value: "unplayed", label: "Unplayed" },
  { value: "playing", label: "Playing" },
  { value: "completed", label: "Completed" },
  { value: "on_hold", label: "On hold" },
  { value: "dropped", label: "Dropped" },
];

export function MetadataDialog({ game, open, onClose }: MetadataDialogProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Edit game details"
      description="Your edits are stored separately and survive future scans."
    >
      {open ? <MetadataForm key={game.id} game={game} onClose={onClose} /> : null}
    </Dialog>
  );
}

function MetadataForm({ game, onClose }: { game: Game; onClose: () => void }) {
  const update = useUpdateGame();
  const [title, setTitle] = useState(game.title);
  const [developer, setDeveloper] = useState(game.developer);
  const [publisher, setPublisher] = useState(game.publisher);
  const [releaseDate, setReleaseDate] = useState(game.releaseDate);
  const [rating, setRating] = useState(game.rating ? String(game.rating) : "");
  const [completionStatus, setCompletionStatus] = useState(game.completionStatus);
  const [description, setDescription] = useState(game.description);
  const [notes, setNotes] = useState(game.notes);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const body: UpdateGameRequest = {
      title,
      developer,
      publisher,
      releaseDate,
      rating: rating === "" ? undefined : Number(rating),
      completionStatus,
      description,
      notes,
    };
    update.mutate(
      { id: game.id, body },
      {
        onSuccess: () => onClose(),
      },
    );
  }

  return (
    <form onSubmit={submit} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Title">
          <Input value={title} onChange={(event) => setTitle(event.target.value)} maxLength={300} />
        </Field>
        <Field label="Developer">
          <Input
            value={developer}
            onChange={(event) => setDeveloper(event.target.value)}
            maxLength={200}
          />
        </Field>
        <Field label="Publisher">
          <Input
            value={publisher}
            onChange={(event) => setPublisher(event.target.value)}
            maxLength={200}
          />
        </Field>
        <Field label="Release date">
          <Input
            value={releaseDate}
            onChange={(event) => setReleaseDate(event.target.value)}
            maxLength={50}
            placeholder="2025 or 2025-04-01"
          />
        </Field>
        <Field label="Personal rating">
          <Select value={rating} onChange={(event) => setRating(event.target.value)}>
            <option value="">Not rated</option>
            {Array.from({ length: 10 }, (_, index) => 10 - index).map((value) => (
              <option key={value} value={value}>
                {value} / 10
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Completion">
          <Select
            value={completionStatus}
            onChange={(event) => setCompletionStatus(event.target.value)}
          >
            {completionOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      <Field label="Description">
        <textarea
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          maxLength={20000}
          rows={5}
          className="field resize-y"
          placeholder="What is this story about?"
        />
      </Field>

      <Field label="Private notes">
        <textarea
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          maxLength={10000}
          rows={4}
          className="field resize-y"
          placeholder="Routes, saves, or anything you want to remember."
        />
      </Field>

      {update.isError ? (
        <p
          className="rounded-xl border border-rust/25 bg-rust/10 px-3.5 py-2.5 text-sm text-rust-soft"
          role="alert"
        >
          {update.error.message}
        </p>
      ) : null}

      <div className="flex justify-end gap-2 border-t border-white/8 pt-4">
        <Button type="button" variant="ghost" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={update.isPending}>
          {update.isPending ? "Saving…" : "Save details"}
        </Button>
      </div>
    </form>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-medium uppercase tracking-[0.13em] text-paper-muted">
        {label}
      </span>
      {children}
    </label>
  );
}
