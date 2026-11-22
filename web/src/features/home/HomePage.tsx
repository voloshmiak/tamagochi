import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api, ApiError, type Action, type Pet, type Reaction, type Stats } from "../../api/client";
import { PetFigure, type Feeling } from "../../components/PetFigure";
import { Backpack } from "../../components/Backpack";
import { Panel } from "../../components/Panel";
import { useUI, today } from "../../store/ui";
import { DEMO } from "./demo";

const STAT_STYLE: Record<keyof Stats, { icon: string; bar: string }> = {
  satiety: { icon: "🍓", bar: "bg-coral" },
  energy: { icon: "🌙", bar: "bg-lilac-deep" },
  mood: { icon: "💖", bar: "bg-pink" },
  cleanliness: { icon: "🫧", bar: "bg-mint-deep" },
};

type Anim = { action: Action; id: number };

const ACTION_STYLE: { action: Action; icon: string; cls: string }[] = [
  { action: "feed", icon: "🍰", cls: "bg-coral [--edge:#c4573a]" },
  { action: "play", icon: "🪶", cls: "bg-mint [--edge:#23804a]" },
  { action: "wash", icon: "🫧", cls: "bg-lilac [--edge:#6a48c9]" },
  { action: "sleep", icon: "💤", cls: "bg-lilac-deep text-white [--edge:#3f2a85]" },
];

function feelingOf(stats: Stats): { feeling: Feeling; status: "great" | "ok" | "bad" } {
  const avg = (stats.satiety + stats.mood + stats.energy + stats.cleanliness) / 4;
  if (avg >= 75) return { feeling: "happy", status: "great" };
  if (avg >= 40) return { feeling: "ok", status: "ok" };
  return { feeling: "sad", status: "bad" };
}

export function HomePage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const progressQuest = useUI((s) => s.progressQuest);
  const [reaction, setReaction] = useState<Reaction>();
  const [anim, setAnim] = useState<Anim>();
  const animTimer = useRef<number>();
  useEffect(() => () => window.clearTimeout(animTimer.current), []);
  const pet = useQuery({
    queryKey: ["pet"],
    queryFn: api.myPet,
    retry: false,
    refetchInterval: (q) => (q.state.data && Object.values(q.state.data.cooldowns).some((s) => s > 0) ? 5000 : false),
  });
  const act = useMutation({
    mutationFn: api.act,
    onSuccess: (res, action) => {
      progressQuest(action);
      qc.setQueryData(["pet"], res.pet);
      setReaction(res.reaction);
      setAnim({ action, id: Date.now() });
      window.clearTimeout(animTimer.current);
      animTimer.current = window.setTimeout(() => setAnim(undefined), 2800);
      void qc.invalidateQueries({ queryKey: ["inventory"] });
    },
  });

  if (pet.isLoading) return null;
  if (pet.error instanceof ApiError && pet.error.status === 404) return <CreatePet />;
  if (!pet.data) return <p role="alert" className="card mx-auto max-w-md text-center">{t("pet.loadError")}</p>;

  return (
    <div className="grid gap-6 lg:grid-cols-[300px_minmax(0,1fr)_300px]">
      <StatusBar />
      <aside className="flex flex-col gap-6">
        <PetCard pet={pet.data} />
        <Vitals pet={pet.data} />
        <WeekChart />
      </aside>

      <section className="flex flex-col gap-4">
        <Room
          pet={pet.data}
          anim={anim}
          bubble={
            reaction
              ? t(`pet.behavior.${reaction.behavior}`, { name: pet.data.name, defaultValue: t("pet.idle", { name: pet.data.name }) })
              : undefined
          }
        />
        <div className="flex flex-wrap justify-center gap-2 rounded-full bg-white p-2 shadow-[0_8px_24px_-12px_rgba(106,72,201,.25)]">
          {ACTION_STYLE.map(({ action, icon, cls }) => {
            const wait = pet.data.cooldowns[action] ?? 0;
            return (
              <button key={action} disabled={act.isPending || wait > 0} onClick={() => act.mutate(action)} className={`chunky ${cls}`}>
                <span aria-hidden>{icon}</span>
                {t(`actions.${action}`)}
                {wait > 0 && <span className="text-xs font-semibold">({t("cooldown", { s: wait })})</span>}
              </button>
            );
          })}
        </div>
        {act.error instanceof ApiError && <p role="alert" className="text-center text-sm font-bold text-coral-deep">{act.error.message}</p>}
      </section>

      <aside className="flex flex-col gap-6">
        <Quests />
        <Backpack />
        <Decor />
      </aside>
    </div>
  );
}

function StatusBar() {
  const { t } = useTranslation();
  const chip = "rounded-full bg-white px-4 py-1.5 text-xs font-bold shadow-[0_4px_12px_-8px_rgba(106,72,201,.4)]";
  return (
    <div className="flex flex-wrap items-center gap-2 lg:col-span-3">
      <span className={chip}>🟢 {t("bar.room")}</span>
      <span className={`${chip} text-mint-deep`}>☀️ {t("bar.weather")}</span>
      <span className="ml-auto flex flex-wrap gap-2">
        <span className={`${chip} text-coral-deep`}>♡ {t("bar.bond")}</span>
        <span className={chip}>♪ {t("bar.music")}</span>
      </span>
    </div>
  );
}

function PetCard({ pet }: { pet: Pet }) {
  const { t } = useTranslation();
  const traits = Object.entries(pet.traits).sort((a, b) => b[1] - a[1]).slice(0, 3);
  return (
    <div className="card flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <span aria-hidden className="grid size-14 place-items-center rounded-full bg-blush text-3xl">🐱</span>
        <div>
          <h2 className="text-xl">{pet.name} 🐾</h2>
          <p className="text-sm text-muted">{t(`pet.appearance.${pet.appearance}`, { defaultValue: pet.appearance })}</p>
        </div>
      </div>
      {traits.length > 0 && (
        <ul aria-label={t("pet.traits")} className="flex flex-wrap gap-1.5">
          {traits.map(([k]) => (
            <li key={k} className="rounded-full bg-lilac/50 px-3 py-0.5 text-xs font-bold text-lilac-deep">
              {t(`pet.traitNames.${k}`, { defaultValue: k })}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function Vitals({ pet }: { pet: Pet }) {
  const { t } = useTranslation();
  const { status } = feelingOf(pet.stats);
  return (
    <div className="card flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl">{t("pet.health")}</h2>
        <span className="rounded-full bg-mint px-2.5 py-0.5 text-xs font-bold text-mint-deep">{t(`pet.status.${status}`)}</span>
      </div>
      {(Object.keys(STAT_STYLE) as (keyof Stats)[]).map((k) => (
        <div key={k}>
          <div className="mb-1 flex justify-between text-sm font-bold">
            <span><span aria-hidden>{STAT_STYLE[k].icon}</span> {t(`stats.${k}`)}</span>
            <span>{pet.stats[k]}%</span>
          </div>
          <div
            role="progressbar"
            aria-label={t(`stats.${k}`)}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={pet.stats[k]}
            className="h-3 overflow-hidden rounded-full bg-lilac/40"
          >
            <div className={`h-full rounded-full ${STAT_STYLE[k].bar}`} style={{ width: `${pet.stats[k]}%` }} />
          </div>
        </div>
      ))}
    </div>
  );
}

function WeekChart() {
  const { t } = useTranslation();
  const days = t("week.days", { returnObjects: true }) as string[];
  const avg = (DEMO.week.reduce((a, b) => a + b, 0) / DEMO.week.length).toFixed(1);
  return (
    <div className="card bg-lilac/30 shadow-none">
      <div className="mb-3 flex items-baseline justify-between text-sm font-bold">
        <h2 className="text-sm">{t("week.title")}</h2>
        <span className="text-mint-deep">{avg}%</span>
      </div>
      <div className="flex h-20 items-end gap-2">
        {DEMO.week.map((v, i) => (
          <div key={i} className="flex flex-1 flex-col items-center gap-1">
            <div className="w-full rounded-t-lg bg-coral/80" style={{ height: `${v * 0.6}px` }} />
            <span className="text-[10px] text-muted">{days[i]}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function Room({ pet, bubble, anim }: { pet: Pet; bubble?: string; anim?: Anim }) {
  const { t } = useTranslation();
  const { feeling } = feelingOf(pet.stats);
  return (
    <div className="relative h-[440px] overflow-hidden rounded-[2rem] shadow-[0_16px_32px_-16px_rgba(196,87,58,.5)]">
      <div className="absolute inset-0 bg-gradient-to-b from-[#ffe9df] to-[#ffd3c4]" />
      <div className="absolute inset-x-0 bottom-0 h-[34%] bg-[#e9cba9]" />
      <div aria-hidden className="absolute left-[8%] top-[12%] h-40 w-32 rounded-t-[4rem] border-[6px] border-white/80 bg-sky-100/80" />
      <div aria-hidden className="absolute bottom-[8%] left-1/2 h-24 w-[70%] -translate-x-1/2 rounded-[50%] bg-cream/90" />
      <div className="absolute left-1/2 top-[8%] max-w-[80%] -translate-x-1/2 rounded-2xl bg-white px-4 py-2 text-center font-serif text-sm font-bold shadow">
        ✨ {bubble ?? t("pet.idle", { name: pet.name })}
      </div>
      <div className="absolute inset-x-0 bottom-[10%] flex justify-center">
        <PetFigure feeling={feeling} label={pet.name} action={anim?.action} runId={anim?.id} />
      </div>
    </div>
  );
}

function Quests() {
  const { t } = useTranslation();
  const { questDay, questProgress, chestClaimed, claimChest } = useUI();
  const fresh = questDay === today();
  const count = (key: string) => (fresh ? (questProgress[key] ?? 0) : 0);
  const done = DEMO.quests.filter((q) => count(q.key) >= q.target).length;
  const allDone = done === DEMO.quests.length;
  const claimed = fresh && chestClaimed;
  return (
    <Panel title={`📋 ${t("quests.title")}`} aside={<span className="rounded-full bg-coral px-2.5 py-0.5 text-xs font-bold">{done} / {DEMO.quests.length}</span>}>
      <ul className="flex flex-col gap-2">
        {DEMO.quests.map((q) => {
          const finished = count(q.key) >= q.target;
          return (
            <li key={q.key} className="rounded-xl bg-canvas px-3 py-2 text-sm">
              <div className="flex justify-between font-bold">
                <span className={finished ? "line-through opacity-70" : ""}>{t(`quests.${q.key}`)}</span>
                <span className="text-coral-deep">+{q.reward} 🍓</span>
              </div>
              <div className="text-xs text-muted">{finished ? "✓" : `${count(q.key)} / ${q.target}`}</div>
            </li>
          );
        })}
        <li className="flex items-center justify-between rounded-xl bg-coral/70 px-3 py-2 text-sm font-bold">
          <span>
            🎁 {t("quests.chest")}
            <br />
            <span className="text-xs font-semibold opacity-80">
              {claimed ? t("quests.chestDone") : allDone ? t("quests.chestReady") : t("quests.chestHint")}
            </span>
          </span>
          <button disabled={!allDone || claimed} onClick={claimChest} className="chunky bg-white px-3 py-1 text-sm [--edge:rgba(0,0,0,.15)]">
            +{DEMO.chestReward}
          </button>
        </li>
      </ul>
    </Panel>
  );
}

function Decor() {
  const { t } = useTranslation();
  return (
    <div className="card bg-lilac/30 shadow-none">
      <h2 className="mb-3 text-base">🖌️ {t("decor.title")}</h2>
      <div className="flex flex-wrap gap-2">
        {DEMO.decor.map((k) => (
          <span key={k} className="rounded-xl bg-white px-3 py-1.5 text-xs font-bold">{t(`decor.${k}`)}</span>
        ))}
      </div>
    </div>
  );
}

function CreatePet() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const create = useMutation({
    mutationFn: () => api.createPet(name, "cat"),
    onSuccess: (pet) => qc.setQueryData(["pet"], pet),
  });
  return (
    <form
      className="card mx-auto flex max-w-sm flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate();
      }}
    >
      <h1 className="text-2xl">{t("pet.createTitle")}</h1>
      <input className="field" placeholder={t("pet.name")} value={name} minLength={2} maxLength={20} required onChange={(e) => setName(e.target.value)} />
      {create.error instanceof ApiError && <p role="alert" className="text-sm font-bold text-coral-deep">{create.error.message}</p>}
      <button type="submit" disabled={create.isPending} className="chunky justify-center bg-coral [--edge:#c4573a]">
        {t("pet.create")}
      </button>
    </form>
  );
}