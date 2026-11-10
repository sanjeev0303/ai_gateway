

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
import { Button } from "@ai-governance/console-ui/ui/button";
import { toast } from "sonner";
import { getApiClient } from "@/lib/session";
import { Metadata } from "next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { MembersClient, type MemberRow } from "./members-client";

export const metadata: Metadata = {
  title: "Members · Tollseal",
  description: "Everyone in the organization, with their role and status.",
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
  const { members } = await getApiClient();
  const memberList = await members.list();

  const rows: MemberRow[] = memberList.items.map((member: OrgMember) => ({
    id: member.user.id,
    name: member.user.display_name || member.user.email,
    email: member.user.email,
    role: member.role,
    status: member.user.status,
  }));

  const active = rows.filter((row) => row.status === "active");

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Members"
        description="Everyone in the organization, with their role and status."
        actions={
          <Button asChild size="sm">
            <a href="/app/organization/invitations">
              <UserPlus className="mr-2 h-4 w-4" /> Invite Member
            </a>
          </Button>
        }
      >
        <section className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
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
    </div>
  );
}