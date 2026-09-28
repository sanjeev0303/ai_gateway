import { UnavailablePanel } from "@/lib/metrics";

export type RangePreset = "today" | "7d" | "30d" | "custom";

export interface TimeRangeValue {
  readonly preset: RangePreset;
  readonly from: string;
  readonly to: string;
}

const PRESETS: readonly { readonly id: RangePreset; readonly label: string }[] = [
  { id: "today", label: "Today" },
  { id: "7d", label: "7 days" },
  { id: "30d", label: "30 days" },
  { id: "custom", label: "Custom" },
];

function isoDay(offsetDays: number): string {
  const day = new Date();
  day.setUTCHours(0, 0, 0, 0);
  day.setUTCDate(day.getUTCDate() - offsetDays);
  return day.toISOString().slice(0, 10);
}

export function presetRange(preset: RangePreset, current: TimeRangeValue): TimeRangeValue {
  if (preset === "custom") {
    return { ...current, preset };
  }
  const span = preset === "today" ? 0 : preset === "7d" ? 6 : 29;
  return { preset, from: isoDay(span), to: isoDay(0) };
}

export function defaultRange(): TimeRangeValue {
  return presetRange("7d", { preset: "7d", from: "", to: "" });
}

export function validateRange(range: TimeRangeValue): string | null {
  if (range.preset !== "custom") {
    return null;
  }
  if (range.from === "" || range.to === "") {
    return "Pick both a start and an end date.";
  }
  return range.from <= range.to ? null : "The start date must not come after the end date.";
}

export function rangeLabel(range: TimeRangeValue): string {
  const preset = PRESETS.find((p) => p.id === range.preset);
  return range.preset === "custom" || preset === undefined
    ? `${range.from} to ${range.to}`
    : preset.label;
}

export function rangeBounds(range: TimeRangeValue): { readonly from: string; readonly to: string } {
  return { from: `${range.from}T00:00:00Z`, to: `${range.to}T23:59:59.999Z` };
}

export function TimeRange({
  value,
  onChange,
}: {
  readonly value: TimeRangeValue;
  readonly onChange: (next: TimeRangeValue) => void;
}) {
  const error = validateRange(value);
  return (
    <div className="range">
      <div className="range-presets" role="group" aria-label="Time range">
        {PRESETS.map((preset) => (
          <button
            key={preset.id}
            type="button"
            className="btn btn-sm"
            aria-pressed={value.preset === preset.id}
            onClick={() => onChange(presetRange(preset.id, value))}
          >
            {preset.label}
          </button>
        ))}
      </div>
      {value.preset === "custom" && (
        <div className="range-custom">
          <input
            className="input"
            type="date"
            value={value.from}
            max={value.to === "" ? undefined : value.to}
            aria-label="Start date"
            onChange={(event) => onChange({ ...value, from: event.target.value })}
          />
          <span>to</span>
          <input
            className="input"
            type="date"
            value={value.to}
            min={value.from === "" ? undefined : value.from}
            aria-label="End date"
            onChange={(event) => onChange({ ...value, to: event.target.value })}
          />
        </div>
      )}
      {error !== null && <span className="field-error">{error}</span>}
    </div>
  );
}

export function ChartSlot({
  title,
  needs,
}: {
  readonly title: string;
  readonly needs: readonly string[];
}) {
  return (
    <div className="chart-frame">
      <h3 className="settings-row-title">{title}</h3>
      <UnavailablePanel what={title} needs={needs} />
    </div>
  );
}

export function RankingSlot({
  title,
  basis,
  needs,
}: {
  readonly title: string;
  readonly basis: string;
  readonly needs: readonly string[];
}) {
  return (
    <div className="chart-frame">
      <h3 className="settings-row-title">{title}</h3>
      <p className="dash-note">{basis} Volume only, not a quality judgement.</p>
      <UnavailablePanel what="This ranking" needs={needs} />
    </div>
  );
}
