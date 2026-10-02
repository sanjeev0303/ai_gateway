import type { Metadata } from "next";
import { MailPlus } from "lucide-react";

import { DataTable, PageShell, StatCard, StatusBadge, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatRelative } from "@/lib/format";
import { NOW } from "@/lib/mock-data/clock";
import { users } from "@/lib/mock-data";
import type { OrganizationRole, StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Invitations · Sentinel",
  description: "Pending access requests and outstanding invitations, with expiry.",
};

const ROLE_TONE: Record<OrganizationRole, StatusTone> = {
  owner: "info",
  admin: "info",
  member: "neutral",
  viewer: "neutral",
};

interface InviteRow {
  id: string;
  name: string;
  email: string;
  role: string;
  roleTone: StatusTone;
  sent: string;
  sentTs: number;
  expires: string;
  state: string;
  stateTone: StatusTone;
}

/** Invitations live as `invited` users until they accept. */
const INVITE_TTL_DAYS = 14;

export default function InvitationsPage() {
  const pending = users.filter((user) => user.status === "invited");

  const rows: InviteRow[] = pending.map((user) => {
    const sentTs = Date.parse(user.createdAt);
    const expiresTs = sentTs + INVITE_TTL_DAYS * 86_400_000;
    const expired = NOW.getTime() > expiresTs;
    return {
      id: user.id,
      name: user.name,
      email: user.email,
      role: user.role,
      roleTone: ROLE_TONE[user.role],
      sent: formatDate(user.createdAt),
      sentTs,
      expires: formatDate(new Date(expiresTs).toISOString()),
      state: expired ? "Expired" : "Pending",
      stateTone: expired ? "danger" : "info",
    };
  });

  const expiredCount = rows.filter((row) => row.stateTone === "danger").length;
  const roleCounts = (Object.keys(ROLE_TONE) as OrganizationRole[])
    .map((role) => ({
      role,
      count: pending.filter((user) => user.role === role).length,
    }))
    .filter((entry) => entry.count > 0);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Invitations"
        description="Access that has been offered but not yet accepted. Invitations lapse after two weeks."
        actions={
          <Button size="sm">
            <MailPlus aria-hidden /> Invite people
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Pending invitations"
            value={String(pending.length)}
            tone={pending.length > 0 ? "info" : "neutral"}
            hint="Awaiting acceptance"
          />
          <StatCard
            label="Expiring this week"
            value={String(
              rows.filter(
                (row) => row.stateTone === "info" && NOW.getTime() > row.sentTs + INVITE_TTL_DAYS * 86_400_000,
              ).length,
            )}
            tone="warning"
            hint="Resend to keep access"
          />
          <StatCard
            label="Expired"
            value={String(expiredCount)}
            tone={expiredCount > 0 ? "danger" : "neutral"}
            hint="Can be reissued"
          />
          <StatCard label="Seat headroom" value="26" hint="Available on the current plan" />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Outstanding invitations</CardTitle>
            <CardDescription>
              Resending keeps the original send date; reissuing creates a fresh expiry window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={[
                { key: "name", header: "Invitee", primary: "name", secondary: "email", sort: "name" },
                { key: "role", header: "Role", primary: "role", tone: "roleTone" },
                { key: "state", header: "State", primary: "state", tone: "stateTone" },
                { key: "sent", header: "Sent", primary: "sent", align: "right", sort: "sentTs" },
                { key: "expires", header: "Expires", primary: "expires", align: "right" },
              ] satisfies readonly Column<InviteRow>[]}
              rows={rows}
              rowKey="id"
              toolbar={
                <p className="text-sm text-muted-foreground">
                  {pending.length === 0
                    ? "No open invitations."
                    : `Oldest invitation sent ${pending.length > 0 ? formatRelative([...pending].sort((a, b) => a.createdAt.localeCompare(b.createdAt))[0].createdAt) : ""}.`}
                </p>
              }
              emptyMessage="No open invitations."
            />
          </CardContent>
        </Card>

        {roleCounts.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Pending by role</CardTitle>
              <CardDescription>What access the open invitations would grant.</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="flex flex-wrap gap-2">
                {roleCounts.map((entry) => (
                  <li key={entry.role}>
                    <StatusBadge tone={ROLE_TONE[entry.role]}>
                      {entry.role}: {entry.count}
                    </StatusBadge>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        )}
      </PageShell>
    </div>
  );
}
