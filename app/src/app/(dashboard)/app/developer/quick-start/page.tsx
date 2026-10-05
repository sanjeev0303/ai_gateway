import type { Metadata } from "next";
import { KeyRound, Rocket, Terminal } from "lucide-react";

import { CodeBlock } from "@/components/developer/code-block";
import { PageShell } from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { codeSnippets } from "@/lib/developer-content";

export const metadata: Metadata = {
  title: "Quick start · Sentinel",
  description: "Authenticate, send a first request, and verify it landed in the audit log.",
};

const STEPS = [
  {
    icon: KeyRound,
    title: "Create a credential",
    body: "Credentials carry a scope (gateway, control, or all) and can be pinned to a team or project. The raw key is shown once, at creation.",
  },
  {
    icon: Terminal,
    title: "Export the key",
    body: "The gateway reads the key from the x-api-key header, so keep it in the environment rather than source control.",
  },
  {
    icon: Rocket,
    title: "Send your first request",
    body: "Every call is authenticated, rate- and budget-checked, then recorded. Tokens, latency and cost land in usage.",
  },
] as const;

export default function DeveloperQuickStartPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Quick start"
        description="Three steps from nothing to a metered, policy-checked request."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Base URL:{" "}
            <code className="font-mono">http://localhost:8084</code> ·{" "}
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
              Credentials are resolved and hashed at the gateway before any provider is
              called. Send the raw key in the x-api-key header.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <CodeBlock
              label="Create a credential"
              language="bash"
              code={`aigw credential create \\
  --name "my-integration" \\
  --subject-type user \\
  --subject-id "$USER_ID" \\
  --scopes gateway`}
            />
            <CodeBlock
              label="Export it"
              language="bash"
              code={`export AIGW_API_KEY="aigw_..."`}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Send a request</CardTitle>
            <CardDescription>
              There is no vendor SDK. The gateway speaks Anthropic&apos;s messages
              shape, so any HTTP client works.
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
              Authentication, limits, and metering all happen on the hot path before
              your caller sees the response.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ol className="space-y-3 text-sm">
              {[
                "The raw key is hashed and the credential is resolved; expiry, revocation, and scope are checked.",
                "Rate limits and budget thresholds are evaluated for the credential's team or project.",
                "The request body is forwarded as sent — only the auth and version headers are rewritten for the target provider.",
                "Tokens and latency are recorded as usage events and rolled up into hourly and daily buckets.",
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
