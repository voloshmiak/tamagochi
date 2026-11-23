import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../../api/client";

export function AuthForm() {
  const qc = useQueryClient();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");

  const submit = useMutation({
    mutationFn: async () => {
      if (mode === "register") await api.register(email, password, nickname);
      return api.login(email, password);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me"] }),
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    submit.mutate();
  };

  return (
    <form className="card" onSubmit={onSubmit}>
      <input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} required />
      <input
        type="password"
        placeholder="Пароль"
        value={password}
        minLength={8}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      {mode === "register" && (
        <input placeholder="Нікнейм" value={nickname} onChange={(e) => setNickname(e.target.value)} required />
      )}
      {submit.error instanceof ApiError && <p className="error">{submit.error.message}</p>}
      <button type="submit" disabled={submit.isPending}>
        {mode === "login" ? "Увійти" : "Зареєструватися"}
      </button>
      <button type="button" className="link" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? "Зареєструватися" : "Увійти"}
      </button>
    </form>
  );
}
