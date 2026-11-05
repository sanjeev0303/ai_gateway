import { cn } from "@/lib/utils";

/*
 * Long hairline curves behind a section: the path a request takes, drawn as
 * light catching thin engraved lines. Computed, so they render the same on the
 * server and in the browser. Decorative only; the homepage motion draws them in
 * with the scroll (they are measured as 1 so a dash of 1 hides the whole line).
 */

/** S-curves sweeping across the frame, each a little offset from the last. */
function sweepPaths(): string[] {
  const lines = [
    { y0: 120, y1: 640, lift: 260 },
    { y0: 170, y1: 690, lift: 260 },
    { y0: 520, y1: 160, lift: -220 },
    { y0: 560, y1: 210, lift: -220 },
    { y0: 820, y1: 420, lift: -160 },
  ];
  return lines.map(({ y0, y1, lift }) => `M-40 ${y0}C420 ${y0 + lift} 980 ${y1 - lift} 1480 ${y1}`);
}

/** Lines arriving from both sides and meeting at the centre, where the seal sits. */
function convergePaths(): string[] {
  const cx = 800;
  const cy = 112;
  const gap = 112;
  const starts = [8, 40, 72, 196, 216];
  const paths: string[] = [];
  for (const y0 of starts) {
    const yEnd = cy + (y0 - cy) * 0.12;
    paths.push(`M0 ${y0}C300 ${y0} ${cx - gap - 260} ${yEnd} ${cx - gap} ${yEnd}`);
    paths.push(`M1600 ${224 - y0}C1300 ${224 - y0} ${cx + gap + 260} ${224 - yEnd} ${cx + gap} ${224 - yEnd}`);
  }
  return paths;
}

const SWEEP = sweepPaths();
const CONVERGE = convergePaths();

export function FlowCurves({ variant = "sweep", className }: { variant?: "sweep" | "converge"; className?: string }) {
  const converge = variant === "converge";
  const paths = converge ? CONVERGE : SWEEP;
  return (
    <svg
      viewBox={converge ? "0 0 1600 224" : "0 0 1440 900"}
      preserveAspectRatio="xMidYMid slice"
      aria-hidden
      focusable="false"
      data-flow-curves={variant}
      className={cn("pointer-events-none absolute inset-0 size-full", className)}
      fill="none"
    >
      {/* The lines: thick and soft, like light along engraved grooves */}
      <g data-flow-lines>
        {paths.map((d, i) => (
          <path
            key={i}
            d={d}
            pathLength={1}
            stroke={i % 2 ? "var(--guilloche-soft)" : "var(--guilloche)"}
            strokeWidth={i % 2 ? 1.5 : 2.5}
            strokeLinecap="round"
          />
        ))}
      </g>
      {/* A short bright pulse on each line, moved along it by the scroll; hidden without motion */}
      <g data-flow-pulses>
        {paths.map((d, i) => (
          <path
            key={i}
            d={d}
            pathLength={1}
            stroke="var(--primary)"
            strokeOpacity={i % 2 ? 0.35 : 0.55}
            strokeWidth={i % 2 ? 2 : 3}
            strokeLinecap="round"
            strokeDasharray="0.07 1"
            strokeDashoffset={0.07}
          />
        ))}
      </g>
    </svg>
  );
}
