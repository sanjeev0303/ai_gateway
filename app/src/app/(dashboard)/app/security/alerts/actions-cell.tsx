"use client";

import { AlertActionsMenu } from "./client";
import type { AlertRow } from "./page";

export function AlertActionsCell({ row }: { row: AlertRow }) {
  return <AlertActionsMenu alert={row.rawAlert} />;
}