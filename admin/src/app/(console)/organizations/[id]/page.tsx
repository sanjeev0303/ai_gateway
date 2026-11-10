import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Activity, BadgeCheck, Coins, CreditCard, KeyRound, Users } from "lucide-react";

import { ApiError, type AdminOrganizationDetail, type Plan } from "@ai-governance/api-client";
import { DataTable, type Column } from "@ai-governance/console-ui/dashboard/data-table";
import { PageShell } from "@ai-governance/console-ui/dashboard/page-shell";
import { RangeTabs } from "@ai-governance/console-ui/dashboard/range-tabs";
import { ErrorState } from "@ai-governance/console-ui/dashboard/state";
import { formatCompact, formatDate, formatNumber, formatRelative } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@ai-governance/console-ui/types/common";
import { Ledger, LedgerCell, Panel } from "../../_components/ledger";
import { usd, WINDOW_LABEL, windowFrom, type SearchParams } from "../../_lib";
import { OrgActions } from "./org-actions";

export const metadata: Metadata = { title: "Organization · Platform console" };

interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  statusTone: StatusTone;
  lastLogin: string;
  joinedAt: string;
}

const MEMBER_COLUMNS: Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", grow: true },
  { key: "role", header: "Role", primary: "role" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "login", header: "Last sign-in", primary: "lastLogin" },
  { key: "joined", header: "Joined", primary: "joinedAt", format: "date" },
];

interface KeyRow {
  id: string;
  name: string;
  prefix: string;
  subject: string;
  status: string;
  statusTone: StatusTone;
  lastUsed: string;
  createdAt: string;
}

const KEY_COLUMNS: Column<KeyRow>[] = [
  { key: "name", header: "Key", primary: "name", secondary: "prefix", grow: true },
  { key: "subject", header: "Issued to", primary: "subject" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "used", header: "Last used", primary: "lastUsed" },
  { key: "created", header: "Created", primary: "createdAt", format: "date" },
];

const KEY_TONE: Record<string, StatusTone> = { active: "success", revoked: "danger", expired: "neutral" };

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export default async function AdminOrganizationPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: SearchParams;
}) {
  const { id } = await params;
  const window = windowFrom(await searchParams);

  let detail: AdminOrganizationDetail;
  let plans: Plan[];
  try {
    const { admin } = await getApiClient();
    [detail, { items: plans }] = await Promise.all([admin.organization(id, window), admin.plans()]);
  } catch (err) {
    if (err instanceof ApiError && (err.status === 404 || err.status === 400)) notFound();
    return (
      <PageShell title="Organization">
        <ErrorState description="This organization could not be loaded. Check that the control API is running, then reload." />
      </PageShell>
    );
  }

  const org = detail.organization;
  const suspended = org.status === "suspended";

  const members: MemberRow[] = detail.members.map((m) => ({
    id: m.user_id,
    name: m.display_name || m.email,
    email: m.email,
    role: capitalize(m.role.replace("_", " ")),
    status: m.status === "disabled" ? "Disabled" : "Active",
    statusTone: m.status === "disabled" ? "danger" : "success",
    lastLogin: m.last_login_at ? formatRelative(m.last_login_at) : "Never",
    joinedAt: m.joined_at,
  }));

  const keys: KeyRow[] = detail.keys.map((k) => ({
    id: k.id,
    name: k.name,
    prefix: `${k.key_prefix}…`,
    subject: k.subject_type === "service_account" ? "Service account" : "User",
    status: capitalize(k.status),
    statusTone: KEY_TONE[k.status] ?? "neutral",
    lastUsed: k.last_used_at ? formatRelative(k.last_used_at) : "Never",
    createdAt: k.created_at,
  }));

  return (
    <PageShell
      title={org.name}
      description={`Slug ${org.slug}. Created ${formatDate(org.created_at)}. Traffic covers the ${WINDOW_LABEL[window]}.`}
      eyebrow={
        <Link href="/organizations" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
          All organizations
        </Link>
      }
      actions={
        <OrgActions
          orgId={org.id}
          orgName={org.name}
          suspended={suspended}
          currentPlan={org.plan_code}
          activeKeys={org.active_keys}
          plans={plans.filter((p) => p.status === "active").map((p) => ({ code: p.code, name: p.name }))}
        />
      }
    >
      <div className="flex flex-col gap-6 pb-12">
        <Ledger title="Organization" aside={<RangeTabs />}>
          <LedgerCell
            label="Status"
            value={suspended ? "Suspended" : "Active"}
            detail={suspended ? "Keys and console refused" : "Serving traffic"}
            tone={suspended ? "danger" : "success"}
            icon={BadgeCheck}
          />
          <LedgerCell
            label="Plan"
            value={org.plan_name || "None"}
            detail={org.subscription_status ? capitalize(org.subscription_status.replace("_", " ")) : "No subscription"}
            icon={CreditCard}
          />
          <LedgerCell label="Members" value={formatNumber(org.members)} detail="People with access" emptyDetail="No members" icon={Users} />
          <LedgerCell label="Active keys" value={formatNumber(org.active_keys)} detail="Gateway keys in use" emptyDetail="No key issued" icon={KeyRound} />
          <LedgerCell
            label="Requests"
            value={formatCompact(org.requests)}
            detail={org.last_request_at ? `Last request ${formatRelative(org.last_request_at)}` : "No traffic yet"}
            icon={Activity}
          />
          <LedgerCell label="Model spend" value={usd(org.cost_micros)} detail="Priced at request time" icon={Coins} />
        </Ledger>

        <Panel title="Members" description="Disable a person from the Users page; that signs them out of every organization.">
          <div className="p-4">
            <DataTable columns={MEMBER_COLUMNS} rows={members} rowKey="id" emptyMessage="This organization has no members." />
          </div>
        </Panel>

        <Panel title="Gateway keys" description="Keys are shown by prefix only; the platform never stores a key in full.">
          <div className="p-4">
            <DataTable columns={KEY_COLUMNS} rows={keys} rowKey="id" emptyMessage="This organization has not issued a gateway key." />
          </div>
        </Panel>
      </div>
    </PageShell>
  );
}
