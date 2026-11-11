import type { ReactNode } from "react";

type Tone = "brass" | "quiet" | "danger" | "success";

const tones: Record<Tone, string> = {
  brass: "border-brass/25 bg-brass/10 text-brass-soft",
  quiet: "border-white/10 bg-white/[0.04] text-paper-muted",
  danger: "border-rust/30 bg-rust/10 text-rust-soft",
  success: "border-emerald-400/20 bg-emerald-400/10 text-emerald-200",
};

export function Badge({ children, tone = "quiet" }: { children: ReactNode; tone?: Tone }) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-1 text-[0.68rem] font-medium leading-none ${tones[tone]}`}
    >
      {children}
    </span>
  );
}
