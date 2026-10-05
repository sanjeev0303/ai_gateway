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

export const metadata: Metadata = {
  title: "Invoices · Sentinel",
  description: "Invoices recorded for this organization, with their itemised lines.",
};

const INVOICE_TONE: Record<InvoiceStatus, StatusTone> = {
  open: "warning",
  void: "neutral",
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
            <TableCell>{line.description}</TableCell>
            <TableCell className="text-right tabular-nums">
              {formatNumber(line.quantity)}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatUsd(microsToUsd(line.unit_micros))}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatUsd(microsToUsd(line.amount_micros))}
            </TableCell>
          </TableRow>
        ))}
        <TableRow>
          <TableCell colSpan={3} className="text-right font-medium">
            Total ({invoice.currency})
          </TableCell>
          <TableCell className="text-right font-medium tabular-nums">
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
        description="Invoices recorded for this organization, with their itemised lines."
      >
        <Card>
          <CardHeader>
            <CardTitle>What an invoice here is</CardTitle>
            <CardDescription>
              A record of priced usage, not a request for payment.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              Each invoice is a row in this control plane holding a billing period and
              its itemised lines. No payment processor is integrated, so an invoice is
              only ever <span className="font-medium">open</span> or{" "}
              <span className="font-medium">void</span> — there is no paid state,
              no card, no charge, and no downloadable document.
            </p>
            <p>
              Figures are derived from recorded requests priced by the worker. See
              Billing → Usage for the underlying consumption.
            </p>
          </CardContent>
        </Card>

        {invoices.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <EmptyState
                title="No invoices yet"
                description="An invoice is only recorded once a billing period has been closed for this organization, so there is nothing to itemise. Recorded consumption is still available under Billing → Usage."
              />
            </CardContent>
          </Card>
        ) : (
          invoices.map((invoice) => (
            <Card key={invoice.id}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {invoice.number}
                  <StatusBadge tone={INVOICE_TONE[invoice.status]}>
                    {invoice.status}
                  </StatusBadge>
                </CardTitle>
                <CardDescription>
                  {formatDate(invoice.period_start)} – {formatDate(invoice.period_end)} ·{" "}
                  issued {formatDate(invoice.issued_at)}
                </CardDescription>
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