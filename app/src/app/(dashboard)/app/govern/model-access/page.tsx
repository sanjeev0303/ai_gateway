import type { Metadata } from "next";

import {
  EmptyState,
  PageShell,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export const metadata: Metadata = {
  title: "Model access · Sentinel",
  description:
    "Grant-based model admission control. No backing API exists yet.",
};

/**
 * Model access control is enforced on the gateway hot path via the compiled
 * policy snapshot.  The control API exposes no endpoint to read, create, or
 * modify model grants, denials, or plan entitlements, so every figure on this
 * page would have to be fabricated.  Fabricated operational metrics are not
 * permitted.
 *
 * The page therefore describes what the system does, links to the real data
 * sources that do exist, and renders an empty state so the navigation entry
 * is not misleading.
 */
export default function ModelAccessPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Model access"
        description="Grant-based admission is enforced on the gateway. The control API does not yet expose endpoints to read or author grants."
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              No API available
              <Badge variant="outline">No backing endpoint</Badge>
            </CardTitle>
            <CardDescription>
              Model access grants, denials, and plan entitlements are not yet
              exposed by the control API.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <EmptyState
              title="No model grant data"
              description="The gateway enforces model allow and block rules from the compiled policy snapshot, including model allowlist and blocklist rules. No control-plane endpoint exposes the individual grant records, denial counts, or plan entitlements, so there is nothing real to display here. Observed spend and request counts per model are available under Observe → Models and Observe → Costs."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Where model access is actually enforced</CardTitle>
            <CardDescription>
              The enforcement path already runs on every request.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <p>
              The gateway consults a compiled, immutable policy snapshot on
              every request. That snapshot includes model allowlist and
              blocklist rules drawn from the{" "}
              <code className="font-mono">policies</code> and{" "}
              <code className="font-mono">policy_rules</code> tables in the
              database. A blocked model receives a 403 before any upstream call
              is made.
            </p>
            <p>
              Because enforcement runs from the snapshot rather than live
              database reads, a grant or revocation takes effect when the
              snapshot is next rebuilt, not at the instant a row is changed.
            </p>
            <p>
              No control-plane endpoint reads those rows or exposes them as
              grants, so there is no admission log to replay, no denial count
              to aggregate, and no entitlement table to render.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
