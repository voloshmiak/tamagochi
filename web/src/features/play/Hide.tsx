import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUI } from "../../store/ui";
import { GameShell } from "./GameShell";

const SPOTS = ["🛋️", "🪴", "📦"];
const pick = () => Math.floor(Math.random() * SPOTS.length);

export function Hide({ onBack }: { onBack: () => void }) {
  const { t } = useTranslation();
  const addBerries = useUI((s) => s.addBerries);
  const [hidden, setHidden] = useState(pick);
  const [tries, setTries] = useState<number[]>([]);
  const found = tries.includes(hidden);
  const reward = tries.length === 1 ? 20 : 10;

  useEffect(() => {
    if (found) addBerries(reward);
  }, [found]);

  const again = () => {
    setHidden(pick());
    setTries([]);
  };

  return (
    <GameShell title={t("play.list.hide")} onBack={onBack}>
      <p className="text-center font-serif text-lg font-bold">
        {found ? t("play.hideFound") : tries.length > 0 ? t("play.hideMiss") : t("play.hidePrompt")}
      </p>
      <ul className="grid grid-cols-3 gap-3">
        {SPOTS.map((s, i) => (
          <li key={i}>
            <button
              onClick={() => !found && !tries.includes(i) && setTries((x) => [...x, i])}
              aria-label={s}
              className={`grid h-32 w-full place-items-center rounded-3xl text-6xl ${found && i === hidden ? "bg-mint" : tries.includes(i) ? "bg-lilac/40" : "bg-blush"}`}
            >
              {found && i === hidden ? "🐱" : s}
            </button>
          </li>
        ))}
      </ul>
      {found && (
        <div className="flex flex-wrap items-center justify-center gap-3">
          <p className="font-serif text-xl font-bold">{t("play.reward")}: +{reward} 🍓</p>
          <button className="chunky bg-coral [--edge:#c4573a]" onClick={again}>{t("play.again")}</button>
        </div>
      )}
    </GameShell>
  );
}
