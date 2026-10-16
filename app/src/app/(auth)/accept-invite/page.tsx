import type { Metadata } from "next";
import { getSession } from "@/lib/session";
import { AcceptInviteClient } from "./accept-invite-client";

export const metadata: Metadata = {
  title: "Accept Invitation · Sentinel",
  description: "Join your organization on Sentinel AI Gateway.",
};

export default async function AcceptInvitePage(props: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await props.searchParams;
  const session = await getSession();

  const user = session?.claims
    ? {
        name: session.claims.display_name,
        email: session.claims.email,
        role: session.claims.role,
      }
    : null;

  return (
    <main className="flex flex-1 items-center justify-center p-6">
      <AcceptInviteClient token={token || ""} user={user} />
    </main>
  );
}
