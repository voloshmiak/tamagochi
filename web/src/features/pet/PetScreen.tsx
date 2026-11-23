import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiError, type Action, type Stats } from "../../api/client";

const ACTIONS: Record<Action, string> = { feed: "Нагодувати", play: "Грати", sleep: "Спати", wash: "Мити" };
const STATS: Record<keyof Stats, string> = {
  satiety: "Ситість",
  mood: "Настрій",
  energy: "Енергія",
  cleanliness: "Чистота",
};

export function PetScreen() {
  const qc = useQueryClient();
  const pet = useQuery({ queryKey: ["pet"], queryFn: api.myPet, retry: false });
  const act = useMutation({
    mutationFn: api.act,
    onSuccess: (res) => qc.setQueryData(["pet"], res.pet),
  });

  if (pet.isLoading) return null;
  if (pet.error instanceof ApiError && pet.error.status === 404) return <CreatePet />;
  if (!pet.data) return <p className="error">Не вдалося завантажити улюбленця</p>;

  return (
    <section className="card">
      <h2>{pet.data.name}</h2>
      {(Object.keys(STATS) as (keyof Stats)[]).map((k) => (
        <label key={k} className="stat">
          {STATS[k]}
          <progress max={100} value={pet.data.stats[k]} />
        </label>
      ))}
      <div className="actions">
        {(Object.keys(ACTIONS) as Action[]).map((a) => (
          <button key={a} disabled={act.isPending} onClick={() => act.mutate(a)}>
            {ACTIONS[a]}
          </button>
        ))}
      </div>
      {act.error instanceof ApiError && <p className="error">{act.error.message}</p>}
    </section>
  );
}

function CreatePet() {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const create = useMutation({
    mutationFn: () => api.createPet(name, "cat"),
    onSuccess: (pet) => qc.setQueryData(["pet"], pet),
  });
  return (
    <form
      className="card"
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate();
      }}
    >
      <input placeholder="Ім'я улюбленця" value={name} minLength={2} maxLength={20} onChange={(e) => setName(e.target.value)} />
      <button type="submit">Створити улюбленця</button>
      {create.error instanceof ApiError && <p className="error">{create.error.message}</p>}
    </form>
  );
}
