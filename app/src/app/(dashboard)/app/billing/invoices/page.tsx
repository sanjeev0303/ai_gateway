import type { Metadata } from "next";

import type { Invoice, InvoiceStatus } from "@ai-governance/api-client";

import { EmptyState, PageShell, StatusBadge } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatDate, formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { ExportAllInvoicesButton, InvoiceActions } from "./client";

export const metadata: Metadata = {
  title: "Invoices · Sentinel",
  description: "Invoices recorded for this organization, with their itemised lines.",
};

const INVOICE_TONE: Record<InvoiceStatus, StatusTone> = {
  open: "warning",
  void: "neutral",
  paid: "success",
};

async function loadInvoices(): Promise<Invoice[]> {
  const { billing } = await getApiClient();
  return (await billing.listInvoices()).items;
}

function InvoiceLines({ invoice }: { invoice: Invoice }) {
  if (invoice.lines.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        This invoice has no itemised lines.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Description</TableHead>
          <TableHead className="text-right">Quantity</TableHead>
          <TableHead className="text-right">Unit</TableHead>
          <TableHead className="text-right">Amount</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {invoice.lines.map((line) => (
          <TableRow key={line.id}>
            <TableCell className="font-medium">{line.description}</TableCell>
            <TableCell className="text-right tabular-nums">
              {formatNumber(line.quantity)}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatUsd(microsToUsd(line.unit_micros))}
            </TableCell>
            <TableCell className="text-right tabular-nums font-semibold">
              {formatUsd(microsToUsd(line.amount_micros))}
            </TableCell>
          </TableRow>
        ))}
        <TableRow className="border-t-2">
          <TableCell colSpan={3} className="text-right font-semibold">
            Total ({invoice.currency.toUpperCase()})
          </TableCell>
          <TableCell className="text-right font-bold text-base tabular-nums text-primary">
            {formatUsd(microsToUsd(invoice.total_micros))}
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  );
}

export default async function InvoicesPage() {
  const invoices = await loadInvoices();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Invoices"
        description="Invoices and billing statements recorded for this organization, with their itemised lines."
        actions={<ExportAllInvoicesButton invoices={invoices} />}
      >
        <Card>
          <CardHeader>
            <CardTitle>What an invoice here is</CardTitle>
            <CardDescription>
              A deterministic statement of recorded token usage and base tier fees.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              Each invoice is recorded per billing cycle with itemised line rows.
              Consumption figures are priced by the async worker from the active price sheet at inference time.
            </p>
            <p className="rounded-md border bg-muted/40 p-3 text-xs">
              <strong>Usage Accounting Scope:</strong> Invoices represent internal cost accounting and usage attribution for governance purposes. No external payment processor or financial settlement transactions are executed.
            </p>
          </CardContent>
        </Card>

        {invoices.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <EmptyState
                title="No invoices yet"
                description="An invoice is only recorded once a billing period has been closed for this organization. Recorded consumption is still available under Billing → Usage."
              />
            </CardContent>
          </Card>
        ) : (
          invoices.map((invoice) => (
            <Card key={invoice.id} className="shadow-sm">
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div>
                  <CardTitle className="flex items-center gap-2 text-lg">
                    <span>{invoice.number}</span>
                    <StatusBadge tone={INVOICE_TONE[invoice.status]}>
                      {invoice.status}
                    </StatusBadge>
                  </CardTitle>
                  <CardDescription className="mt-1">
                    {formatDate(invoice.period_start)} – {formatDate(invoice.period_end)} ·{" "}
                    issued {formatDate(invoice.issued_at)}
                    {invoice.payment_provider && (
                      <span className="ml-2 inline-flex items-center text-xs font-medium text-muted-foreground">
                        · Processed via {invoice.payment_provider.toUpperCase()}
                        {invoice.payment_id ? ` (${invoice.payment_id})` : ""}
                      </span>
                    )}
                  </CardDescription>
                </div>
                <InvoiceActions invoice={invoice} />
              </CardHeader>
              <CardContent>
                <InvoiceLines invoice={invoice} />
              </CardContent>
            </Card>
          ))
        )}
      </PageShell>
    </div>
  );
}