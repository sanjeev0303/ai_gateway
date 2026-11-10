"use client";

import { ErrorState } from "@ai-governance/console-ui/dashboard/state";

/** Catches anything a platform page throws past its own error handling. */
export default function AdminError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  console.error("Platform console error:", error);
  return (
    <ErrorState
      title="This page could not be shown"
      description={error.digest ? `Reload to try again. Reference ${error.digest} in the server logs.` : "Reload to try again."}
      onRetry={reset}
    />
  );
}
