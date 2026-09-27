"use client";

import { useEffect, useId, useState } from "react";

import { MonitorIcon, MoonIcon, SunIcon } from "@/components/icons";

type Mode = "light" | "dark" | "system";

const STORAGE_KEY = "theme";
const MODES: readonly { value: Mode; label: string; Icon: typeof SunIcon }[] = [
  { value: "light", label: "Light", Icon: SunIcon },
  { value: "dark", label: "Dark", Icon: MoonIcon },
  { value: "system", label: "System", Icon: MonitorIcon },
];

function readStoredMode(): Mode {
  const stored = localStorage.getItem(STORAGE_KEY);
  return MODES.some((m) => m.value === stored) ? (stored as Mode) : "system";
}

function applyMode(mode: Mode): void {
  const dark =
    mode === "dark" ||
    (mode === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}

export function ThemeSwitch() {
  const group = useId();
  const [mode, setMode] = useState<Mode | null>(null);

  useEffect(() => {
    setMode(readStoredMode());
  }, []);

  useEffect(() => {
    if (mode !== "system") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyMode("system");
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [mode]);

  function choose(next: Mode) {
    setMode(next);
    localStorage.setItem(STORAGE_KEY, next);
    applyMode(next);
  }

  return (
    <div className="theme-switch" role="group" aria-label="Color theme">
      {MODES.map(({ value, label, Icon }) => (
        <label className="theme-opt" key={value}>
          <input
            className="theme-opt-input"
            type="radio"
            name={group}
            value={value}
            checked={mode === value}
            onChange={() => choose(value)}
          />
          <span className="theme-opt-face">
            <Icon width={16} height={16} />
            <span className="sr-only">{label}</span>
          </span>
        </label>
      ))}
    </div>
  );
}
