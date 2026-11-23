import type { CSSProperties } from "react";
import type { Action } from "../api/client";

// Стан улюбленця визначається показниками: найнижчий із ситості/енергії/чистоти нижче порогу, інакше радіє.
export type Feeling = "happy" | "hungry" | "sleepy" | "dirty";

const INK = "#3b2a2a";

const PARTICLES: Record<Action, string[]> = {
  feed: ["🍓", "🍰", "🍓"],
  play: ["✨", "🪶", "✨", "🪶"],
  wash: ["🫧", "🫧", "🫧", "🫧"],
  sleep: ["💤", "💤", "💤"],
};

type Eyes = "happy" | "dots" | "flat" | "closed";
type Mouth = "smile" | "flat" | "frown" | "open" | "small";

const STATE_EYES: Record<Feeling, Eyes> = { happy: "happy", hungry: "dots", sleepy: "flat", dirty: "dots" };
const STATE_MOUTH: Record<Feeling, Mouth> = { happy: "smile", hungry: "small", sleepy: "flat", dirty: "frown" };

export function PetFigure({ feeling, label, action, runId }: { feeling: Feeling; label: string; action?: Action; runId?: number }) {
  const eyes: Eyes = action === "sleep" ? "closed" : action ? "happy" : STATE_EYES[feeling];
  const mouth: Mouth = action === "feed" ? "open" : action === "sleep" ? "flat" : action ? "smile" : STATE_MOUTH[feeling];
  const dirty = feeling === "dirty" && action !== "wash";

  return (
    <div className="relative w-56">
      <div key={runId} className={`pet ${action ? `pet-${action}` : `pet-${feeling}`}`}>
        <svg viewBox="0 0 200 200" role="img" aria-label={label} className="w-56 drop-shadow-[0_14px_10px_rgba(120,60,40,.25)]">
          <path d="M50 84 L58 28 L98 58 Z" fill="#ff9f86" />
          <path d="M150 84 L142 28 L102 58 Z" fill="#ff9f86" />
          <ellipse cx="100" cy="122" rx="72" ry="64" fill="#ff9f86" />
          <ellipse cx="100" cy="142" rx="44" ry="38" fill="#fff1ec" />
          <circle cx="56" cy="124" r="12" fill="#ff7f9a" opacity=".5" />
          <circle cx="144" cy="124" r="12" fill="#ff7f9a" opacity=".5" />
          {dirty && (
            <g fill="#7a5a44" opacity=".45">
              <ellipse cx="70" cy="90" rx="9" ry="6" />
              <ellipse cx="136" cy="98" rx="11" ry="7" />
              <ellipse cx="112" cy="166" rx="10" ry="6" />
              <ellipse cx="82" cy="160" rx="7" ry="5" />
            </g>
          )}
          {eyes === "happy" && (
            <g fill="none" stroke={INK} strokeWidth="5" strokeLinecap="round">
              <path d="M68 110 q11 -15 22 0" />
              <path d="M110 110 q11 -15 22 0" />
            </g>
          )}
          {eyes === "closed" && (
            <g fill="none" stroke={INK} strokeWidth="5" strokeLinecap="round">
              <path d="M68 108 q11 11 22 0" />
              <path d="M110 108 q11 11 22 0" />
            </g>
          )}
          {eyes === "flat" && (
            <g fill="none" stroke={INK} strokeWidth="5" strokeLinecap="round">
              <path d="M70 110 h20" />
              <path d="M110 110 h20" />
            </g>
          )}
          {eyes === "dots" && (
            <g fill={INK}>
              <circle cx="80" cy="108" r="6" />
              <circle cx="120" cy="108" r="6" />
            </g>
          )}
          <ellipse cx="100" cy="125" rx="6" ry="4" fill={INK} />
          {mouth === "open" && <ellipse cx="100" cy="140" rx="10" ry="8" fill="#c4573a" />}
          {mouth === "small" && <ellipse cx="100" cy="139" rx="5" ry="6" fill={INK} />}
          {(mouth === "smile" || mouth === "flat" || mouth === "frown") && (
            <path
              d={mouth === "smile" ? "M86 134 q14 14 28 0" : mouth === "flat" ? "M92 137 h16" : "M90 142 q10 -10 20 0"}
              fill="none"
              stroke={INK}
              strokeWidth="4"
              strokeLinecap="round"
            />
          )}
        </svg>
      </div>

      {/* Постійні індикатори стану (коли немає анімації дії) */}
      {!action && feeling === "hungry" && <span aria-hidden className="state-icon text-3xl">🍓</span>}
      {!action && feeling === "sleepy" && (
        <>
          <span aria-hidden className="particle particle-loop text-2xl" style={{ left: "62%" }}>💤</span>
          <span aria-hidden className="particle particle-loop text-2xl" style={{ left: "72%", animationDelay: "1.2s" }}>💤</span>
        </>
      )}
      {!action && feeling === "dirty" && <span aria-hidden className="fly text-xl">🪰</span>}

      {action &&
        PARTICLES[action].map((emoji, i) => (
          <span
            key={`${runId}-${i}`}
            aria-hidden
            className="particle text-2xl"
            style={{ left: `${18 + i * 17}%`, animationDelay: `${i * 0.3}s`, "--dx": `${(i - 1.5) * 14}px` } as CSSProperties}
          >
            {emoji}
          </span>
        ))}
    </div>
  );
}