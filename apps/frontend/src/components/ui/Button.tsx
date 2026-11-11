import type { ButtonHTMLAttributes, ReactNode } from "react";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md";

const variants: Record<Variant, string> = {
  primary:
    "border-brass/35 bg-brass text-ink-950 hover:bg-brass-soft hover:border-brass-soft shadow-[0_10px_30px_-14px_rgba(201,168,106,0.65)]",
  secondary:
    "border-white/12 bg-white/[0.045] text-paper hover:border-brass/35 hover:bg-white/[0.075]",
  ghost: "border-transparent bg-transparent text-paper-muted hover:bg-white/[0.05] hover:text-paper",
  danger: "border-rust/30 bg-rust/12 text-rust-soft hover:bg-rust/20",
};

const sizes: Record<Size, string> = {
  sm: "h-9 px-3.5 text-xs",
  md: "h-11 px-5 text-sm",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  children: ReactNode;
}

export function Button({
  variant = "secondary",
  size = "md",
  className = "",
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`inline-flex items-center justify-center gap-2 rounded-full border font-medium tracking-[-0.01em] transition disabled:cursor-not-allowed disabled:opacity-55 ${variants[variant]} ${sizes[size]} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
