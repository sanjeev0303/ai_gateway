import type { Metadata } from "next";
import Link from "next/link";
import { Activity, Building2, Coins, CreditCard, KeyRound, Users } from "lucide-react";

import type { AdminOverview, AdminServices } from "@ai-governance/api-client";
import { AreaChart } from "@ai-governance/console-ui/charts/area-chart";
import { chartColor } from "@ai-governance/console-ui/charts/utils";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { RangeTabs } from "@ai-governance/console-ui/dashboard/range-tabs";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { StatusBadge } from "@ai-governance/console-ui/dashboard/status-badge";
import { formatCompact, formatNumber, formatPercent, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import { Ledger, LedgerCell, Panel, Register, RegisterEmpty, RegisterRow } from "./_components/ledger";
import { actionLabel, formatBytes, serviceLabel, usd, WINDOW_LABEL, windowFrom, type SearchParams } from "./_lib";

export const metadata: Metadata = { title: "Platform overview · Tollseal" };

export default async function AdminOverviewPage({ searchParams }: { searchParams: SearchParams }) {
  const params = await searchParams;
  const window = windowFrom(params);

  let overview: AdminOverview;
  let services: AdminServices;
  try {
    const { admin } = await getApiClient();
    [overview, services] = await Promise.all([admin.overview(window), admin.services()]);
  } catch {
    return (
      <PageShell title="Platform overview">
        <ErrorState description="The control API did not answer. Check that it is running, then reload this page." />
      </PageShell>
    );
  }

  const { stats, outbox } = overview;
  const failureRate = stats.requests > 0 ? stats.failed_requests / stats.requests : 0;
  const traffic = overview.daily.map((day) => ({ date: day.day.slice(5), value: day.requests }));

  return (
    <PageShell
      title="Platform overview"
      description={`Every organization on this deployment, over the ${WINDOW_LABEL[window]}.`}
      actions={<RangeTabs />}
    >
      <div className="flex flex-col gap-6 pb-12">
        <Ledger title="Platform">
          <LedgerCell
            label="Organizations"
            value={formatNumber(stats.orgs_total)}
            detail={stats.orgs_suspended > 0 ? `${formatNumber(stats.orgs_suspended)} suspended` : "None suspended"}
            emptyDetail="No organization yet"
            icon={Building2}
            href="/organizations"
          />
          <LedgerCell
            label="Users"
            value={formatNumber(stats.users_total)}
            detail={stats.users_disabled > 0 ? `${formatNumber(stats.users_disabled)} disabled` : "None disabled"}
            emptyDetail="No one has signed up"
            icon={Users}
            href="/users"
          />
          <LedgerCell
            label="Requests"
            value={formatCompact(stats.requests)}
            detail={`${formatPercent(failureRate)} failed`}
            tone={failureRate >= 0.05 ? "warning" : "neutral"}
            icon={Activity}
            href={`/usage?range=${window}`}
          />
          <LedgerCell label="Model spend" value={usd(stats.cost_micros)} detail={`${formatCompact(stats.tokens)} tokens`} icon={Coins} href={`/usage?range=${window}`} />
          <LedgerCell
            label="Active gateway keys"
            value={formatNumber(stats.active_keys)}
            detail="Across all organizations"
            emptyDetail="No key issued yet"
            icon={KeyRound}
          />
          <LedgerCell
            label="Monthly revenue"
            value={usd(stats.mrr_micros)}
            detail={`${formatNumber(stats.paid_subscriptions)} paid subscriptions`}
            emptyDetail="No paid subscriptions"
            icon={CreditCard}
            href="/billing"
          />
        </Ledger>

        {overview.live_alerts > 0 && (
          <p role="alert" className="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-5 py-3 text-sm text-rose-800 dark:text-rose-300">
            {formatNumber(overview.live_alerts)} platform {overview.live_alerts === 1 ? "alert needs" : "alerts need"} attention.{" "}
            <Link href="/alerts" className="font-medium underline underline-offset-4">
              Open Alerts
            </Link>
          </p>
        )}
        {outbox.retrying > 0 && (
          <p role="status" className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-5 py-3 text-sm text-amber-800 dark:text-amber-300">
            {formatNumber(outbox.retrying)} background events are failing to write and being retried.{" "}
            <Link href="/services" className="font-medium underline underline-offset-4">
              Open Services
            </Link>
          </p>
        )}

        <div className="grid gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
          <Panel title="Requests per day" description="Every gateway request on the platform, by UTC day.">
            <div className="p-4">
              {traffic.length > 0 ? (
                <AreaChart label="Platform requests per day" series={[{ label: "Requests", color: chartColor(0), data: traffic }]} formatValue={formatCompact} />
              ) : (
                <RegisterEmpty>No gateway requests in this range.</RegisterEmpty>
              )}
            </div>
          </Panel>

          <Panel
            title="Services"
            description="The latest check of each part of the platform."
            action={
              <Link href="/services" className="text-sm font-medium text-primary underline-offset-4 hover:underline">
                Details
              </Link>
            }
          >
            {services.latest.length > 0 ? (
              <Register>
                {services.latest.map((s) => (
                  <RegisterRow
                    key={s.service}
                    primary={serviceLabel(s.service)}
                    secondary={
                      s.up
                        ? [s.cpu_percent !== undefined ? `CPU ${s.cpu_percent.toFixed(1)}%` : "", s.rss_bytes !== undefined ? `memory ${formatBytes(s.rss_bytes)}` : ""]
                            .filter(Boolean)
                            .join(", ") || `checked ${formatRelative(s.sampled_at)}`
                        : s.error || "Not answering"
                    }
                    value={<StatusBadge tone={s.up ? "success" : "danger"}>{s.up ? "Up" : "Down"}</StatusBadge>}
                  />
                ))}
              </Register>
            ) : (
              <RegisterEmpty>No checks yet. The worker records the first one within 30 seconds of starting.</RegisterEmpty>
            )}
          </Panel>
        </div>

        <div className="grid gap-5 lg:grid-cols-2">
          <Panel title="Top organizations by spend" description={`Model spend over the ${WINDOW_LABEL[window]}.`}>
            {overview.top_orgs.length > 0 ? (
              <Register>
                {overview.top_orgs.map((org) => (
                  <RegisterRow
                    key={org.key}
                    primary={org.label}
                    secondary={`${formatCompact(org.requests)} requests`}
                    value={usd(org.cost_micros)}
                    href={`/organizations/${org.key}`}
                  />
                ))}
              </Register>
            ) : (
              <RegisterEmpty>No organization has sent traffic in this range.</RegisterEmpty>
            )}
          </Panel>

          <Panel
            title="Recent platform actions"
            action={
              <Link href="/audit" className="text-sm font-medium text-primary underline-offset-4 hover:underline">
                Full audit log
              </Link>
            }
          >
            {overview.recent_actions.length > 0 ? (
              <Register>
                {overview.recent_actions.map((entry) => (
                  <RegisterRow
                    key={entry.id}
                    primary={entry.organization_name ? `${actionLabel(entry.action)}: ${entry.organization_name}` : actionLabel(entry.action)}
                    secondary={`${entry.actor_email}, ${formatRelative(entry.created_at)}`}
                  />
                ))}
              </Register>
            ) : (
              <RegisterEmpty>No platform admin has taken an action yet. Suspensions, plan changes and revocations will be listed here.</RegisterEmpty>
            )}
          </Panel>
        </div>
      </div>
    </PageShell>
  );
}
