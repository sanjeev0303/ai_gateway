// No "log usage event" step: nothing on the request path calls the
// usage_events insert, so adding one would make this diagram false.
const STAGES = [
  "authn",
  "resolve policy snapshot",
  "authorize model / credential",
  "enforce rate + budget",
  "forward",
  "stream",
] as const;
export function RequestPathDiagram() {
  const boxH = 40;
  const gap = 10;
  const top = 8;
  const width = 320;
  const height = top + STAGES.length * (boxH + gap) + 34;

  return (
    <svg
      aria-hidden="true"
      role="presentation"
      viewBox={`0 0 ${width} ${height}`}
      width="100%"
      style={{ height: "auto", display: "block", maxWidth: 360 }}
    >
      {STAGES.map((label, i) => {
        const y = top + i * (boxH + gap);
        const last = i === STAGES.length - 1;
        return (
          <g key={label}>
            <rect
              x={0.75}
              y={y + 0.75}
              width={width - 1.5}
              height={boxH - 1.5}
              rx={6}
              fill="var(--card-bg)"
              stroke="var(--border)"
              strokeWidth={1.5}
            />
            <text
              x={14}
              y={y + boxH / 2 + 5}
              fontSize={14}
              fontFamily="var(--font-geist-sans), system-ui, sans-serif"
              fill={last ? "var(--accent)" : "var(--fg)"}
              fontWeight={500}
            >
              {label}
            </text>
            {!last && (
              <path
                d={`M ${width / 2} ${y + boxH} v ${gap - 2}`}
                stroke="var(--border)"
                strokeWidth={1.5}
                fill="none"
              />
            )}
          </g>
        );
      })}
      <text
        x={0}
        y={top + STAGES.length * (boxH + gap) + 18}
        fontSize={14}
        fontFamily="var(--font-geist-sans), system-ui, sans-serif"
        fill="var(--muted)"
      >
        No database on the hot path.
      </text>
    </svg>
  );
}
