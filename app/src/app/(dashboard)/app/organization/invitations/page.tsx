import type { Metadata } from "next";
import { MailPlus } from "lucide-react";

import { PageShell, StatCard, StatusBadge } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatDate, formatRelative } from "@/lib/format";
import { ROLE_ORDER, ROLE_TONE, toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { InviteDialog, InvitationsClient, type InviteRow } from "./invitations-client";

export const metadata: Metadata = {
  title: "Invitations · Sentinel",
  description: "Pending access requests and outstanding invitations, with expiry.",
};

const STATE_TONE: Record<string, StatusTone> = {
  pending: "info",
  accepted: "success",
  revoked: "neutral",
  expired: "danger",
};

const DAY_MS = 86_400_000;

const lapsingWithin = (expiresAt: string, days: number, now: Date = new Date()): boolean =>
  Date.parse(expiresAt) - now.getTime() <= days * DAY_MS;



export default async function InvitationsPage() {
  const { invitations } = await getApiClient();
  const { items } = await invitations.list();

  const rows: InviteRow[] = items
    .filter((invite) => invite.status === "pending" || invite.status === "expired")
    .map((invite) => ({
      id: invite.id,
      email: invite.email,
      role: invite.role,
      roleTone: toneFor(ROLE_TONE, invite.role),
      state: invite.status,
      stateTone: toneFor(STATE_TONE, invite.status),
      sent: formatDate(invite.created_at),
      sentTs: Date.parse(invite.created_at),
      expires: invite.expires_at ? formatDate(invite.expires_at) : "—",
    }));

  const pending = items.filter((invite) => invite.status === "pending");
  const expired = items.filter((invite) => invite.status === "expired");
  const expiringSoon = pending.filter((invite) => lapsingWithin(invite.expires_at, 7)).length;

  const oldest = pending.reduce<string | null>(
    (acc, invite) => (acc === null || invite.created_at < acc ? invite.created_at : acc),
    null,
  );

  const roleCounts = ROLE_ORDER.map((role) => ({
    role,
    count: pending.filter((invite) => invite.role === role).length,
  })).filter((entry) => entry.count > 0);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Invitations"
        description="Access that has been offered but not yet accepted. Invitations lapse after two weeks."
        actions={
          <InviteDialog>
            <Button size="sm">
              <MailPlus aria-hidden /> Invite people
            </Button>
          </InviteDialog>
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
            value={String(expiringSoon)}
            tone={expiringSoon > 0 ? "warning" : "neutral"}
            hint="Resend to keep access"
          />
          <StatCard
            label="Expired"
            value={String(expired.length)}
            tone={expired.length > 0 ? "danger" : "neutral"}
            hint="Can be reissued"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Outstanding invitations</CardTitle>
            <CardDescription>
              Resending keeps the original send date; reissuing creates a fresh expiry window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <InvitationsClient
              initialRows={rows}
              oldestStr={oldest !== null ? formatRelative(oldest) : null}
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
                    <StatusBadge tone={toneFor(ROLE_TONE, entry.role)}>
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
