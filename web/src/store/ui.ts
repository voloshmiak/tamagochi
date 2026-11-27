import { create } from "zustand";
import { persist } from "zustand/middleware";
import { DEMO } from "../features/home/demo";

export const today = () => new Date().toLocaleDateString("sv");

export type Tab = "room" | "shop" | "play" | "diary";
export interface Note {
  when: string;
  text: string;
}

interface UIState {
  tab: Tab;
  setTab: (tab: Tab) => void;
  sound: boolean;
  toggleSound: () => void;
  volume: number;
  setVolume: (v: number) => void;

  berries: number;
  addBerries: (n: number) => void;
  owned: Record<string, number>;
  buy: (id: string, price: number) => boolean;
  questDay: string;
  questProgress: Record<string, number>;
  chestClaimed: boolean;
  progressQuest: (key: string) => void;
  claimChest: () => void;
  notes: Note[];
  addNote: (text: string, when: string) => void;
  notifications: boolean;
  toggleNotifications: () => void;
  quiet: boolean;
  toggleQuiet: () => void;
  quietFrom: string;
  quietTo: string;
  setQuietHours: (from: string, to: string) => void;
}

export const useUI = create<UIState>()(
  persist(
    (set, get) => ({
      tab: "room",
      setTab: (tab) => set({ tab }),
      sound: true,
      toggleSound: () => set((s) => ({ sound: !s.sound })),
      volume: 0.15,
      setVolume: (volume) => set({ volume }),

      berries: DEMO.berries,
      addBerries: (n) => set((s) => ({ berries: s.berries + n })),
      owned: {},
      buy: (id, price) => {
        if (get().berries < price) return false;
        set((s) => ({ berries: s.berries - price, owned: { ...s.owned, [id]: (s.owned[id] ?? 0) + 1 } }));
        return true;
      },
      questDay: "",
      questProgress: {},
      chestClaimed: false,
      progressQuest: (key) =>
        set((s) => {
          const q = DEMO.quests.find((x) => x.key === key);
          if (!q) return s;
          const fresh = s.questDay === today();
          const progress = fresh ? s.questProgress : {};
          const before = progress[key] ?? 0;
          if (before >= q.target) return s;
          const after = before + 1;
          return {
            questDay: today(),
            questProgress: { ...progress, [key]: after },
            chestClaimed: fresh ? s.chestClaimed : false,
            berries: s.berries + (after === q.target ? q.reward : 0),
          };
        }),
      claimChest: () => set((s) => ({ berries: s.berries + DEMO.chestReward, chestClaimed: true })),
      notes: [],
      addNote: (text, when) => set((s) => ({ notes: [{ when, text }, ...s.notes] })),
      notifications: true,
      toggleNotifications: () => set((s) => ({ notifications: !s.notifications })),
      quiet: false,
      toggleQuiet: () => set((s) => ({ quiet: !s.quiet })),
      quietFrom: "22:00",
      quietTo: "08:00",
      setQuietHours: (quietFrom, quietTo) => set({ quietFrom, quietTo }),
    }),
    {
      name: "tamagochi-ui",
      version: 1,
      migrate: (persisted) => ({ ...(persisted as object), volume: 0.15 }) as UIState,
      partialize: (s) => ({
        sound: s.sound,
        volume: s.volume,
        berries: s.berries,
        owned: s.owned,
        questDay: s.questDay,
        questProgress: s.questProgress,
        chestClaimed: s.chestClaimed,
        notes: s.notes,
        notifications: s.notifications,
        quiet: s.quiet,
        quietFrom: s.quietFrom,
        quietTo: s.quietTo,
      }),
    },
  ),
);