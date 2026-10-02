"use client";

import * as React from "react";

export const MOBILE_BREAKPOINT = 768;

const mobileQuery = () =>
  window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);

function subscribe(onStoreChange: () => void) {
  const query = mobileQuery();
  query.addEventListener("change", onStoreChange);
  return () => query.removeEventListener("change", onStoreChange);
}

/**
 * Server renders `false` and the client snapshot arrives on the first commit,
 * so markup stays identical while resize updates are handled natively.
 */
export function useIsMobile(): boolean {
  return React.useSyncExternalStore(
    subscribe,
    () => mobileQuery().matches,
    () => false,
  );
}
