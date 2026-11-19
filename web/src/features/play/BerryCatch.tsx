import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUI } from "../../store/ui";
import { GameShell } from "./GameShell";

const DURATION = 20;

export function BerryCatch({ onBack }: { onBack: () => void }) {
  const { t } = useTranslation();
  const addBerries = useUI((s) => s.addBerries);
  const [round, setRound] = useState(0);
  const [left, setLeft] = useState(DURATION);
  const [score, setScore] = useState(0);
  const [drops, setDrops] = useState<{ id: number; x: number }[]>([]);
  const nextId = useRef(0);
  const done = left <= 0;

  useEffect(() => {
    setLeft(DURATION);
    setScore(0);
    setDrops([]);
    const tick = setInterval(() => setLeft((l) => Math.max(0, l - 1)), 1000);
    return () => clearInterval(tick);
  }, [round]);

  useEffect(() => {
    if (done) return;
    const spawn = setInterval(() => setDrops((d) => [...d, { id: nextId.current++, x: 5 + Math.random() * 85 }]), 650);
    return () => clearInterval(spawn);
  }, [done, round]);

  useEffect(() => {
    if (done && score > 0) addBerries(score * 2);
  }, [done]);

  const remove = (id: number) => setDrops((d) => d.filter((i) => i.id !== id));

  return (
    <GameShell
      title={t("play.list.berries")}
      onBack={onBack}
      stats={<><span>⏱ {left}{t("play.sec")}</span><span>{t("play.score")}: {score}</span></>}
    >
      <div className="relative h-[380px] overflow-hidden rounded-3xl bg-gradient-to-b from-[#ffe9df] to-[#ffd3c4]">
        {!done &&
          drops.map((d) => (
            <button
              key={d.id}
              aria-label={t("play.list.berries")}
              onClick={() => {
                setScore((s) => s + 1);
                remove(d.id);
              }}
              onAnimationEnd={() => remove(d.id)}
              className="fall absolute text-4xl"
              style={{ left: `${d.x}%` }}
            >
              🍓
            </button>
          ))}
        {done && (
          <div className="absolute inset-0 grid place-content-center justify-items-center gap-3 text-center">
            <p className="font-serif text-2xl font-bold">{t("play.reward")}: +{score * 2} 🍓</p>
            <button className="chunky bg-coral [--edge:#c4573a]" onClick={() => setRound((r) => r + 1)}>{t("play.again")}</button>
          </div>
        )}
      </div>
    </GameShell>
  );
}
