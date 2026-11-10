import { cn } from "../lib/utils";

/*
 * Guilloche: the engraved line patterns of banknotes and passports. These are
 * computed the way a guilloche lathe cuts them, so the curves are deterministic
 * and render the same on the server and in the browser. Decorative only.
 */

function closedPath(points: [number, number][]): string {
  const [first, ...rest] = points;
  if (!first) return "";
  let d = `M${first[0].toFixed(1)} ${first[1].toFixed(1)}`;
  for (const [x, y] of rest) d += `L${x.toFixed(1)} ${y.toFixed(1)}`;
  return d + "Z";
}

/** One rosette ring: a radius that waves n times around the circle, drawn in several phase-shifted copies. */
function ringPaths(base: number, amplitude: number, waves: number, copies: number, steps: number): string[] {
  const paths: string[] = [];
  for (let c = 0; c < copies; c++) {
    const phase = (c / copies) * ((2 * Math.PI) / waves);
    const points: [number, number][] = [];
    for (let i = 0; i < steps; i++) {
      const t = (i / steps) * 2 * Math.PI;
      const r = base + amplitude * Math.sin(waves * t + phase * waves);
      points.push([r * Math.cos(t), r * Math.sin(t)]);
    }
    paths.push(closedPath(points));
  }
  return paths;
}

const OUTER = ringPaths(84, 9, 18, 7, 216);
const INNER = ringPaths(58, 7, 12, 6, 168);

export function GuillocheRosette({ className }: { className?: string }) {
  return (
    <svg
      viewBox="-100 -100 200 200"
      aria-hidden
      focusable="false"
      data-guilloche
      className={cn("pointer-events-none", className)}
      fill="none"
    >
      <circle r="97" stroke="var(--guilloche)" strokeWidth="0.6" />
      <circle r="94.5" stroke="var(--guilloche-soft)" strokeWidth="0.4" />
      {OUTER.map((d, i) => (
        <path key={`o${i}`} d={d} pathLength={1} stroke="var(--guilloche)" strokeWidth="0.45" />
      ))}
      {INNER.map((d, i) => (
        <path key={`i${i}`} d={d} pathLength={1} stroke="var(--guilloche-soft)" strokeWidth="0.5" />
      ))}
      <circle r="44" stroke="var(--guilloche)" strokeWidth="0.5" />
    </svg>
  );
}

/** A horizontal band of interfering waves, for the edge of a note or a section rule. */
function bandPaths(width: number, lines: number, steps: number): string[] {
  const paths: string[] = [];
  for (let l = 0; l < lines; l++) {
    const phase = (l / lines) * Math.PI;
    let d = "";
    for (let i = 0; i <= steps; i++) {
      const x = (i / steps) * width;
      const y = 12 + 8 * Math.sin((x / width) * Math.PI * 14 + phase) * Math.cos((x / width) * Math.PI * 3 - phase / 2);
      d += `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`;
    }
    paths.push(d);
  }
  return paths;
}

const BAND = bandPaths(1200, 6, 240);

export function GuillocheBand({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 1200 24"
      preserveAspectRatio="none"
      aria-hidden
      focusable="false"
      data-guilloche
      className={cn("pointer-events-none h-6 w-full", className)}
      fill="none"
    >
      {BAND.map((d, i) => (
        <path key={i} d={d} pathLength={1} stroke={i % 2 ? "var(--guilloche-soft)" : "var(--guilloche)"} strokeWidth="0.8" vectorEffect="non-scaling-stroke" />
      ))}
    </svg>
  );
}
