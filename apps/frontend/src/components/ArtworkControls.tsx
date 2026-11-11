import { ImagePlus, Trash2 } from "lucide-react";
import { useRef, useState } from "react";
import type { Game } from "../lib/api/generated";
import { useDeleteArtwork, useUploadArtwork } from "../lib/api/queries";
import { Button } from "./ui/Button";

interface ArtworkControlsProps {
  game: Game;
}

const MAX_UPLOAD = 10 * 1024 * 1024;

export function ArtworkControls({ game }: ArtworkControlsProps) {
  const upload = useUploadArtwork();
  const remove = useDeleteArtwork();
  const coverInput = useRef<HTMLInputElement>(null);
  const bannerInput = useRef<HTMLInputElement>(null);
  const [error, setError] = useState("");

  function choose(kind: "cover" | "banner", file: File | undefined) {
    if (!file) return;
    setError("");
    if (file.size > MAX_UPLOAD) {
      setError("Image must be smaller than 10 MB.");
      return;
    }
    upload.mutate({ gameID: game.id, kind, file });
  }

  return (
    <section className="surface mt-6 px-6 py-6">
      <div className="text-brass flex items-center gap-2">
        <ImagePlus size={16} />
        <h2 className="font-display text-paper text-xl">Artwork</h2>
      </div>
      <p className="text-paper-muted mt-2 text-xs leading-relaxed">
        Uploaded images are stored in Oryn's data directory. Original game files are never changed.
      </p>

      <div className="mt-4 space-y-3">
        <ArtworkRow
          label="Cover"
          present={Boolean(game.coverArtworkId)}
          busy={upload.isPending || remove.isPending}
          onChoose={() => coverInput.current?.click()}
          onRemove={() => remove.mutate({ gameID: game.id, kind: "cover" })}
        />
        <ArtworkRow
          label="Banner"
          present={Boolean(game.bannerArtworkId)}
          busy={upload.isPending || remove.isPending}
          onChoose={() => bannerInput.current?.click()}
          onRemove={() => remove.mutate({ gameID: game.id, kind: "banner" })}
        />
      </div>

      <input
        ref={coverInput}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="sr-only"
        aria-label="Choose cover image"
        onChange={(event) => {
          choose("cover", event.target.files?.[0]);
          event.target.value = "";
        }}
      />
      <input
        ref={bannerInput}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="sr-only"
        aria-label="Choose banner image"
        onChange={(event) => {
          choose("banner", event.target.files?.[0]);
          event.target.value = "";
        }}
      />

      {error || upload.isError || remove.isError ? (
        <p className="text-rust-soft mt-3 text-xs" role="alert">
          {error || upload.error?.message || remove.error?.message}
        </p>
      ) : null}
    </section>
  );
}

function ArtworkRow({
  label,
  present,
  busy,
  onChoose,
  onRemove,
}: {
  label: string;
  present: boolean;
  busy: boolean;
  onChoose: () => void;
  onRemove: () => void;
}) {
  return (
    <div className="bg-ink-950/30 flex items-center justify-between gap-3 rounded-xl border border-white/8 px-3.5 py-3">
      <div>
        <p className="text-paper text-sm">{label}</p>
        <p className="text-paper-faint mt-0.5 text-[0.68rem]">
          {present ? "Present" : "Using generated fallback"}
        </p>
      </div>
      <div className="flex items-center gap-1.5">
        <Button type="button" size="sm" variant="secondary" onClick={onChoose} disabled={busy}>
          Replace
        </Button>
        {present ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={onRemove}
            disabled={busy}
            aria-label={`Remove ${label.toLowerCase()} artwork`}
          >
            <Trash2 size={14} />
          </Button>
        ) : null}
      </div>
    </div>
  );
}
