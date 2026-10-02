"use client";

import * as React from "react";

export type Theme = "light" | "dark" | "system";

const STORAGE_KEY = "sentinel-theme";
const DARK_QUERY = "(prefers-color-scheme: dark)";

interface UseThemeResult {
  theme: Theme;
  resolvedTheme: "light" | "dark";
  setTheme: (theme: Theme) => void;
}

const subscribers = new Set<() => void>();

const emit = () => {
  for (const notify of subscribers) notify();
};

const readTheme = (): Theme => {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return stored === "light" || stored === "dark" || stored === "system"
      ? stored
      : "system";
  } catch {
    // Private browsing modes can throw on localStorage access.
    return "system";
  }
};

const prefersDark = (): boolean => window.matchMedia(DARK_QUERY).matches;

const subscribeToSystem = (onStoreChange: () => void) => {
  const query = window.matchMedia(DARK_QUERY);
  query.addEventListener("change", onStoreChange);
  return () => query.removeEventListener("change", onStoreChange);
};

const subscribe = (onStoreChange: () => void) => {
  subscribers.add(onStoreChange);
  const unsubscribeSystem = subscribeToSystem(onStoreChange);
  return () => {
    subscribers.delete(onStoreChange);
    unsubscribeSystem();
  };
};

export function useTheme(): UseThemeResult {
  const theme = React.useSyncExternalStore<Theme>(
    subscribe,
    readTheme,
    () => "system",
  );
  const systemDark = React.useSyncExternalStore(
    subscribeToSystem,
    prefersDark,
    () => false,
  );

  const resolvedTheme =
    theme === "system" ? (systemDark ? "dark" : "light") : theme;

  React.useEffect(() => {
    document.documentElement.classList.toggle("dark", resolvedTheme === "dark");
  }, [resolvedTheme]);

  const setTheme = React.useCallback((next: Theme) => {
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Persisting is best-effort; the in-memory value still applies.
    }
    emit();
  }, []);

  return { theme, resolvedTheme, setTheme };
}
