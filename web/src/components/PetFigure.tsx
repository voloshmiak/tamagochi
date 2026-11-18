export type Feeling = "happy" | "ok" | "sad";

const INK = "#3b2a2a";

export function PetFigure({ feeling, label }: { feeling: Feeling; label: string }) {
  return (
    <svg viewBox="0 0 200 200" role="img" aria-label={label} className="w-56 drop-shadow-[0_14px_10px_rgba(120,60,40,.25)]">
      <path d="M50 84 L58 28 L98 58 Z" fill="#ff9f86" />
      <path d="M150 84 L142 28 L102 58 Z" fill="#ff9f86" />
      <ellipse cx="100" cy="122" rx="72" ry="64" fill="#ff9f86" />
      <ellipse cx="100" cy="142" rx="44" ry="38" fill="#fff1ec" />
      <circle cx="56" cy="124" r="12" fill="#ff7f9a" opacity=".5" />
      <circle cx="144" cy="124" r="12" fill="#ff7f9a" opacity=".5" />
      {feeling === "happy" && (
        <g fill="none" stroke={INK} strokeWidth="5" strokeLinecap="round">
          <path d="M68 110 q11 -15 22 0" />
          <path d="M110 110 q11 -15 22 0" />
        </g>
      )}
      {feeling !== "happy" && (
        <g fill={INK}>
          <circle cx="80" cy="108" r="6" />
          <circle cx="120" cy="108" r="6" />
        </g>
      )}
      <ellipse cx="100" cy="125" rx="6" ry="4" fill={INK} />
      <path
        d={feeling === "happy" ? "M86 134 q14 14 28 0" : feeling === "ok" ? "M92 137 h16" : "M90 142 q10 -10 20 0"}
        fill="none"
        stroke={INK}
        strokeWidth="4"
        strokeLinecap="round"
      />
    </svg>
  );
}
