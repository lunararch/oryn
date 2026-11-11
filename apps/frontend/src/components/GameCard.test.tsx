import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { Game } from "../lib/api/generated";
import { GameCard } from "./GameCard";

const game: Game = {
  id: "game-1",
  title: "The Longest Road",
  description: "A quiet story about leaving home.",
  developer: "Paper Lantern",
  publisher: "",
  releaseDate: "2024",
  version: "1.0",
  relativePath: "The Longest Road",
  availability: "available",
  completionStatus: "unplayed",
  favorite: false,
  notes: "",
  addedAt: "2025-01-02T03:04:05Z",
  lastSeenAt: "2025-01-02T03:04:05Z",
};

describe("GameCard", () => {
  it("renders game information and links to details", () => {
    render(
      <MemoryRouter>
        <GameCard game={game} view="grid" onToggleFavorite={vi.fn()} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "The Longest Road" })).toBeInTheDocument();
    expect(screen.getByText("Paper Lantern")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /the longest road/i })).toHaveAttribute(
      "href",
      "/games/game-1",
    );
  });

  it("calls the favorite handler without following the card link", async () => {
    const onToggleFavorite = vi.fn();
    render(
      <MemoryRouter>
        <GameCard game={game} view="grid" onToggleFavorite={onToggleFavorite} />
      </MemoryRouter>,
    );

    await userEvent.click(screen.getByRole("button", { name: /add to favorites/i }));
    expect(onToggleFavorite).toHaveBeenCalledWith(game, true);
  });

  it("shows a missing badge for unavailable games", () => {
    render(
      <MemoryRouter>
        <GameCard
          game={{ ...game, availability: "missing" }}
          view="list"
          onToggleFavorite={vi.fn()}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText("Missing")).toBeInTheDocument();
  });
});
