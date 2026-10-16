"use client";

import * as React from "react";
import { Download, Printer, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import type { Invoice } from "@ai-governance/api-client";
import { formatDate, formatUsd, microsToUsd } from "@/lib/format";

export function ExportAllInvoicesButton({ invoices }: { invoices: Invoice[] }) {
  const handleExport = () => {
    if (invoices.length === 0) {
      toast.info("No invoices available to export.");
      return;
    }

    const header = "Invoice Number,Status,Currency,Total (USD),Period Start,Period End,Issued At\n";
    const rows = invoices
      .map((inv) =>
        [
          `"${inv.number}"`,
          inv.status,
          inv.currency.toUpperCase(),
          microsToUsd(inv.total_micros).toFixed(2),
          `"${formatDate(inv.period_start)}"`,
          `"${formatDate(inv.period_end)}"`,
          `"${formatDate(inv.issued_at)}"`,
        ].join(",")
      )
      .join("\n");

    const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `sentinel-invoices-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success("Invoices exported successfully.");
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport} className="gap-1.5 text-xs">
      <Download className="size-3.5" /> Export All (CSV)
    </Button>
  );
}

export function InvoiceActions({ invoice }: { invoice: Invoice }) {
  const handleDownload = () => {
    const header = "Line Item,Quantity,Unit Price (USD),Amount (USD)\n";
    const lines = invoice.lines
      .map((line) =>
        [
          `"${line.description.replace(/"/g, '""')}"`,
          line.quantity,
          microsToUsd(line.unit_micros).toFixed(4),
          microsToUsd(line.amount_micros).toFixed(2),
        ].join(",")
      )
      .join("\n");

    const totalRow = `\n"Total (${invoice.currency.toUpperCase()})",,,${microsToUsd(
      invoice.total_micros
    ).toFixed(2)}`;

    const content = `Invoice Number: ${invoice.number}\nPeriod: ${formatDate(
      invoice.period_start
    )} - ${formatDate(invoice.period_end)}\nIssued: ${formatDate(
      invoice.issued_at
    )}\nStatus: ${invoice.status.toUpperCase()}\n\n${header}${lines}${totalRow}`;

    const blob = new Blob([content], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `${invoice.number}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success(`Statement ${invoice.number} downloaded.`);
  };

  const handlePrint = () => {
    window.print();
  };

  return (
    <div className="flex items-center gap-2 print:hidden">
      <Button variant="outline" size="sm" onClick={handleDownload} className="gap-1 text-xs h-7">
        <Download className="size-3" /> Download CSV
      </Button>
      <Button variant="ghost" size="sm" onClick={handlePrint} className="gap-1 text-xs h-7">
        <Printer className="size-3" /> Print
      </Button>
    </div>
  );
}
