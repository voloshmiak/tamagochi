import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./api/client";
import { AuthForm } from "./features/auth/AuthForm";
import { PetScreen } from "./features/pet/PetScreen";

export function App() {
  const qc = useQueryClient();
  const me = useQuery({ queryKey: ["me"], queryFn: api.me, retry: false });
  const logout = useMutation({ mutationFn: api.logout, onSuccess: () => qc.clear() });

  return (
    <main>
      <header>
        <h1>Тамагочі</h1>
        {me.data && (
          <button className="link" onClick={() => logout.mutate()}>
            Вийти
          </button>
        )}
      </header>
      {me.isLoading ? null : me.data ? <PetScreen /> : <AuthForm />}
    </main>
  );
}
