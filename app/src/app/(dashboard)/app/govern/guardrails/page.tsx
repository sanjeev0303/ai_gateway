import type { Metadata } from "next";

import type { Guardrail, GuardrailEvent } from "@ai-governance/api-client";
import { PageShell } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { GuardrailsClient } from "./client";

export const metadata: Metadata = {
  title: "Guardrails · Sentinel",
  description: "Input and output detectors, PII scanners, and policy intervention controls.",
};

export default async function GuardrailsPage() {
  let initialGuardrails: Guardrail[] = [];
  let initialEvents: GuardrailEvent[] = [];

  try {
    const { guardrails } = await getApiClient();
    const [resGuardrails, resEvents] = await Promise.allSettled([
      guardrails.list(),
      guardrails.listEvents(),
    ]);

    if (resGuardrails.status === "fulfilled") {
      initialGuardrails = resGuardrails.value.items ?? [];
    }
    if (resEvents.status === "fulfilled") {
      initialEvents = resEvents.value.items ?? [];
    }
  } catch (err) {
    console.error("Failed to load initial guardrails:", err);
  }

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Guardrails & Content Detectors"
        description="Inspect and intercept requests containing PII, API secrets, or disallowed keywords with automated blocking and audit logging."
      >
        <GuardrailsClient
          initialGuardrails={initialGuardrails}
          initialEvents={initialEvents}
        />
      </PageShell>
    </div>
  );
}
