import type { ReactNode } from "react";

export function Panel({ title, aside, children }: { title: string; aside?: ReactNode; children: ReactNode }) {
  return (
    <div className="card flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="text-xl">{title}</h2>
        {aside}
      </div>
      {children}
    </div>
  );
}
