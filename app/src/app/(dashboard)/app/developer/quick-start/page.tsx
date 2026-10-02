import type { Metadata } from "next";
import { KeyRound, Rocket, Terminal } from "lucide-react";

import { CodeBlock } from "@/components/developer/code-block";
import { PageShell, StatusBadge } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { developerApiKeys, codeSnippets, organization } from "@/lib/mock-data";

export const metadata: Metadata = {
  title: "Quick start · Sentinel",
  description: "Authenticate, send a first request, and verify it landed in the audit log.",
};

const STEPS = [
  {
    icon: KeyRound,
    title: "Create an API key",
    body: "Keys are scoped and environment-bound. A production key can call the gateway; a development key cannot reach production projects.",
  },
  {
    icon: Terminal,
    title: "Set the environment variable",
    body: "The gateway reads the key from the environment so it never ends up in source control.",
  },
  {
    icon: Rocket,
    title: "Send your first request",
    body: "Every call is metered, policy-checked, and recorded. The response carries the same identifiers shown in the dashboard.",
  },
] as const;

export default function DeveloperQuickStartPage() {
  const productionKey = developerApiKeys.find(
    (key) => key.environment === "production" && key.status === "active",
  );

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Quick start"
        description="Three steps from nothing to a metered, policy-checked request."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Base URL:{" "}
            <code className="font-mono">https://api.sentinel.dev/v1</code> ·{" "}
            {organization.region}
          </p>
        }
      >
        <div className="grid gap-4 lg:grid-cols-3">
          {STEPS.map((step, index) => (
            <Card key={step.title}>
              <CardHeader>
                <div className="flex items-center gap-2.5">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                    {index + 1}
                  </span>
                  <step.icon className="size-4 text-muted-foreground" aria-hidden />
                </div>
                <CardTitle className="text-base">{step.title}</CardTitle>
                <CardDescription>{step.body}</CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Authenticate</CardTitle>
            <CardDescription>
              Bearer tokens are checked at the gateway, before any provider is called.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {productionKey && (
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium">{productionKey.name}</p>
                  <p className="font-mono text-xs text-muted-foreground">
                    {productionKey.maskedKey}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge tone="info">{productionKey.environment}</StatusBadge>
                  <StatusBadge tone="success">{productionKey.status}</StatusBadge>
                </div>
              </div>
            )}
            <CodeBlock
              label="Create a key"
              language="bash"
              code={`sentinel keys create \\
  --name "my-integration" \\
  --environment production \\
  --scope gateway:write`}
            />
            <CodeBlock
              label="Export it"
              language="bash"
              code={`export SENTINEL_API_KEY="sk_live_..."`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Send a request</CardTitle>
            <CardDescription>
              The same call in three languages. All three hit the identical endpoint and
              produce an identical audit entry.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {codeSnippets.map((snippet) => (
              <CodeBlock
                key={snippet.id}
                label={snippet.label}
                language={snippet.language}
                code={snippet.code}
              />
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>What happens next</CardTitle>
            <CardDescription>
              The call is not just billed — it is evaluated against the policies that
              applied at request time.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ol className="space-y-3 text-sm">
              {[
                "Credentials are resolved and the key's scopes are checked.",
                "The request is matched against active policies, budgets, and rate limits.",
                "The provider is called; latency, tokens, and cost are recorded.",
                "The trace appears under Live activity and is attributed to a project and team.",
                "Guardrails evaluate the response before it reaches your caller.",
              ].map((line, index) => (
                <li key={line} className="flex gap-3">
                  <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-[0.7rem] font-medium tabular-nums">
                    {index + 1}
                  </span>
                  <span className="text-muted-foreground">{line}</span>
                </li>
              ))}
            </ol>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
