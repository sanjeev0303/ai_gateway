import type { ReactNode } from "react";

import { UnavailablePanel } from "@/lib/metrics";

export function gap(blurb: string, needs: readonly string[]): () => ReactNode {
  return function Gap() {
    return (
      <>
        <p className="dash-note">{blurb}</p>
        <UnavailablePanel what="This view" needs={needs} />
      </>
    );
  };
}

export const USAGE_NEEDS = ["GET /v1/usage/aggregates", "GET /v1/usage/daily"] as const;
export const ACTIVITY_NEEDS = ["GET /v1/requests", "GET /v1/requests/stream (SSE)"] as const;
