import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { Panel } from "../../components/Panel";
import { useUI } from "../../store/ui";

function Toggle({ label, hint, on, onChange }: { label: string; hint?: string; on: boolean; onChange: () => void }) {
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
        onClick={onChange}
        className={`relative h-7 w-12 shrink-0 rounded-full transition-colors ${on ? "bg-mint-deep" : "bg-lilac"}`}
      >
        <span className={`absolute top-1 size-5 rounded-full bg-white transition-all ${on ? "left-6" : "left-1"}`} />
      </button>
    </div>
  );
}

export function DiaryPage() {
  const { t, i18n } = useTranslation();
  const ui = useUI();
  const [text, setText] = useState("");
  const lang = i18n.resolvedLanguage ?? "uk";
  const demo = t("diary.entries", { returnObjects: true }) as { when: string; text: string }[];
  const entries = [...ui.notes, ...demo];

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!text.trim()) return;
    const when = new Date().toLocaleString(lang, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
    ui.addNote(text.trim(), when);
    setText("");
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
      <section className="flex flex-col gap-4">
        <h1 className="text-3xl">{t("diary.title")}</h1>
        <form onSubmit={submit} className="card flex flex-col gap-3">
          <label className="text-sm font-bold" htmlFor="note">{t("diary.write")}</label>
          <textarea id="note" className="field min-h-20" placeholder={t("diary.placeholder")} value={text} maxLength={280} onChange={(e) => setText(e.target.value)} />
          <button type="submit" className="chunky self-end bg-coral [--edge:#c4573a]">{t("diary.add")}</button>
        </form>
        <ol className="flex flex-col gap-3">
          {entries.map((e, i) => (
            <li key={i} className="card flex gap-4">
              <span aria-hidden className="grid size-12 shrink-0 place-items-center rounded-full bg-blush text-2xl">📖</span>
              <div>
                <p className="text-xs font-bold text-muted">{e.when}</p>
                <p className="font-serif">{e.text}</p>
              </div>
            </li>
          ))}
        </ol>
      </section>

      <aside>
        <Panel title={`⚙️ ${t("diary.settings")}`}>
          <div className="flex flex-col gap-2">
            <p className="text-sm font-bold">{t("diary.language")}</p>
            <div className="flex gap-2">
              {(["uk", "en"] as const).map((l) => (
                <button
                  key={l}
                  aria-pressed={lang === l}
                  onClick={() => void i18n.changeLanguage(l)}
                  className={`chunky ${lang === l ? "bg-coral [--edge:#c4573a]" : "bg-lilac [--edge:#6a48c9]"}`}
                >
                  {l === "uk" ? "🇺🇦 UK" : "🇬🇧 EN"}
                </button>
              ))}
            </div>
          </div>
          <Toggle label={t("diary.sound")} on={ui.sound} onChange={ui.toggleSound} />
          <Toggle label={t("diary.notifications")} hint={t("diary.notificationsHint")} on={ui.notifications} onChange={ui.toggleNotifications} />
          <Toggle label={t("diary.quiet")} hint={t("diary.quietHint")} on={ui.quiet} onChange={ui.toggleQuiet} />
          {ui.quiet && (
            <div className="grid grid-cols-2 gap-2 text-sm font-bold">
              <label>{t("diary.from")}<input type="time" className="field mt-1" value={ui.quietFrom} onChange={(e) => ui.setQuietHours(e.target.value, ui.quietTo)} /></label>
              <label>{t("diary.to")}<input type="time" className="field mt-1" value={ui.quietTo} onChange={(e) => ui.setQuietHours(ui.quietFrom, e.target.value)} /></label>
            </div>
          )}
        </Panel>
      </aside>
    </div>
  );
}
