import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import { ITEM_EMOJI } from "../features/home/demo";
import { useUI } from "../store/ui";
import { Panel } from "./Panel";

export function Backpack() {
  const { t } = useTranslation();
  const inv = useQuery({ queryKey: ["inventory"], queryFn: api.inventory, retry: false });
  const owned = useUI((s) => s.owned);
  const merged: Record<string, number> = { ...(inv.data?.items ?? {}) };
  for (const [id, n] of Object.entries(owned)) merged[id] = (merged[id] ?? 0) + n;
  const items = Object.entries(merged);
  return (
    <Panel title={`🎒 ${t("backpack.title")}`}>
      {items.length === 0 ? (
        <p className="text-sm text-muted">{t("backpack.empty")}</p>
      ) : (
        <ul className="grid grid-cols-4 gap-2">
          {items.map(([id, count]) => (
            <li key={id} className="relative grid aspect-square place-items-center rounded-xl bg-canvas text-2xl" title={t(`backpack.items.${id}`, { defaultValue: id })}>
              <span aria-hidden>{ITEM_EMOJI[id] ?? "🎁"}</span>
              <span className="absolute bottom-0.5 right-1.5 text-[10px] font-bold">x{count}</span>
              <span className="sr-only">{t(`backpack.items.${id}`, { defaultValue: id })}</span>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
