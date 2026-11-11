import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: ReactNode;
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="surface flex flex-col items-center px-6 py-16 text-center">
      <div className="border-brass/20 bg-brass/8 text-brass grid h-14 w-14 place-items-center rounded-2xl border">
        <Icon size={24} strokeWidth={1.6} />
      </div>
      <h2 className="font-display text-paper mt-5 text-2xl">{title}</h2>
      <p className="text-paper-muted mt-2 max-w-md text-sm leading-relaxed">{description}</p>
      {action ? <div className="mt-6">{action}</div> : null}
    </div>
  );
}
