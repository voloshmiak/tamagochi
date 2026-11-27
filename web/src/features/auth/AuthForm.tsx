import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api, ApiError, oauthStartURL, type AuthProvider } from "../../api/client";

const OAUTH_MESSAGES: Record<string, string> = {
  error: "auth.oauthError",
  denied: "auth.oauthDenied",
  no_email: "auth.oauthNoEmail",
};

function takeOAuthResult(): string | null {
  const params = new URLSearchParams(window.location.search);
  const result = params.get("oauth");
  if (!result) return null;
  params.delete("oauth");
  const query = params.toString();
  window.history.replaceState(null, "", window.location.pathname + (query ? `?${query}` : "") + window.location.hash);
  return OAUTH_MESSAGES[result] ?? OAUTH_MESSAGES.error;
}

const PROVIDER_LABELS: Record<AuthProvider, string> = { google: "auth.withGoogle", github: "auth.withGithub" };

export function AuthForm() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");
  const [oauthMessage] = useState(takeOAuthResult);
  const providers = useQuery({ queryKey: ["auth-providers"], queryFn: api.authProviders, staleTime: Infinity });

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

  const error = submit.error instanceof ApiError ? submit.error : null;
  const errorText =
    error?.status === 429 && error.code === "rate_limited"
      ? t("auth.tooMany", { minutes: Math.max(1, Math.ceil(error.retryAfter / 60)) })
      : error?.message;
  const list = providers.data?.providers ?? [];

  return (
    <form className="card mx-auto flex max-w-sm flex-col gap-3" onSubmit={onSubmit}>
      <h1 className="text-2xl">{t("auth.title")}</h1>
      {oauthMessage && <p role="alert" className="text-sm font-bold text-coral-deep">{t(oauthMessage)}</p>}
      <input className="field" type="email" placeholder={t("auth.email")} value={email} maxLength={254} onChange={(e) => setEmail(e.target.value)} required />
      <input
        className="field"
        type="password"
        placeholder={t("auth.password")}
        value={password}
        minLength={8}
        maxLength={72}
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
      {errorText && <p role="alert" className="text-sm font-bold text-coral-deep">{errorText}</p>}
      <button type="submit" disabled={submit.isPending} className="chunky justify-center bg-coral [--edge:#c4573a]">
        {mode === "login" ? t("auth.login") : t("auth.register")}
      </button>
      {list.length > 0 && (
        <>
          <p className="text-center text-xs font-bold uppercase text-muted">{t("auth.or")}</p>
          {list.map((p) => (
            <a key={p} href={oauthStartURL(p)} className="chunky justify-center bg-lilac [--edge:#6a48c9]">
              {t(PROVIDER_LABELS[p])}
            </a>
          ))}
        </>
      )}
      <button type="button" className="text-sm font-bold text-lilac-deep underline" onClick={() => setMode(mode === "login" ? "register" : "login")}>
        {mode === "login" ? t("auth.toRegister") : t("auth.toLogin")}
      </button>
    </form>
  );
}
