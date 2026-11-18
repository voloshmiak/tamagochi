import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

export function GameShell({ title, onBack, stats, children }: { title: string; onBack: () => void; stats?: ReactNode; children: ReactNode }) {
  const { t } = useTranslation();
  return (
    <div className="card flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button onClick={onBack} className="text-sm font-bold text-lilac-deep underline">← {t("play.back")}</button>
        <h2 className="text-xl">{title}</h2>
        <div className="flex gap-3 text-sm font-bold">{stats}</div>
      </div>
      {children}
    </div>
  );
}
