import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUI } from "../../store/ui";
import { GameShell } from "./GameShell";

const EMOJI = ["🍓", "🐟", "🎾", "🧼", "🍎", "🪶"];
const deal = () => [...EMOJI, ...EMOJI].map((e, id) => ({ id, e })).sort(() => Math.random() - 0.5);

export function Memo({ onBack }: { onBack: () => void }) {
  const { t } = useTranslation();
  const addBerries = useUI((s) => s.addBerries);
  const [cards, setCards] = useState(deal);
  const [open, setOpen] = useState<number[]>([]);
  const [matched, setMatched] = useState<number[]>([]);
  const [moves, setMoves] = useState(0);
  const done = matched.length === cards.length;
  const reward = Math.max(10, 40 - moves * 2);

  useEffect(() => {
    if (done) addBerries(reward);
  }, [done]);

  const flip = (id: number) => {
    if (open.length === 2 || open.includes(id) || matched.includes(id)) return;
    const next = [...open, id];
    setOpen(next);
    if (next.length < 2) return;
    setMoves((m) => m + 1);
    const [a, b] = next.map((i) => cards.find((c) => c.id === i)!);
    if (a.e === b.e) {
      setMatched((m) => [...m, a.id, b.id]);
      setOpen([]);
    } else {
      setTimeout(() => setOpen([]), 700);
    }
  };

  const reset = () => {
    setCards(deal());
    setOpen([]);
    setMatched([]);
    setMoves(0);
  };

  return (
    <GameShell title={t("play.list.memo")} onBack={onBack} stats={<span>{t("play.moves")}: {moves}</span>}>
      <ul className="grid grid-cols-4 gap-3">
        {cards.map((c) => {
          const shown = open.includes(c.id) || matched.includes(c.id);
          return (
            <li key={c.id}>
              <button
                onClick={() => flip(c.id)}
                aria-label={shown ? c.e : "?"}
                className={`grid aspect-square w-full place-items-center rounded-2xl text-4xl transition-colors ${matched.includes(c.id) ? "bg-mint" : shown ? "bg-white shadow-inner" : "bg-lilac"}`}
              >
                {shown ? c.e : "❔"}
              </button>
            </li>
          );
        })}
      </ul>
      {done && (
        <div className="flex flex-wrap items-center justify-center gap-3">
          <p className="font-serif text-xl font-bold">{t("play.reward")}: +{reward} 🍓</p>
          <button className="chunky bg-coral [--edge:#c4573a]" onClick={reset}>{t("play.again")}</button>
        </div>
      )}
    </GameShell>
  );
}
