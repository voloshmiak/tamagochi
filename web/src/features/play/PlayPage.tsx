import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Panel } from "../../components/Panel";
import { DEMO } from "../home/demo";
import { BerryCatch } from "./BerryCatch";
import { Hide } from "./Hide";
import { Memo } from "./Memo";

const GAMES = [
  { id: "berries", icon: "🍓", bg: "bg-coral", Game: BerryCatch },
  { id: "memo", icon: "🃏", bg: "bg-mint", Game: Memo },
  { id: "hide", icon: "🫣", bg: "bg-lilac", Game: Hide },
] as const;

export function PlayPage() {
  const { t } = useTranslation();
  const [active, setActive] = useState<string | null>(null);
  const current = GAMES.find((g) => g.id === active);

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
      <section className="flex flex-col gap-4">
        <h1 className="text-3xl">{t("play.title")}</h1>
        {current ? (
          <current.Game onBack={() => setActive(null)} />
        ) : (
          <ul className="grid gap-4 sm:grid-cols-3">
            {GAMES.map((g) => (
              <li key={g.id} className="card flex flex-col items-center gap-3 text-center">
                <span aria-hidden className={`grid size-20 place-items-center rounded-full text-4xl ${g.bg}`}>{g.icon}</span>
                <h2 className="text-lg">{t(`play.list.${g.id}`)}</h2>
                <p className="text-sm text-muted">{t(`play.desc.${g.id}`)}</p>
                <button onClick={() => setActive(g.id)} className="chunky bg-lilac-deep text-white [--edge:#3f2a85]">{t("play.start")}</button>
              </li>
            ))}
          </ul>
        )}
      </section>
      <aside>
        <Panel title={`🏆 ${t("play.friends")}`}>
          <ol className="flex flex-col gap-2">
            {DEMO.friends.map((f, i) => (
              <li key={f.nick} className={`flex items-center gap-3 rounded-xl px-3 py-2 text-sm ${i === 0 ? "bg-coral/70" : "bg-canvas"}`}>
                <span className="w-5 font-extrabold">{i + 1}</span>
                <span className="flex-1 font-bold">{f.nick}<span className="block text-xs font-normal text-muted">{f.pet}</span></span>
                <span className="font-extrabold">{f.score}</span>
              </li>
            ))}
          </ol>
          <button className="chunky justify-center bg-mint [--edge:#23804a]">{t("play.visit")}</button>
        </Panel>
      </aside>
    </div>
  );
}
