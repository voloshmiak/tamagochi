import { useTranslation } from "react-i18next";
import type { User } from "../api/client";
import { useUI, type Tab } from "../store/ui";

const TABS: Tab[] = ["room", "shop", "play", "diary"];

export function Header({ user, hasPet, onLogout }: { user?: User; hasPet: boolean; onLogout: () => void }) {
  const { t, i18n } = useTranslation();
  const { tab, setTab, sound, toggleSound, berries } = useUI();

  return (
    <header className="flex flex-wrap items-center gap-x-6 gap-y-3 bg-white/80 px-4 py-3 backdrop-blur sm:px-6">
      <div className="flex items-center gap-3">
        <span aria-hidden className="grid size-10 place-items-center rounded-full bg-blush text-xl">🐾</span>
        <div className="leading-tight">
          <p className="text-2xl font-extrabold text-coral-deep">{t("brand")}</p>
          <p className="text-xs text-muted">{t("tagline")}</p>
        </div>
      </div>

      {user && hasPet && (
        <nav aria-label="main" className="order-last flex w-full gap-1 overflow-x-auto rounded-full bg-lilac/40 p-1.5 lg:order-none lg:w-auto">
          {TABS.map((k) => (
            <button
              key={k}
              onClick={() => setTab(k)}
              aria-current={tab === k ? "page" : undefined}
              className={`shrink-0 whitespace-nowrap rounded-full px-4 py-2 text-sm font-bold sm:px-5 ${tab === k ? "bg-coral shadow-[0_3px_0_rgba(0,0,0,.15)]" : "text-ink/80 hover:bg-white/60"}`}
            >
              {t(`nav.${k}`)}
            </button>
          ))}
        </nav>
      )}

      <div className="ml-auto flex flex-wrap items-center justify-end gap-2 sm:gap-3">
        {user && hasPet && (
          <span className="rounded-full bg-blush px-4 py-1.5 text-sm font-extrabold">
            🍓 {berries.toLocaleString(i18n.language)} <span className="text-xs font-bold text-muted">{t("common.berries")}</span>
          </span>
        )}
        <select
          aria-label={t("common.language")}
          value={i18n.resolvedLanguage}
          onChange={(e) => void i18n.changeLanguage(e.target.value)}
          className="rounded-full bg-lilac/40 px-3 py-2 text-sm font-bold"
        >
          <option value="uk">UK</option>
          <option value="en">EN</option>
        </select>
        <button
          onClick={toggleSound}
          aria-pressed={sound}
          aria-label={t("common.sound")}
          className="grid size-10 place-items-center rounded-full bg-lilac text-lg"
        >
          {sound ? "🔊" : "🔇"}
        </button>
        {user && (
          <div className="flex items-center gap-2 sm:gap-3">
            <span className="max-w-[7rem] truncate rounded-full bg-mint px-3 py-1 text-sm font-bold text-mint-deep">{user.nickname}</span>
            <button className="text-sm font-bold text-coral-deep underline" onClick={onLogout}>
              {t("common.logout")}
            </button>
          </div>
        )}
      </div>
    </header>
  );
}