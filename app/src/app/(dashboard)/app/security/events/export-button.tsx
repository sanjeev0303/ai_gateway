"use client";

import * as React from "react";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import type { EventRow } from "./page";

export function ExportEventsButton({ rows }: { rows: EventRow[] }) {
  const handleExport = () => {
    if (rows.length === 0) {
      toast.info("No security events to export.");
      return;
    }

    const header = "Title,Category,Severity,Source,Detail,Occurred At\n";
    const csvContent = rows
      .map((r) =>
        [
          `"${r.title.replace(/"/g, '""')}"`,
          `"${r.category}"`,
          `"${r.severity}"`,
          `"${r.source}"`,
          `"${r.detail.replace(/"/g, '""')}"`,
          `"${r.occurredAt}"`,
        ].join(",")
      )
      .join("\n");

    const blob = new Blob([header + csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `sentinel-security-events-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success("Security events exported to CSV.");
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport} className="gap-1.5 text-xs shadow-sm">
      <Download className="size-3.5" /> Export Events (CSV)
    </Button>
  );
}
