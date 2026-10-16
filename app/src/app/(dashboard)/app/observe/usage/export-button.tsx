"use client";

import * as React from "react";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { formatNumber, microsToUsd } from "@/lib/format";

export function ExportUsageButton({ summary }: { summary: any }) {
  const handleExport = () => {
    const header = "Metric,Value\n";
    const rows = [
      ["Total Requests", summary.Requests ?? 0],
      ["Input Tokens", summary.InputTokens ?? 0],
      ["Output Tokens", summary.OutputTokens ?? 0],
      ["Cached Input Tokens", summary.CachedTokens ?? 0],
      ["Total Tokens", summary.TotalTokens ?? 0],
      ["Estimated Spend (USD)", `$${microsToUsd(summary.TotalCostMicros ?? 0).toFixed(4)}`],
      ["Error Count", summary.Errors ?? 0],
      ["Average Latency (ms)", `${summary.AvgLatencyMs ?? 0}ms`],
      ["Active Employees", summary.ActiveUsers ?? 0],
    ]
      .map(([k, v]) => `"${k}","${v}"`)
      .join("\n");

    const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `sentinel-usage-summary-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success("Usage summary exported to CSV.");
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport} className="gap-1.5 text-xs shadow-sm">
      <Download className="size-3.5" /> Export (CSV)
    </Button>
  );
}
