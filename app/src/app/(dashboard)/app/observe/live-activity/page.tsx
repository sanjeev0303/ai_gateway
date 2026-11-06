import type { Metadata } from "next";

import { PageShell } from "@/components/dashboard";
import { getDirectory, nameMap } from "@/lib/directory";
import { getApiClient } from "@/lib/session";

import { LiveActivityContent } from "./live-activity-content";

export const metadata: Metadata = {
  title: "Live activity · Tollseal",
  description: "Requests streaming through the gateway right now.",
};

/**
 * Initial snapshot only.
 *
 * This runs on the server so the first paint already has rows and so the browser
 * never talks to the database. Ongoing updates arrive over SSE in
 * `useLiveActivity`, which reconnects on its own.
 */

import { ObserveFilters } from "@/components/dashboard";

async function loadActivity() {
  const { activity } = await getApiClient();

  const [events, activeUsers] = await Promise.all([
    activity.events(),
    activity.activeUsers(),
  ]);

  return {
    events: events.data,
    activeUsers: activeUsers.count,
  };
}

export default async function LiveActivityPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  await searchParams;
  const [{ events, activeUsers }, directory] = await Promise.all([
    loadActivity(),
    getDirectory(),
  ]);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Live activity"
        description="Requests the gateway is processing right now, newest first."
      >
        <ObserveFilters exclude={["service_account_id"]} />
        <LiveActivityContent
          initialEvents={events}
          initialActiveUsers={activeUsers}
          names={{
            teams: nameMap(directory.teams),
            projects: nameMap(directory.projects),
            users: nameMap(directory.users),
          }}
        />
      </PageShell>
    </div>
  );
}
