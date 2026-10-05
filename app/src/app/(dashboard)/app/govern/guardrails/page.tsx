import type { Metadata } from "next";

import { EmptyState, PageShell } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export const metadata: Metadata = {
  title: "Guardrails · Sentinel",
  description: "Input and output detectors, their thresholds, and how often each one actually intervenes.",
};

export default function GuardrailsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Guardrails"
        description="Input and output detectors, their thresholds, and how often each one actually intervenes."
      >
        <Card>
          <CardHeader>
            <CardTitle>No guardrail data</CardTitle>
            <CardDescription>
              The control API exposes no guardrail, detector, or intervention endpoint.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <EmptyState
              title="No detectors are running"
              description="No detector, threshold, or intervention count is recorded anywhere, so there is nothing to configure or tune here. The request outcomes a guardrail would be judged on are real and visible under Observe → Live activity."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
