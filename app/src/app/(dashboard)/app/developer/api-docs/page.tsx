import type { Metadata } from "next";

import { CodeBlock } from "@/components/developer/code-block";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { organization } from "@/lib/mock-data";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "API docs · Sentinel",
  description: "Gateway endpoints, authentication scopes, and the error contract.",
};

interface EndpointRow {
  id: string;
  method: string;
  methodTone: StatusTone;
  path: string;
  summary: string;
  scope: string;
}

const COLUMNS: readonly Column<EndpointRow>[] = [
  { key: "method", header: "Method", primary: "method", tone: "methodTone" },
  { key: "path", header: "Path", primary: "path", sort: "path" },
  { key: "summary", header: "Purpose", primary: "summary", secondary: "scope", sort: "summary" },
];

const ENDPOINTS: readonly EndpointRow[] = [
  { id: "e1", method: "POST", methodTone: "success", path: "/v1/chat/completions", summary: "Create a chat completion through the gateway", scope: "gateway:write" },
  { id: "e2", method: "POST", methodTone: "success", path: "/v1/embeddings", summary: "Create embeddings for a batch of inputs", scope: "gateway:write" },
  { id: "e3", method: "GET", methodTone: "info", path: "/v1/models", summary: "List models available to the calling project", scope: "gateway:read" },
  { id: "e4", method: "GET", methodTone: "info", path: "/v1/models/{model_id}", summary: "Retrieve one model with its entitlement state", scope: "gateway:read" },
  { id: "e5", method: "GET", methodTone: "info", path: "/v1/usage", summary: "Query metered usage and cost for a window", scope: "usage:read" },
  { id: "e6", method: "GET", methodTone: "info", path: "/v1/audit/events", summary: "Read the immutable administrative audit trail", scope: "audit:read" },
  { id: "e7", method: "GET", methodTone: "info", path: "/v1/budgets", summary: "List budget thresholds and current consumption", scope: "budgets:read" },
  { id: "e8", method: "PATCH", methodTone: "warning", path: "/v1/budgets/{budget_id}", summary: "Adjust a budget threshold or its period", scope: "budgets:write" },
  { id: "e9", method: "GET", methodTone: "info", path: "/v1/billing/invoices", summary: "List invoices for the authenticated organization", scope: "billing:read" },
];

const ERRORS = [
  { status: "400", name: "invalid_request", meaning: "The payload failed validation before any policy was evaluated." },
  { status: "401", name: "unauthenticated", meaning: "The key is missing, malformed, or revoked." },
  { status: "403", name: "policy_denied", meaning: "A policy matched and blocked the request. The reason is in `error.detail`." },
  { status: "404", name: "model_not_found", meaning: "The model is not in the registry, or not entitled for this project." },
  { status: "429", name: "rate_limited", meaning: "Rate or budget ceiling hit. `retry_after` is in seconds." },
  { status: "502", name: "upstream_error", meaning: "The provider failed. Safe to retry with backoff." },
] as const;

export default function ApiDocsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="API docs"
        description="One REST surface. Everything the dashboard does is reachable from here."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Base URL <code className="font-mono">https://api.sentinel.dev</code> ·{" "}
            <StatusBadge tone="info">v1 stable</StatusBadge>
          </p>
        }
      >
        <section className="grid gap-4 sm:grid-cols-3">
          <StatCard label="Endpoints" value={String(ENDPOINTS.length)} hint="Stable in v1" />
          <StatCard
            label="Auth"
            value="Bearer"
            tone="info"
            hint="Scoped API key"
          />
          <StatCard
            label="Region"
            value={organization.region}
            hint="Requests stay in-region"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Authentication</CardTitle>
            <CardDescription>
              Scopes are enforced at the gateway. A key is refused before the request
              reaches a provider if it lacks the scope the endpoint requires.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="Authorization header"
              language="bash"
              code={`curl https://api.sentinel.dev/v1/models \\
  -H "Authorization: Bearer $SENTINEL_API_KEY"`}
            />
            <div className="flex flex-wrap gap-2">
              {["gateway:read", "gateway:write", "usage:read", "budgets:read", "audit:read", "billing:read"].map(
                (scope) => (
                  <StatusBadge key={scope} tone="neutral">
                    {scope}
                  </StatusBadge>
                ),
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Endpoints</CardTitle>
            <CardDescription>
              The required scope for each route is listed in the second line of its row.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={ENDPOINTS}
              rowKey="id"
              emptyMessage="No endpoints available."
            />
          </CardContent>
        </Card>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Request and response</CardTitle>
              <CardDescription>
                Responses echo the identifiers that tie the call back to a trace.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <CodeBlock
                label="Response"
                language="bash"
                code={`{
  "id": "resp_8f3a12",
  "model": "claude-sonnet-4-5",
  "choices": [
    { "message": { "role": "assistant", "content": "…" } }
  ],
  "usage": { "input_tokens": 128, "output_tokens": 412 },
  "sentinel": {
    "trace_id": "trc_1b9d40",
    "project": "customer-portal",
    "policy_decision": "allow",
    "cost_usd": 0.0043
  }
}`}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Error contract</CardTitle>
              <CardDescription>
                Errors share one shape, so a client handles them with a single branch.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="divide-y">
                {ERRORS.map((error) => (
                  <li key={error.status} className="flex items-start gap-3 py-2.5">
                    <span className="w-10 shrink-0 font-mono text-sm tabular-nums text-muted-foreground">
                      {error.status}
                    </span>
                    <div className="min-w-0">
                      <p className="font-mono text-sm">{error.name}</p>
                      <p className="text-xs text-muted-foreground">{error.meaning}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Rate limits</CardTitle>
            <CardDescription>
              Per key, per organization. Every response carries the current window state.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <CodeBlock
              label="Response headers"
              language="bash"
              code={`X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 847
X-RateLimit-Reset: 1789543200
Retry-After: 12   # present on 429 only`}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
