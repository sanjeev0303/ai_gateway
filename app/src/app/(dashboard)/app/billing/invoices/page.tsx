import type { Metadata } from "next";
import { Download } from "lucide-react";

import { BarChart } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatUsd } from "@/lib/format";
import { invoices } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Invoices · Sentinel",
  description: "Statements for every completed billing period, with line-item detail.",
};

const INVOICE_STATUS_TONE: Record<string, StatusTone> = {
  paid: "success",
  open: "info",
  past_due: "danger",
  void: "neutral",
};

interface InvoiceRow {
  id: string;
  number: string;
  issued: string;
  issuedTs: number;
  due: string;
  status: string;
  statusTone: StatusTone;
  subtotal: number;
  tax: number;
  total: number;
  lines: number;
}

const COLUMNS: readonly Column<InvoiceRow>[] = [
  { key: "number", header: "Invoice", primary: "number", sort: "number" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "issued", header: "Issued", primary: "issued", align: "right", sort: "issuedTs" },
  { key: "due", header: "Due", primary: "due", align: "right" },
  { key: "lines", header: "Lines", primary: "lines", align: "right" },
  { key: "subtotal", header: "Subtotal", primary: "subtotal", format: "usd", align: "right" },
  { key: "tax", header: "Tax", primary: "tax", format: "usd", align: "right" },
  { key: "total", header: "Total", primary: "total", format: "usd", align: "right", sort: "total" },
];

export default function InvoicesPage() {
  const rows: InvoiceRow[] = invoices.map((invoice) => ({
    id: invoice.id,
    number: invoice.number,
    issued: formatDate(invoice.issuedAt),
    due: formatDate(invoice.dueAt),
    issuedTs: Date.parse(invoice.issuedAt),
    status: invoice.status,
    statusTone: INVOICE_STATUS_TONE[invoice.status] ?? "neutral",
    subtotal: invoice.subtotal,
    tax: invoice.tax,
    total: invoice.total,
    lines: invoice.lines.length,
  }));

  const open = invoices.filter((invoice) => invoice.status === "open");
  const paidTotal = invoices
    .filter((invoice) => invoice.status === "paid")
    .reduce((sum, invoice) => sum + invoice.total, 0);
  const average =
    invoices.reduce((sum, invoice) => sum + invoice.total, 0) / invoices.length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Invoices"
        description="A statement per billing period. The current cycle remains open until it closes."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Invoices" value={String(invoices.length)} hint="Most recent periods" />
          <StatCard
            label="Open"
            value={String(open.length)}
            tone={open.length > 0 ? "info" : "neutral"}
            hint={open[0] ? `${open[0].number} is still accruing` : "Nothing outstanding"}
          />
          <StatCard
            label="Paid to date"
            value={formatUsd(paidTotal)}
            tone="success"
            hint="Across settled periods"
          />
          <StatCard
            label="Average invoice"
            value={formatUsd(average)}
            hint="Platform fee plus metered usage"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All invoices</CardTitle>
            <CardDescription>
              Statements are immutable once issued; corrections appear as credit notes.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              toolbar={
                <div className="flex flex-wrap items-center gap-2">
                  {Object.entries(INVOICE_STATUS_TONE)
                    .filter(([status]) => invoices.some((invoice) => invoice.status === status))
                    .map(([status, tone]) => (
                      <StatusBadge key={status} tone={tone}>
                        {invoices.filter((invoice) => invoice.status === status).length} {status}
                      </StatusBadge>
                    ))}
                </div>
              }
              emptyMessage="No invoices issued yet."
            />
          </CardContent>
        </Card>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Invoice total by period</CardTitle>
              <CardDescription>What each billing period actually charged.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={[...invoices]
                  .sort((a, b) => a.issuedAt.localeCompare(b.issuedAt))
                  .map((invoice) => ({ label: invoice.number, value: invoice.total }))}
                formatValue={(value) => formatUsd(value)}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Line items</CardTitle>
              <CardDescription>
                The two components that make up every statement.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {invoices.slice(0, 2).map((invoice) => (
                <div key={invoice.id} className="rounded-lg border">
                  <div className="flex items-center justify-between gap-3 border-b p-3">
                    <p className="font-mono text-sm font-medium">{invoice.number}</p>
                    <StatusBadge tone={INVOICE_STATUS_TONE[invoice.status] ?? "neutral"}>
                      {invoice.status}
                    </StatusBadge>
                  </div>
                  <ul className="divide-y text-sm">
                    {invoice.lines.map((line) => (
                      <li key={line.id} className="flex items-center justify-between gap-4 p-3">
                        <div className="min-w-0">
                          <p className="truncate">{line.description}</p>
                          <p className="text-xs text-muted-foreground">
                            {line.quantity} × {formatUsd(line.unitPrice)}
                          </p>
                        </div>
                        <span className="tabular-nums font-medium">{formatUsd(line.amount)}</span>
                      </li>
                    ))}
                    <li className="flex items-center justify-between gap-4 p-3">
                      <span className="text-muted-foreground">Tax</span>
                      <span className="tabular-nums">{formatUsd(invoice.tax)}</span>
                    </li>
                    <li className="flex items-center justify-between gap-4 p-3">
                      <span className="font-medium">Total</span>
                      <span className="tabular-nums font-semibold">{formatUsd(invoice.total)}</span>
                    </li>
                  </ul>
                  <div className="border-t p-3">
                    <Button size="sm" variant="outline">
                      <Download aria-hidden /> Download PDF
                    </Button>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </PageShell>
    </div>
  );
}
