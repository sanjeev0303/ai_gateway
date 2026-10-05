import type { Metadata } from "next";

import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatNumber, formatRelative } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Gateway credentials · Sentinel",
  description: "Every credential the gateway holds, with expiry and scope.",
};

const CREDENTIAL_TONE: Record<string, StatusTone> = {
  active: "success",
  expired: "danger",
  revoked: "neutral",
};

const EXPIRY_WINDOW_DAYS = 30;

const short = (id?: string) => (id ? `${id.slice(0, 8)}…` : "—");

interface CredentialRow {
  id: string;
  name: string;
  prefix: string;
  subject: string;
  subjectId: string;
  owner: string;
  ownerId: string;
  scopes: string;
  status: string;
  tone: StatusTone;
  expires: string;
  expiresSort: string;
  lastUsed: string;
}

const COLUMNS: readonly Column<CredentialRow>[] = [
  {
    key: "name",
    header: "Credential",
    primary: "name",
    secondary: "prefix",
    sort: "name",
  },
  {
    key: "subject",
    header: "Subject",
    primary: "subject",
    secondary: "subjectId",
    sort: "subject",
  },
  { key: "owner", header: "Owner", primary: "owner", secondary: "ownerId", sort: "owner" },
  { key: "scopes", header: "Scopes", primary: "scopes", sort: "scopes" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  {
    key: "expires",
    header: "Expires",
    primary: "expires",
    sort: "expiresSort",
    align: "right",
  },
  {
    key: "lastUsed",
    header: "Last used",
    primary: "lastUsed",
    sort: "lastUsed",
    align: "right",
  },
];

async function loadGatewayCredentials() {
  const { credentials } = await getApiClient();
  const list = await credentials.listGateway();
  const now = new Date();
  const windowEnd = now.getTime() + EXPIRY_WINDOW_DAYS * 24 * 60 * 60 * 1000;

  const rows: CredentialRow[] = [...list.items]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((credential) => {
      const owner = credential.project_id
        ? "Project"
        : credential.team_id
          ? "Team"
          : "Organization";

      return {
        id: credential.id,
        name: credential.name,
        prefix: credential.key_prefix,
        subject: credential.subject_type.replace(/_/g, " "),
        subjectId: short(credential.subject_id),
        owner,
        ownerId: short(credential.project_id ?? credential.team_id),
        scopes: credential.scopes.length ? credential.scopes.join(", ") : "none",
        status: credential.status,
        tone: toneFor(CREDENTIAL_TONE, credential.status),
        expires: credential.expires_at ? formatDate(credential.expires_at) : "Never",
        expiresSort: credential.expires_at ?? "9999-12-31",
        lastUsed: credential.last_used_at
          ? formatRelative(credential.last_used_at, now)
          : "Never",
      };
    });

  const expiring = list.items.filter((credential) => {
    if (credential.status !== "active" || !credential.expires_at) return false;
    const at = new Date(credential.expires_at).getTime();
    return at >= now.getTime() && at <= windowEnd;
  });

  const schedule = list.items
    .filter((credential) => Boolean(credential.expires_at))
    .sort((a, b) => (a.expires_at ?? "").localeCompare(b.expires_at ?? ""));

  return {
    rows,
    total: rows.length,
    active: rows.filter((row) => row.status === "active").length,
    expiring: expiring.length,
    expired: list.items.filter((credential) => credential.status === "expired").length,
    schedule: schedule.map((credential) => ({
      id: credential.id,
      name: credential.name,
      status: credential.status,
      tone: toneFor(CREDENTIAL_TONE, credential.status),
      expires: formatDate(credential.expires_at ?? ""),
    })),
  };
}

export default async function GatewayCredentialsPage() {
  const { rows, total, active, expiring, expired, schedule } = await loadGatewayCredentials();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Gateway credentials"
        description="Keys the gateway accepts from callers, scoped to a subject and optionally to one team or project."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Gateway credentials"
            value={formatNumber(total)}
            hint={`${formatNumber(active)} active`}
          />
          <StatCard
            label={`Expiring within ${EXPIRY_WINDOW_DAYS} days`}
            value={formatNumber(expiring)}
            tone={expiring > 0 ? "warning" : "success"}
            hint="Active, with an expiry date set"
          />
          <StatCard
            label="Expired"
            value={formatNumber(expired)}
            tone={expired > 0 ? "danger" : "neutral"}
            hint="Past its expiry date"
          />
          <StatCard
            label="Revoked"
            value={formatNumber(total - active - expired)}
            tone={total - active - expired > 0 ? "neutral" : "success"}
            hint="Withdrawn before expiry"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Stored credentials</CardTitle>
            <CardDescription>
              Only the key prefix is stored and returned. The secret itself is shown
              once, at creation, and cannot be read back.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No gateway credentials yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Expiry schedule</CardTitle>
            <CardDescription>
              Credentials carrying an expiry date, soonest first. Nothing rotates
              automatically — the control API only records the date.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {schedule.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No gateway credential has an expiry date set.
              </p>
            ) : (
              <ul className="space-y-2">
                {schedule.map((credential) => (
                  <li
                    key={credential.id}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{credential.name}</p>
                      <p className="truncate text-xs text-muted-foreground">
                        expires {credential.expires}
                      </p>
                    </div>
                    <StatusBadge tone={credential.tone}>{credential.status}</StatusBadge>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
