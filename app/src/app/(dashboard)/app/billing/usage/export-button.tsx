"use client";

import * as React from "react";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { microsToUsd } from "@/lib/format";

export function ExportBillingUsageButton({ summary }: { summary: any }) {
  const handleExport = () => {
    const header = "Billing Metric,Recorded Value\n";
    const rows = [
      ["Billing Window", "Trailing 30 Days"],
      ["Total Requests Metered", summary.Requests ?? 0],
      ["Prompt / Input Tokens", summary.InputTokens ?? 0],
      ["Completion / Output Tokens", summary.OutputTokens ?? 0],
      ["Cached Input Tokens", summary.CachedTokens ?? 0],
      ["Total Billed Tokens", summary.TotalTokens ?? 0],
      ["Total Metered Cost (USD)", `$${microsToUsd(summary.TotalCostMicros ?? 0).toFixed(4)}`],
      ["Error Rate / Failed Requests", summary.Errors ?? 0],
      ["Active Workload Subjects", summary.ActiveUsers ?? 0],
    ]
      .map(([k, v]) => `"${k}","${v}"`)
      .join("\n");

    const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `sentinel-metered-billing-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success("Billing usage statement exported.");
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport} className="gap-1.5 text-xs shadow-sm">
      <Download className="size-3.5" /> Export Billing Statement (CSV)
    </Button>
  );
}
