"use client";

import * as React from "react";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import type { AuditRow } from "./page";

export function ExportAuditLogsButton({ rows }: { rows: AuditRow[] }) {
  const handleExport = () => {
    if (rows.length === 0) {
      toast.info("No audit logs to export.");
      return;
    }

    const header = "Action,Actor Type,Actor ID,Resource Type,Resource ID,Origin,Occurred At\n";
    const csvContent = rows
      .map((r) =>
        [
          `"${r.action}"`,
          `"${r.actor}"`,
          `"${r.actorId}"`,
          `"${r.resource}"`,
          `"${r.resourceId}"`,
          `"${r.origin}"`,
          `"${r.occurredAt}"`,
        ].join(",")
      )
      .join("\n");

    const blob = new Blob([header + csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `sentinel-audit-logs-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success("Audit logs exported to CSV.");
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport} className="gap-1.5 text-xs shadow-sm">
      <Download className="size-3.5" /> Export Audit Trail (CSV)
    </Button>
  );
}
