import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api, ApiError } from "../../api/client";

export function AuthForm() {
  const { t } = useTranslation();
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
    <form className="card mx-auto flex max-w-sm flex-col gap-3" onSubmit={onSubmit}>
      <h1 className="text-2xl">{t("auth.title")}</h1>
      <input className="field" type="email" placeholder={t("auth.email")} value={email} onChange={(e) => setEmail(e.target.value)} required />
      <input
        className="field"
        type="password"
        placeholder={t("auth.password")}
        value={password}
        minLength={8}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      {mode === "register" && (
        <input
          className="field"
          placeholder={t("auth.nickname")}
          value={nickname}
          minLength={3}
          maxLength={20}
          onChange={(e) => setNickname(e.target.value)}
          required
        />
      )}
      {submit.error instanceof ApiError && <p role="alert" className="text-sm font-bold text-coral-deep">{submit.error.message}</p>}
      <button type="submit" disabled={submit.isPending} className="chunky justify-center bg-coral [--edge:#c4573a]">
        {mode === "login" ? t("auth.login") : t("auth.register")}
      </button>
      <button type="button" className="text-sm font-bold text-lilac-deep underline" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? t("auth.toRegister") : t("auth.toLogin")}
      </button>
    </form>
  );
}
