import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { UpdateGameRequest } from "./generated";
import {
  getCurrentScan,
  getGame,
  getLibraries,
  getStats,
  listGames,
  listScans,
  setFavorite,
  startScan,
  updateGame,
  type ListGamesParams,
} from "./client";

export const queryKeys = {
  games: (params: ListGamesParams) => ["games", params] as const,
  game: (id: string) => ["game", id] as const,
  stats: ["stats"] as const,
  libraries: ["libraries"] as const,
  currentScan: ["scan", "current"] as const,
  scans: ["scans"] as const,
};

export function useGames(params: ListGamesParams) {
  return useQuery({
    queryKey: queryKeys.games(params),
    queryFn: () => listGames(params),
    placeholderData: (previous) => previous,
  });
}

export function useGame(id: string) {
  return useQuery({
    queryKey: queryKeys.game(id),
    queryFn: () => getGame(id),
    enabled: Boolean(id),
  });
}

export function useStats() {
  return useQuery({ queryKey: queryKeys.stats, queryFn: getStats });
}

export function useLibraries() {
  return useQuery({ queryKey: queryKeys.libraries, queryFn: getLibraries });
}

export function useCurrentScan() {
  return useQuery({
    queryKey: queryKeys.currentScan,
    queryFn: getCurrentScan,
    refetchInterval: (query) => (query.state.data?.running ? 1000 : 5000),
  });
}

export function useScanHistory() {
  return useQuery({ queryKey: queryKeys.scans, queryFn: listScans });
}

function useInvalidateLibrary() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: ["games"] });
    void queryClient.invalidateQueries({ queryKey: queryKeys.stats });
    void queryClient.invalidateQueries({ queryKey: queryKeys.libraries });
  };
}

export function useSetFavorite() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({ id, favorite }: { id: string; favorite: boolean }) =>
      setFavorite(id, favorite),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.game(variables.id) });
      invalidateLibrary();
    },
  });
}

export function useUpdateGame() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateGameRequest }) => updateGame(id, body),
    onSuccess: (game) => {
      queryClient.setQueryData(queryKeys.game(game.id), game);
      invalidateLibrary();
    },
  });
}

export function useStartScan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (allowMissing: boolean) => startScan(allowMissing),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.currentScan });
      void queryClient.invalidateQueries({ queryKey: queryKeys.scans });
    },
  });
}
