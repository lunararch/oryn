import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { Shell } from "./Shell";
import { DashboardPage } from "../pages/DashboardPage";
import { GameDetailPage } from "../pages/GameDetailPage";
import { LibraryPage } from "../pages/LibraryPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { SettingsPage } from "../pages/SettingsPage";
import { CollectionsPage } from "../pages/CollectionsPage";
import { CollectionDetailPage } from "../pages/CollectionDetailPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<Shell />}>
            <Route index element={<DashboardPage />} />
            <Route path="library" element={<LibraryPage />} />
            <Route path="games/:gameId" element={<GameDetailPage />} />
            <Route path="collections" element={<CollectionsPage />} />
            <Route path="collections/:collectionId" element={<CollectionDetailPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
