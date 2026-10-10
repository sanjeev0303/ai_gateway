import type { Metadata } from "next";

import { PageShell } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";

import { LiveActivityContent } from "./live-activity-content";

export const metadata: Metadata = {
  title: "Live activity · Sentinel",
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

  const [events, activeRequests, activeUsers] = await Promise.all([
    activity.events(),
    activity.activeRequests(),
    activity.activeUsers(),
  ]);

  return {
    events: events.data,
    activeRequests: activeRequests.count,
    activeUsers: activeUsers.count,
  };
}

export default async function LiveActivityPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  // TODO: pass filters down to SSE / client component if we filter on client
  const { events, activeRequests, activeUsers } = await loadActivity();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Live activity"
        description="Requests the gateway is processing right now, newest first."
      >
        <ObserveFilters />
        <LiveActivityContent
          initialEvents={events}
          initialActiveRequests={activeRequests}
          initialActiveUsers={activeUsers}
        />
      </PageShell>
    </div>
  );
}
