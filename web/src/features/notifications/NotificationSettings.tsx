import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api, type NotificationKind, type NotificationPreferences } from "../../api/client";
import { disablePush, enablePush, pushState, type PushState } from "./push";

const KINDS: NotificationKind[] = ["low_stat", "friend_request", "trade"];

export function Toggle({ label, hint, on, onChange, disabled }: { label: string; hint?: string; on: boolean; onChange: () => void; disabled?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl bg-canvas px-3 py-2">
      <span className="text-sm font-bold">
        {label}
        {hint && <span className="block text-xs font-normal text-muted">{hint}</span>}
      </span>
      <button
        type="button"
        role="switch"
        aria-checked={on}
        aria-label={label}
        disabled={disabled}
        onClick={onChange}
        className={`relative h-7 w-12 shrink-0 rounded-full transition-colors disabled:opacity-50 ${on ? "bg-mint-deep" : "bg-lilac"}`}
      >
        <span className={`absolute top-1 size-5 rounded-full bg-white transition-all ${on ? "left-6" : "left-1"}`} />
      </button>
    </div>
  );
}

const timezone = () => Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";

export function NotificationSettings() {
  const { t, i18n } = useTranslation();
  const qc = useQueryClient();
  const lang = i18n.resolvedLanguage === "en" ? "en" : "uk";
  const [push, setPush] = useState<PushState>("off");
  const [pushError, setPushError] = useState("");

  useEffect(() => {
    void pushState().then(setPush);
  }, []);

  const prefs = useQuery({ queryKey: ["notification-preferences"], queryFn: api.notificationPreferences });

  const save = useMutation({
    mutationFn: api.setNotificationPreferences,
    onSuccess: (data) => qc.setQueryData(["notification-preferences"], data),
  });

  const update = (patch: Partial<NotificationPreferences>) => {
    if (!prefs.data) return;
    const next = { ...prefs.data, ...patch, timezone: timezone(), lang } as NotificationPreferences;
    qc.setQueryData(["notification-preferences"], next);
    save.mutate(next);
  };

  useEffect(() => {
    if (prefs.data && (prefs.data.timezone !== timezone() || prefs.data.lang !== lang)) update({});
  }, [prefs.data?.timezone, prefs.data?.lang, lang]);

  const togglePush = async () => {
    setPushError("");
    try {
      setPush(push === "on" ? await disablePush() : await enablePush());
    } catch (e) {
      const name = e instanceof Error ? e.name : "";
      const message = e instanceof Error ? e.message : String(e);
      setPushError(
        message === "push-disabled"
          ? t("diary.pushOff")
          : name === "SecurityError" || /SSL|certificate/i.test(message)
            ? t("diary.pushCert")
            : message,
      );
    }
  };

  const hint =
    push === "unsupported" ? t("diary.pushUnsupported") : push === "denied" ? t("diary.pushDenied") : t("diary.notificationsHint");
  const p = prefs.data;

  return (
    <>
      <Toggle
        label={t("diary.notifications")}
        hint={hint}
        on={push === "on"}
        disabled={push === "unsupported" || push === "denied"}
        onChange={() => void togglePush()}
      />
      {pushError && <p role="alert" className="text-xs font-bold text-coral-deep">{pushError}</p>}
      {p && (
        <>
          <div className="flex flex-col gap-1 rounded-xl bg-canvas px-3 py-2 text-sm">
            <p className="font-bold">{t("diary.kinds")}</p>
            {KINDS.map((k) => (
              <label key={k} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  className="accent-lilac-deep"
                  checked={!p.disabled.includes(k)}
                  onChange={(e) =>
                    update({ disabled: e.target.checked ? p.disabled.filter((x) => x !== k) : [...p.disabled, k] })
                  }
                />
                {t(`diary.kind.${k}`)}
              </label>
            ))}
            <p className="text-xs text-muted">{t("diary.dailyLimit")}</p>
          </div>
          <Toggle
            label={t("diary.quiet")}
            hint={t("diary.quietHint")}
            on={p.quietHours.enabled}
            onChange={() => update({ quietHours: { ...p.quietHours, enabled: !p.quietHours.enabled } })}
          />
          {p.quietHours.enabled && (
            <div className="grid grid-cols-2 gap-2 text-sm font-bold">
              <label>
                {t("diary.from")}
                <input
                  type="time"
                  className="field mt-1"
                  value={p.quietHours.from}
                  onChange={(e) => e.target.value && update({ quietHours: { ...p.quietHours, from: e.target.value } })}
                />
              </label>
              <label>
                {t("diary.to")}
                <input
                  type="time"
                  className="field mt-1"
                  value={p.quietHours.to}
                  onChange={(e) => e.target.value && update({ quietHours: { ...p.quietHours, to: e.target.value } })}
                />
              </label>
            </div>
          )}
        </>
      )}
    </>
  );
}
