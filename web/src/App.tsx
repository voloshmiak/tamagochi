import { lazy, Suspense } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./api/client";
import { Footer } from "./components/Footer";
import { Header } from "./components/Header";
import { MusicPlayer } from "./components/MusicPlayer";
import { HomePage } from "./features/home/HomePage";
import { useUI } from "./store/ui";

const AuthForm = lazy(() => import("./features/auth/AuthForm").then((m) => ({ default: m.AuthForm })));
const ShopPage = lazy(() => import("./features/shop/ShopPage").then((m) => ({ default: m.ShopPage })));
const PlayPage = lazy(() => import("./features/play/PlayPage").then((m) => ({ default: m.PlayPage })));
const DiaryPage = lazy(() => import("./features/diary/DiaryPage").then((m) => ({ default: m.DiaryPage })));

export function App() {
  const qc = useQueryClient();
  const tab = useUI((s) => s.tab);
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, retry: false });
  const pet = useQuery({ queryKey: ["pet"], queryFn: api.myPet, retry: false, enabled: !!me.data });
  const hasPet = !!pet.data;
  const logout = useMutation({ mutationFn: api.logout, onSuccess: () => qc.clear() });

  return (
    <div className="flex min-h-dvh flex-col">
      <MusicPlayer />
      <Header user={me.data} hasPet={hasPet} onLogout={() => logout.mutate()} />
      <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 py-6">
        <Suspense fallback={null}>
        {me.isLoading ? null : !me.data ? (
          <AuthForm />
        ) : tab === "room" || !hasPet ? (
          <HomePage />
        ) : tab === "shop" ? (
          <ShopPage />
        ) : tab === "play" ? (
          <PlayPage />
        ) : (
          <DiaryPage />
        )}
        </Suspense>
      </main>
      <Footer />
    </div>
  );
}