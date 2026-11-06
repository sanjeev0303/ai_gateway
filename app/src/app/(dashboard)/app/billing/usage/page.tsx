import type { Metadata } from "next";
import Link from "next/link";

import { PageShell, StatCard } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatUsd, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import { ExportBillingUsageButton } from "./export-button";

export const metadata: Metadata = {
  title: "Usage · Tollseal",
  description: "Recorded metered spend for the trailing 30 days, with plan context.",
};

const DAY = 24 * 60 * 60 * 1000;

async function loadBillingUsage() {
  const { billing, organization, usage } = await getApiClient();
  const window = { from: new Date(Date.now() - 30 * DAY) };

  const [org, summary, invoiceList] = await Promise.all([
    organization.get(),
    usage.summary(window),
    billing.listInvoices(),
  ]);

  return {
    organization: org,
    summary,
    invoices: invoiceList.items,
  };
}

export default async function BillingUsagePage() {
  const { organization, summary, invoices } = await loadBillingUsage();

  const totalSpend = microsToUsd(summary.TotalCostMicros);

  const planInfo = organization.plan;
  const billingInfo = organization.billing;
  const periodParts = [
    billingInfo?.current_period_start
      ? formatDate(billingInfo.current_period_start)
      : null,
    billingInfo?.current_period_end ? formatDate(billingInfo.current_period_end) : null,
  ].filter((part): part is string => Boolean(part));

  const subscriptionLine = [
    planInfo ? `${planInfo.name} plan` : "No plan on record",
    billingInfo?.status ?? null,
    periodParts.length === 2 ? periodParts.join(" – ") : null,
    billingInfo?.cancel_at_period_end ? "cancels at period end" : null,
  ]
    .filter((part): part is string => Boolean(part))
    .join(" · ");

  const openInvoices = invoices.filter((invoice) => invoice.status === "open").length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="Metered consumption recorded against this organization over the trailing 30 days."
        eyebrow={<p className="text-xs text-muted-foreground">{subscriptionLine}</p>}
        actions={<ExportBillingUsageButton summary={summary} />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Recorded cost"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Allowance and invoicing</CardTitle>
            <CardDescription>
              What the control plane records, and what it does not.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              A plan carries a name, a code, and a status — no quota, no included
              allowance, and no overage rate is stored — so there is no allowance meter
              or overage projection to derive here. Treat the figure above as recorded
              consumption, not as an amount billed.
            </p>
            <p>
              {invoices.length === 0
                ? "No invoice has been recorded for this organization yet."
                : `${invoices.length} invoice${invoices.length === 1 ? "" : "s"} recorded, ${openInvoices} still open.`}{" "}
              An invoice is a record of priced usage with itemised lines, not a request
              for payment: no payment processor is integrated, so there is no card on
              file, no charge, and no paid state. Billing → Invoices has the detail.
            </p>
          </CardContent>
        </Card>

        <p className="text-sm">
          <Link
            href="/app/observe/costs"
            className="font-medium text-primary underline-offset-4 hover:underline"
          >
            Daily spend and breakdowns are on Costs
          </Link>
        </p>
      </PageShell>
    </div>
  );
}
