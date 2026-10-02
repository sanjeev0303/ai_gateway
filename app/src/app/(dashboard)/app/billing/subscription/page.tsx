import type { Metadata } from "next";

import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { formatDate, formatUsd } from "@/lib/format";
import { billingSummary, organization } from "@/lib/mock-data";
import { PLAN_LABELS } from "@/types";

export const metadata: Metadata = {
  title: "Subscription · Sentinel",
  description: "The current plan, what is included, and how the recurring charge is composed.",
};

interface FeatureRow {
  id: string;
  feature: string;
  detail: string;
  included: string;
}

const COLUMNS: readonly Column<FeatureRow>[] = [
  { key: "feature", header: "Feature", primary: "feature", secondary: "detail", sort: "feature" },
  { key: "included", header: "Included in plan", primary: "included", align: "right" },
];

export default function SubscriptionPage() {
  const { allowance, paymentMethod, nextInvoiceDate } = billingSummary;
  const subtotal =
    billingSummary.monthlySubscription + billingSummary.meteredUsage;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Subscription"
        description="What the organization is on, what it costs, and when the next charge lands."
        actions={
          <Button size="sm" variant="outline">
            Change plan
          </Button>
        }
        eyebrow={
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge tone="success">{PLAN_LABELS[organization.plan]}</StatusBadge>
            <span className="text-xs text-muted-foreground">
              Renews {formatDate(nextInvoiceDate)}
            </span>
          </div>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Platform fee"
            value={formatUsd(billingSummary.monthlySubscription)}
            hint="Recurring, per month"
          />
          <StatCard
            label="Metered this cycle"
            value={formatUsd(billingSummary.meteredUsage)}
            tone="info"
            hint="Billed in arrears"
          />
          <StatCard
            label="Current cycle"
            value={formatUsd(subtotal)}
            hint="Subscription plus metered usage"
          />
          <StatCard
            label="Included allowance"
            value={formatUsd(allowance.included)}
            hint={`Hard cap ${allowance.hardCap === null ? "none" : formatUsd(allowance.hardCap)}`}
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Recurring charge</CardTitle>
              <CardDescription>How the current cycle is composed.</CardDescription>
            </CardHeader>
            <CardContent>
              <dl className="space-y-3 text-sm">
                <div className="flex items-center justify-between gap-4">
                  <dt className="text-muted-foreground">Platform subscription</dt>
                  <dd className="tabular-nums font-medium">
                    {formatUsd(billingSummary.monthlySubscription)}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <dt className="text-muted-foreground">Metered usage to date</dt>
                  <dd className="tabular-nums font-medium">
                    {formatUsd(billingSummary.meteredUsage)}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-4">
                  <dt className="text-muted-foreground">Overage above allowance</dt>
                  <dd className="tabular-nums font-medium">
                    {allowance.used > allowance.included
                      ? formatUsd((allowance.used - allowance.included) * allowance.overageRate)
                      : formatUsd(0)}
                  </dd>
                </div>
                <Separator />
                <div className="flex items-center justify-between gap-4">
                  <dt className="font-medium">Estimated total</dt>
                  <dd className="text-lg font-semibold tabular-nums">{formatUsd(subtotal)}</dd>
                </div>
              </dl>
            </CardContent>
            <CardFooter>
              <p className="text-xs text-muted-foreground">
                Final amount is fixed when the cycle closes on {formatDate(nextInvoiceDate)}.
              </p>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Payment method</CardTitle>
              <CardDescription>Charged automatically at the end of each cycle.</CardDescription>
            </CardHeader>
            <CardContent>
              {paymentMethod ? (
                <div className="space-y-4">
                  <div className="flex items-center gap-3 rounded-lg border p-4">
                    <Badge variant="outline" className="uppercase">
                      {paymentMethod.brand}
                    </Badge>
                    <div className="min-w-0 flex-1">
                      <p className="font-mono text-sm">•••• {paymentMethod.last4}</p>
                      <p className="text-xs text-muted-foreground">
                        Expires{" "}
                        {String(paymentMethod.expiresMonth).padStart(2, "0")}/
                        {paymentMethod.expiresYear}
                      </p>
                    </div>
                    {paymentMethod.isDefault && (
                      <StatusBadge tone="info">Default</StatusBadge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Billing contact: {organization.billingEmail}
                  </p>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No payment method on file.
                </p>
              )}
            </CardContent>
            <CardFooter className="gap-2">
              <Button size="sm" variant="outline">
                Update card
              </Button>
              <Button size="sm" variant="ghost">
                Billing portal
              </Button>
            </CardFooter>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>What the plan includes</CardTitle>
            <CardDescription>
              Entitlements are enforced at the gateway, not merely documented.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={[
                {
                  id: "f1",
                  feature: "Model access",
                  detail: "Registry models a project may call",
                  included: "All registered",
                },
                {
                  id: "f2",
                  feature: "Included usage",
                  detail: "Metered model usage before overage",
                  included: formatUsd(allowance.included),
                },
                {
                  id: "f3",
                  feature: "Seats",
                  detail: "Licensed members in the organization",
                  included: `${organization.seatsTotal} seats`,
                },
                {
                  id: "f4",
                  feature: "Retention",
                  detail: "How long request and trace data is kept",
                  included: "400 days",
                },
                {
                  id: "f5",
                  feature: "Audit log",
                  detail: "Immutable history of administrative changes",
                  included: "Unlimited",
                },
                {
                  id: "f6",
                  feature: "Data residency",
                  detail: "Region in which request data is stored",
                  included: organization.region,
                },
                {
                  id: "f7",
                  feature: "Support",
                  detail: "Response target for severity-one incidents",
                  included: "1 hour",
                },
              ] satisfies readonly FeatureRow[]}
              rowKey="id"
              emptyMessage="No entitlements defined."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
