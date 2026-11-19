import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./api/client";
import { Footer } from "./components/Footer";
import { Header } from "./components/Header";
import { MusicPlayer } from "./components/MusicPlayer";
import { AuthForm } from "./features/auth/AuthForm";
import { DiaryPage } from "./features/diary/DiaryPage";
import { HomePage } from "./features/home/HomePage";
import { PlayPage } from "./features/play/PlayPage";
import { ShopPage } from "./features/shop/ShopPage";
import { useUI } from "./store/ui";

export function App() {
  const qc = useQueryClient();
  const tab = useUI((s) => s.tab);
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, retry: false });
  // Тот же ключ, что и в HomePage: запрос выполняется один раз и кэшируется.
  const pet = useQuery({ queryKey: ["pet"], queryFn: api.myPet, retry: false, enabled: !!me.data });
  const hasPet = !!pet.data;
  const logout = useMutation({ mutationFn: api.logout, onSuccess: () => qc.clear() });

  return (
    <div className="flex min-h-dvh flex-col">
      <MusicPlayer />
      <Header user={me.data} hasPet={hasPet} onLogout={() => logout.mutate()} />
      <main className="mx-auto w-full max-w-[1280px] flex-1 px-4 py-6">
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
      </main>
      <Footer />
    </div>
  );
}