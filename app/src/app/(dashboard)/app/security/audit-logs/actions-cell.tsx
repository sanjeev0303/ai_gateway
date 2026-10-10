"use client";

import { AuditLogActions } from "./client";
import type { AuditRow } from "./page";

export function AuditActionsCell({ row }: { row: AuditRow }) {
  return <AuditLogActions record={row.rawAlert} />;
}