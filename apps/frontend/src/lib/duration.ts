const durationFormatter = new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 });

export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return seconds > 0 ? "<1m" : "0m";
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours === 0) return `${minutes}m`;
  if (minutes === 0) return `${durationFormatter.format(hours)}h`;
  return `${durationFormatter.format(hours)}h ${minutes}m`;
}
