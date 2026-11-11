const API_BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";

export type ArtworkVariant = "card" | "detail";

export function artworkUrl(artworkId: string | undefined, variant: ArtworkVariant): string | null {
  if (!artworkId) return null;
  return `${API_BASE}/artwork/${encodeURIComponent(artworkId)}/${variant}`;
}
