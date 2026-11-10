import type { Metadata } from "next";
import { CalendarRange, CreditCard, Landmark, Receipt, Users, Wallet } from "lucide-react";

import type { AdminBilling, AdminPage, AdminPayment, AdminRevenue } from "@ai-governance/api-client";
import { AreaChart } from "@ai-governance/console-ui/charts/area-chart";
import { chartColor } from "@ai-governance/console-ui/charts/utils";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatNumber, formatUsd, microsToUsd } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { Ledger, LedgerCell, Panel, Register, RegisterEmpty, RegisterRow } from "../_components/ledger";
import { hrefWith, PAGE_SIZE, pageFrom, usd, type SearchParams } from "../_lib";

export const metadata: Metadata = { title: "Earnings · Platform console" };

interface PaymentRow {
  id: string;
  paidAt: string;
  organization: string;
  href: string;
  invoice: string;
  amount: string;
  amountMicros: number;
  provider: string;
  reference: string;
}

const PAYMENT_COLUMNS: Column<PaymentRow>[] = [
  { key: "paid", header: "Paid", primary: "paidAt", format: "dateTime" },
  { key: "org", header: "Organization", primary: "organization", href: "href", grow: true },
  { key: "invoice", header: "Invoice", primary: "invoice" },
  { key: "provider", header: "Paid through", primary: "provider", secondary: "reference" },
  { key: "amount", header: "Amount", primary: "amount", sort: "amountMicros", align: "right" },
];

interface CustomerRow {
  id: string;
  organization: string;
  href: string;
  plan: string;
  lifetime: string;
  lifetimeMicros: number;
  payments: number;
  lastPaidAt: string;
}

const CUSTOMER_COLUMNS: Column<CustomerRow>[] = [
  { key: "org", header: "Organization", primary: "organization", href: "href", grow: true },
  { key: "plan", header: "Plan", primary: "plan" },
  { key: "payments", header: "Payments", primary: "payments", format: "number", align: "right" },
  { key: "lifetime", header: "Paid in total", primary: "lifetime", sort: "lifetimeMicros", align: "right" },
  { key: "last", header: "Last paid", primary: "lastPaidAt", format: "date" },
];

interface InvoiceRow {
  id: string;
  number: string;
  organization: string;
  href: string;
  status: string;
  statusTone: StatusTone;
  total: string;
  totalMicros: number;
  issuedAt: string;
}

const INVOICE_TONE: Record<string, StatusTone> = { paid: "success", open: "info", draft: "neutral", void: "neutral", uncollectible: "danger" };

const INVOICE_COLUMNS: Column<InvoiceRow>[] = [
  { key: "number", header: "Invoice", primary: "number" },
  { key: "org", header: "Organization", primary: "organization", href: "href", grow: true },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "total", header: "Total", primary: "total", sort: "totalMicros", align: "right" },
  { key: "issued", header: "Issued", primary: "issuedAt", format: "date" },
];

const PROVIDER_LABEL: Record<string, string> = { stripe: "Stripe", razorpay: "Razorpay" };

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export default async function EarningsPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const { page, offset } = pageFrom(params);

  let revenue: AdminRevenue;
  let payments: AdminPage<AdminPayment>;
  let billing: AdminBilling;
  try {
    const { admin } = await getApiClient();
    [revenue, payments, billing] = await Promise.all([admin.revenue(), admin.payments({ limit: PAGE_SIZE, offset }), admin.billing()]);
  } catch {
    return (
      <PageShell title="Earnings">
        <ErrorState description="Earnings could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const { summary } = revenue;
  let mrr = 0;
  let paidSubscriptions = 0;
  for (const plan of billing.plans) {
    mrr += plan.price_micros * plan.subscriptions;
    if (plan.price_micros > 0) paidSubscriptions += plan.subscriptions;
  }

  const monthly = revenue.monthly.map((m) => ({ date: m.month, value: microsToUsd(m.revenue_micros) }));
  const hasRevenue = revenue.monthly.some((m) => m.revenue_micros > 0);
  const hasMore = offset + payments.items.length < payments.total;

  return (
    <PageShell
      title="Earnings"
      description="Money actually taken: every paid invoice, from Stripe and Razorpay, across all organizations. Monthly recurring revenue is list price times live subscriptions."
    >
      <div className="flex flex-col gap-6 pb-12">
        <Ledger title="Earnings">
          <LedgerCell label="Earned in total" value={usd(summary.lifetime_micros)} detail={`${formatNumber(summary.payments)} payments`} emptyDetail="No payment yet" icon={Landmark} />
          <LedgerCell label="This month" value={usd(summary.this_month_micros)} detail="Since the 1st, UTC" emptyDetail="Nothing paid this month" icon={CalendarRange} />
          <LedgerCell label="Last 30 days" value={usd(summary.last_30d_micros)} detail="Rolling window" emptyDetail="Nothing paid in 30 days" icon={Wallet} />
          <LedgerCell label="Paying customers" value={formatNumber(summary.paying_orgs)} detail="Paid at least once" emptyDetail="No paying customer yet" icon={Users} />
          <LedgerCell label="Monthly recurring" value={usd(mrr)} detail="At list prices" emptyDetail="No paid subscription" icon={CreditCard} />
          <LedgerCell label="Paid subscriptions" value={formatNumber(paidSubscriptions)} detail="Active or trialing" emptyDetail="None yet" icon={Receipt} />
        </Ledger>

        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
          <Panel title="Earnings by month" description="Paid invoices in each of the last 12 months.">
            <div className="p-4">
              {hasRevenue ? (
                <AreaChart label="Earnings by month" series={[{ label: "Earned", color: chartColor(1), data: monthly }]} formatValue={formatUsd} />
              ) : (
                <RegisterEmpty>No payment in the last 12 months.</RegisterEmpty>
              )}
            </div>
          </Panel>
          <Panel title="Plans" description="Live subscriptions on each plan.">
            {billing.plans.length > 0 ? (
              <Register>
                {billing.plans.map((plan) => (
                  <RegisterRow
                    key={plan.plan_code}
                    primary={plan.plan_name}
                    secondary={plan.price_micros > 0 ? `${usd(plan.price_micros)} a month` : "Free"}
                    value={formatNumber(plan.subscriptions)}
                  />
                ))}
              </Register>
            ) : (
              <RegisterEmpty>No plan is defined.</RegisterEmpty>
            )}
          </Panel>
        </div>

        <Panel title="Payments" description={`${formatNumber(payments.total)} payments, newest first.`}>
          <div className="p-4">
            <DataTable
              columns={PAYMENT_COLUMNS}
              rows={payments.items.map((p) => ({
                id: p.id,
                paidAt: p.paid_at,
                organization: p.organization_name,
                href: `/organizations/${p.organization_id}`,
                invoice: p.number,
                amount: usd(p.total_micros),
                amountMicros: p.total_micros,
                provider: PROVIDER_LABEL[p.provider] ?? (p.provider || "—"),
                reference: p.payment_id,
              }))}
              rowKey="id"
              emptyMessage="No payment has been taken yet."
              serverPaginated
              currentPage={page - 1}
              hasMore={hasMore}
              prevHref={page > 1 ? hrefWith("/billing", params, { page: page - 1 }) : undefined}
              nextHref={hasMore ? hrefWith("/billing", params, { page: page + 1 }) : undefined}
            />
          </div>
        </Panel>

        <Panel title="Paying customers" description="Every organization that has paid, highest total first.">
          <div className="p-4">
            <DataTable
              columns={CUSTOMER_COLUMNS}
              rows={revenue.customers.map((c) => ({
                id: c.organization_id,
                organization: c.organization_name,
                href: `/organizations/${c.organization_id}`,
                plan: c.plan_name || "No plan",
                lifetime: usd(c.lifetime_micros),
                lifetimeMicros: c.lifetime_micros,
                payments: c.payments,
                lastPaidAt: c.last_paid_at,
              }))}
              rowKey="id"
              emptyMessage="No organization has paid yet."
            />
          </div>
        </Panel>

        <Panel title="All invoices" description="The 50 most recent of any status, including open and uncollectible ones.">
          <div className="p-4">
            <DataTable
              columns={INVOICE_COLUMNS}
              rows={billing.invoices.map((i) => ({
                id: i.id,
                number: i.number,
                organization: i.organization_name,
                href: `/organizations/${i.organization_id}`,
                status: capitalize(i.status),
                statusTone: INVOICE_TONE[i.status] ?? "neutral",
                total: usd(i.total_micros),
                totalMicros: i.total_micros,
                issuedAt: i.issued_at,
              }))}
              rowKey="id"
              emptyMessage="No invoice has been issued yet."
            />
          </div>
        </Panel>
      </div>
    </PageShell>
  );
}
