import { create } from "zustand";
import { persist } from "zustand/middleware";
import { DEMO } from "../features/home/demo";

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

  // Local demo state: shop, mini-games, diary and settings have no backend yet.
  berries: number;
  addBerries: (n: number) => void;
  owned: Record<string, number>;
  buy: (id: string, price: number) => boolean;
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

      berries: DEMO.berries,
      addBerries: (n) => set((s) => ({ berries: s.berries + n })),
      owned: {},
      buy: (id, price) => {
        if (get().berries < price) return false;
        set((s) => ({ berries: s.berries - price, owned: { ...s.owned, [id]: (s.owned[id] ?? 0) + 1 } }));
        return true;
      },
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
      partialize: (s) => ({
        sound: s.sound,
        berries: s.berries,
        owned: s.owned,
        notes: s.notes,
        notifications: s.notifications,
        quiet: s.quiet,
        quietFrom: s.quietFrom,
        quietTo: s.quietTo,
      }),
    },
  ),
);
