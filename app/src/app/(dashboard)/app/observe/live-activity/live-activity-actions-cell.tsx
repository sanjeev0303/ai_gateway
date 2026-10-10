"use client";

import { LiveActivityDrawer } from "./live-activity-drawer";
import type { LiveRow } from "./live-activity-drawer";

export function LiveActivityActionsCell({ row }: { row: LiveRow }) {
  return <LiveActivityDrawer row={row} />;
}