interface GamePosterProps {
  title: string;
  className?: string;
  showText?: boolean;
}

function hashTitle(title: string): number {
  let hash = 0;
  for (let index = 0; index < title.length; index += 1) {
    hash = (hash * 31 + title.charCodeAt(index)) >>> 0;
  }
  return hash;
}

function initials(title: string): string {
  const words = title
    .replace(/[^\p{L}\p{N}\s]/gu, " ")
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) return "?";
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return `${words[0][0]}${words[1][0]}`.toUpperCase();
}

export function GamePoster({ title, className = "", showText = true }: GamePosterProps) {
  const hash = hashTitle(title);
  const hue = hash % 360;
  const secondHue = (hue + 38) % 360;
  const angle = 120 + (hash % 90);

  return (
    <div
      aria-hidden="true"
      className={`poster relative isolate overflow-hidden ${className}`}
      style={{
        backgroundImage: `linear-gradient(${angle}deg, hsl(${hue} 34% 22%) 0%, hsl(${secondHue} 38% 12%) 58%, #12100d 100%)`,
      }}
    >
      <div className="poster-glow" />
      <div className="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-ink-950/65 to-transparent" />
      {showText ? (
        <div className="relative flex h-full flex-col justify-between p-4">
          <span className="text-[0.62rem] font-semibold uppercase tracking-[0.28em] text-paper/45">
            Oryn
          </span>
          <span className="font-display text-4xl leading-none text-paper/90 drop-shadow-lg sm:text-5xl">
            {initials(title)}
          </span>
        </div>
      ) : null}
    </div>
  );
}
