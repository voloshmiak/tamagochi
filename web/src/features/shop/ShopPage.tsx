import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Backpack } from "../../components/Backpack";
import { useUI } from "../../store/ui";
import { DEMO } from "../home/demo";

export function ShopPage() {
  const { t, i18n } = useTranslation();
  const { berries, buy } = useUI();
  const [msg, setMsg] = useState<{ text: string; ok: boolean }>();

  const onBuy = (id: string, price: number, name: string) =>
    setMsg(buy(id, price) ? { text: t("shop.bought", { name: t(name) }), ok: true } : { text: t("shop.notEnough"), ok: false });

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
      <section className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h1 className="text-3xl">{t("shop.title")}</h1>
          <span className="rounded-full bg-blush px-4 py-1.5 font-extrabold">🍓 {berries.toLocaleString(i18n.language)}</span>
        </div>
        <p role="status" className={`min-h-5 text-sm font-bold ${msg?.ok ? "text-mint-deep" : "text-coral-deep"}`}>{msg?.text}</p>
        <ul className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {DEMO.shop.map((g) => (
            <li key={g.id} className="card flex flex-col items-center gap-2 text-center">
              <span aria-hidden className="grid size-20 place-items-center rounded-full bg-blush text-4xl">{g.icon}</span>
              <h2 className="text-lg">{t(g.name)}</h2>
              <p className="rounded-full bg-cream px-3 py-0.5 text-sm font-extrabold text-coral-deep">{g.price} 🍓</p>
              <button onClick={() => onBuy(g.id, g.price, g.name)} className="chunky mt-1 bg-coral [--edge:#c4573a]">
                {t("shop.buy")}
              </button>
            </li>
          ))}
        </ul>
      </section>
      <aside className="flex flex-col gap-6">
        <Backpack />
      </aside>
    </div>
  );
}
