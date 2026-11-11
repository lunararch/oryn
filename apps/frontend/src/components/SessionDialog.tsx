import { useMemo, useState, type FormEvent } from "react";
import type { PlaySession } from "../lib/api/generated";
import { useCreateSession, useUpdateSession } from "../lib/api/queries";
import { formatDuration } from "../lib/duration";
import { Button } from "./ui/Button";
import { Dialog } from "./ui/Dialog";
import { Input } from "./ui/Input";

interface SessionDialogProps {
  gameID: string;
  open: boolean;
  session?: PlaySession;
  onClose: () => void;
}

function localInput(date: Date): string {
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

export function SessionDialog({ gameID, open, session, onClose }: SessionDialogProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={session ? "Edit play session" : "Log a play session"}
      description="Manual sessions are useful when you played outside the desktop launcher."
    >
      {open ? <SessionForm gameID={gameID} session={session} onClose={onClose} /> : null}
    </Dialog>
  );
}

function SessionForm({
  gameID,
  session,
  onClose,
}: {
  gameID: string;
  session?: PlaySession;
  onClose: () => void;
}) {
  const create = useCreateSession();
  const update = useUpdateSession();
  const [startedAt, setStartedAt] = useState(() =>
    localInput(session ? new Date(session.startedAt) : new Date(Date.now() - 60 * 60_000)),
  );
  const [endedAt, setEndedAt] = useState(() =>
    localInput(session?.endedAt ? new Date(session.endedAt) : new Date()),
  );
  const [notes, setNotes] = useState(session?.notes ?? "");

  const duration = useMemo(() => {
    const start = new Date(startedAt).getTime();
    const end = new Date(endedAt).getTime();
    if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return 0;
    return Math.floor((end - start) / 1000);
  }, [endedAt, startedAt]);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input = {
      startedAt: new Date(startedAt).toISOString(),
      endedAt: new Date(endedAt).toISOString(),
      notes,
    };
    if (session) {
      update.mutate({ sessionID: session.id, gameID, input }, { onSuccess: () => onClose() });
    } else {
      create.mutate({ gameID, input }, { onSuccess: () => onClose() });
    }
  }

  const pending = create.isPending || update.isPending;
  const error = create.error ?? update.error;

  return (
    <form onSubmit={submit} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block">
          <span className="text-paper-muted mb-1.5 block text-xs font-medium tracking-[0.13em] uppercase">
            Started
          </span>
          <Input
            type="datetime-local"
            value={startedAt}
            onChange={(event) => setStartedAt(event.target.value)}
            required
          />
        </label>
        <label className="block">
          <span className="text-paper-muted mb-1.5 block text-xs font-medium tracking-[0.13em] uppercase">
            Ended
          </span>
          <Input
            type="datetime-local"
            value={endedAt}
            onChange={(event) => setEndedAt(event.target.value)}
            required
          />
        </label>
      </div>

      <div className="border-brass/15 bg-brass/[0.055] text-brass-soft rounded-xl border px-3.5 py-2.5 text-sm">
        Duration: {duration > 0 ? formatDuration(duration) : "enter a valid time range"}
      </div>

      <label className="block">
        <span className="text-paper-muted mb-1.5 block text-xs font-medium tracking-[0.13em] uppercase">
          Notes
        </span>
        <textarea
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          rows={3}
          maxLength={2000}
          className="field resize-y"
          placeholder="Route, chapter, or anything worth remembering."
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
        <Button type="submit" variant="primary" disabled={pending || duration <= 0}>
          {pending ? "Saving…" : session ? "Save session" : "Log session"}
        </Button>
      </div>
    </form>
  );
}
