import { X } from "lucide-react";
import { useEffect, useId, useRef, type ReactNode } from "react";

interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
}

export function Dialog({ open, onClose, title, description, children }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      aria-describedby={description ? descriptionId : undefined}
      onClose={onClose}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === ref.current) onClose();
      }}
      className="m-auto w-[min(92vw,44rem)] max-h-[88vh] overflow-y-auto rounded-[1.6rem] border border-white/12 bg-ink-900 p-0 text-paper shadow-[0_40px_120px_-20px_rgba(0,0,0,0.95)] backdrop:bg-ink-950/80 backdrop:backdrop-blur-sm"
    >
      <div className="sticky top-0 z-10 flex items-start justify-between gap-6 border-b border-white/8 bg-ink-900/95 px-6 py-5 backdrop-blur">
        <div>
          <h2 id={titleId} className="font-display text-2xl text-paper">
            {title}
          </h2>
          {description ? (
            <p id={descriptionId} className="mt-1 text-sm text-paper-muted">
              {description}
            </p>
          ) : null}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close dialog"
          className="grid h-9 w-9 shrink-0 place-items-center rounded-full border border-white/10 text-paper-muted transition hover:bg-white/5 hover:text-paper"
        >
          <X size={17} />
        </button>
      </div>
      <div className="px-6 py-5">{children}</div>
    </dialog>
  );
}
