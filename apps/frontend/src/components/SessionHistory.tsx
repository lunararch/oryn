import { Clock3, Pencil, Plus, Trash2 } from "lucide-react";
import type { PlaySession } from "../lib/api/generated";
import { useDeleteSession, useSessions } from "../lib/api/queries";
import { formatDate } from "../lib/format";
import { formatDuration } from "../lib/duration";
import { Button } from "./ui/Button";

interface SessionHistoryProps {
  gameID: string;
  onCreate: () => void;
  onEdit: (session: PlaySession) => void;
}

export function SessionHistory({ gameID, onCreate, onEdit }: SessionHistoryProps) {
  const sessions = useSessions(gameID);
  const remove = useDeleteSession();

  return (
    <section className="surface px-6 py-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="font-display text-paper text-2xl">Play history</h2>
          <p className="text-paper-muted mt-1 text-sm">
            Manual sessions contribute to playtime and recently played.
          </p>
        </div>
        <Button size="sm" variant="secondary" onClick={onCreate}>
          <Plus size={14} />
          Log session
        </Button>
      </div>

      {sessions.isLoading ? (
        <p className="text-paper-muted mt-5 text-sm">Loading play history…</p>
      ) : sessions.isError ? (
        <p className="text-rust-soft mt-5 text-sm">Could not load play history.</p>
      ) : sessions.data && sessions.data.items.length === 0 ? (
        <div className="mt-5 flex items-center gap-3 rounded-2xl border border-white/8 bg-white/[0.02] px-4 py-4">
          <Clock3 size={18} className="text-paper-faint shrink-0" />
          <p className="text-paper-muted text-sm">
            No sessions recorded yet. Log one manually, or use the desktop launcher when it arrives.
          </p>
        </div>
      ) : (
        <ul className="mt-5 divide-y divide-white/7">
          {sessions.data?.items.map((session) => (
            <li key={session.id} className="flex flex-wrap items-center gap-3 py-3.5">
              <div className="min-w-0 flex-1">
                <p className="text-paper text-sm">
                  {formatDate(session.startedAt)}
                  {session.endedAt
                    ? ` · ended ${new Date(session.endedAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}`
                    : ""}
                </p>
                <p className="text-paper-faint mt-0.5 text-xs">
                  {formatDuration(session.durationSeconds)} · {session.source}
                  {session.notes ? ` · ${session.notes}` : ""}
                </p>
              </div>
              {session.source === "manual" ? (
                <div className="flex items-center gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    aria-label="Edit session"
                    onClick={() => onEdit(session)}
                  >
                    <Pencil size={14} />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    aria-label="Delete session"
                    disabled={remove.isPending}
                    onClick={() => remove.mutate({ sessionID: session.id, gameID })}
                  >
                    <Trash2 size={14} />
                  </Button>
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      {remove.isError ? (
        <p className="text-rust-soft mt-3 text-xs" role="alert">
          {remove.error.message}
        </p>
      ) : null}
    </section>
  );
}
