import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { UpdateGameRequest } from "./generated";
import {
  addGameToCollection,
  createCollection,
  createSession,
  deleteArtwork,
  deleteCollection,
  deleteSession,
  getCollection,
  getCurrentScan,
  getGame,
  getLibraries,
  getStats,
  listCollections,
  listCollectionsForGame,
  listGames,
  listScans,
  listSessions,
  removeGameFromCollection,
  setFavorite,
  startScan,
  updateCollection,
  updateGame,
  updateSession,
  uploadArtwork,
  type ListGamesParams,
  type SessionInput,
} from "./client";

export const queryKeys = {
  games: (params: ListGamesParams) => ["games", params] as const,
  game: (id: string) => ["game", id] as const,
  sessions: (gameID: string) => ["sessions", gameID] as const,
  collections: ["collections"] as const,
  collection: (id: string) => ["collection", id] as const,
  gameCollections: (gameID: string) => ["game", gameID, "collections"] as const,
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
    mutationFn: ({ id, favorite }: { id: string; favorite: boolean }) => setFavorite(id, favorite),
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

export function useCollections() {
  return useQuery({ queryKey: queryKeys.collections, queryFn: listCollections });
}

export function useCollection(id: string) {
  return useQuery({
    queryKey: queryKeys.collection(id),
    queryFn: () => getCollection(id),
    enabled: Boolean(id),
  });
}

export function useGameCollections(gameID: string) {
  return useQuery({
    queryKey: queryKeys.gameCollections(gameID),
    queryFn: () => listCollectionsForGame(gameID),
    enabled: Boolean(gameID),
  });
}

export function useCreateCollection() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, description }: { name: string; description: string }) =>
      createCollection(name, description),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.collections });
    },
  });
}

export function useUpdateCollection() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name, description }: { id: string; name: string; description: string }) =>
      updateCollection(id, name, description),
    onSuccess: (collection) => {
      queryClient.setQueryData(queryKeys.collection(collection.id), collection);
      void queryClient.invalidateQueries({ queryKey: queryKeys.collections });
    },
  });
}

export function useDeleteCollection() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCollection(id),
    onSuccess: (_data, id) => {
      queryClient.removeQueries({ queryKey: queryKeys.collection(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.collections });
    },
  });
}

export function useToggleCollectionMembership() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      collectionID,
      gameID,
      contains,
    }: {
      collectionID: string;
      gameID: string;
      contains: boolean;
    }) =>
      contains
        ? removeGameFromCollection(collectionID, gameID)
        : addGameToCollection(collectionID, gameID),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.gameCollections(variables.gameID) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.collections });
      void queryClient.invalidateQueries({ queryKey: ["games"] });
    },
  });
}

export function useSessions(gameID: string) {
  return useQuery({
    queryKey: queryKeys.sessions(gameID),
    queryFn: () => listSessions(gameID),
    enabled: Boolean(gameID),
  });
}

export function useCreateSession() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({ gameID, input }: { gameID: string; input: SessionInput }) =>
      createSession(gameID, input),
    onSuccess: (session) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.sessions(session.gameId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.game(session.gameId) });
      invalidateLibrary();
    },
  });
}

export function useUpdateSession() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({
      sessionID,
      input,
    }: {
      sessionID: string;
      gameID: string;
      input: SessionInput;
    }) => updateSession(sessionID, input),
    onSuccess: (session) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.sessions(session.gameId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.game(session.gameId) });
      invalidateLibrary();
    },
  });
}

export function useDeleteSession() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({ sessionID }: { sessionID: string; gameID: string }) => deleteSession(sessionID),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.sessions(variables.gameID) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.game(variables.gameID) });
      invalidateLibrary();
    },
  });
}

export function useUploadArtwork() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({
      gameID,
      kind,
      file,
    }: {
      gameID: string;
      kind: "cover" | "banner";
      file: File;
    }) => uploadArtwork(gameID, kind, file),
    onSuccess: (game) => {
      queryClient.setQueryData(queryKeys.game(game.id), game);
      invalidateLibrary();
    },
  });
}

export function useDeleteArtwork() {
  const queryClient = useQueryClient();
  const invalidateLibrary = useInvalidateLibrary();
  return useMutation({
    mutationFn: ({ gameID, kind }: { gameID: string; kind: "cover" | "banner" }) =>
      deleteArtwork(gameID, kind),
    onSuccess: (game) => {
      queryClient.setQueryData(queryKeys.game(game.id), game);
      invalidateLibrary();
    },
  });
}
