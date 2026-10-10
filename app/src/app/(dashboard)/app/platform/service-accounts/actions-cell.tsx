"use client";

import { ServiceAccountActions } from "./client";
import type { ServiceAccountRow } from "./page";

export function ServiceAccountActionsCell({ row }: { row: ServiceAccountRow }) {
  return <ServiceAccountActions account={row} />;
}