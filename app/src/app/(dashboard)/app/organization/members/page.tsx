

import * as React from "react";
import { UserPlus, Loader2, Trash2, CheckCircle, XCircle } from "lucide-react";
import {
  DataTable,
  EmptyState,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { formatUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";
import { Metadata } from "next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MembersClient, type MemberRow } from "./members-client";

export const metadata: Metadata = {
  title: "Members · Sentinel",
  description: "Everyone in the organization, with their role, status, and attributed spend.",
};

interface OrgMember {
  user: {
    id: string;
    display_name: string;
    email: string;
    status: string;
  };
  role: string;
}


export default async function MembersPage() {
  const { members, usage } = await getApiClient();
  const [memberList, spendByUser] = await Promise.all([members.list(), usage.byUser()]);

  const spendMap = new Map(
    spendByUser.items.map((row: any) => [row.UserID, formatUsd(row.CostMicros)]),
  );

  const rows: MemberRow[] = memberList.items.map((member: OrgMember) => ({
    id: member.user.id,
    name: member.user.display_name || member.user.email,
    email: member.user.email,
    role: member.role,
    status: member.user.status,
    spend: spendMap.get(member.user.id) ?? "--",
  }));

  const active = rows.filter((row) => row.status === "active");

  return (
    <PageShell title="Members" description="Everyone in the organization, with their role and status">
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Active members" value={String(active.length)} tone="success" hint="Enabled and signing in" />
      </section>

      <div className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle>All members</CardTitle>
          </CardHeader>
          <CardContent>
            <MembersClient initialRows={rows} />
          </CardContent>
        </Card>
      </div>
    </PageShell>
  );
}