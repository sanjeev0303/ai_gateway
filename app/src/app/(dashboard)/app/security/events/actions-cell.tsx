"use client";

import { SecurityEventActions } from "./client";
import type { EventRow } from "./page";

export function EventActionsCell({ row }: { row: EventRow }) {
  return <SecurityEventActions event={row.rawEvent} />;
}