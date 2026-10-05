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

// Hardcoded from gatewayv2/server.go. Anthropic-compatible, no /v1/chat/completions.
// Scopes are credential scopes (see domain/credentials.go), not per-endpoint permissions.
const ENDPOINTS: readonly EndpointRow[] = [
  { id: "e1", method: "POST", methodTone: "success", path: "/v1/messages", summary: "Create a message through the gateway", scope: "gateway" },
  { id: "e2", method: "POST", methodTone: "success", path: "/v1/messages/count_tokens", summary: "Count input tokens without calling a provider", scope: "gateway" },
  { id: "e3", method: "GET", methodTone: "info", path: "/v1/models", summary: "List models the calling credential can use", scope: "gateway" },
  { id: "e4", method: "GET", methodTone: "info", path: "/v1/gateway/stats", summary: "Gateway pipeline statistics", scope: "gateway" },
  { id: "e5", method: "GET", methodTone: "info", path: "/v1/gateway/health", summary: "Upstream provider reachability", scope: "gateway" },
  { id: "e6", method: "GET", methodTone: "info", path: "/livez", summary: "Liveness probe", scope: "none" },
  { id: "e7", method: "GET", methodTone: "info", path: "/readyz", summary: "Readiness probe", scope: "none" },
  { id: "e8", method: "GET", methodTone: "info", path: "/healthz", summary: "Health probe", scope: "none" },
];

const SCOPES = ["gateway", "control", "all"];

// Statuses and types are the ones gateway.go actually emits via writeError.
const ERRORS = [
  { status: "401", name: "authentication_error", meaning: "The credential was not found, expired, or revoked." },
  { status: "403", name: "policy_denied", meaning: "The authorization stage refused the request." },
  { status: "405", name: "method_not_allowed", meaning: "The path exists but not for this HTTP method." },
  { status: "429", name: "rate_limit_exceeded", meaning: "The credential exceeded its rate limit." },
  { status: "429", name: "budget_exceeded", meaning: "The credential exceeded its budget ceiling." },
  { status: "500", name: "routing_error", meaning: "No upstream could be routed for the request." },
  { status: "503", name: "upstream_unavailable", meaning: "The tenant's upstream could not be resolved." },
  { status: "503", name: "policy_unavailable", meaning: "The policy snapshot could not be loaded." },
] as const;

export default function ApiDocsPage() {
  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="API docs"
        description="The gateway is the model data plane. It is a separate service from the control plane the dashboard talks to."
        eyebrow={
          <p className="text-xs text-muted-foreground">
            Base URL <code className="font-mono">http://localhost:8084</code> ·{" "}
            <StatusBadge tone="info">Anthropic-compatible</StatusBadge>
          </p>
        }
      >
        <section className="grid gap-4 sm:grid-cols-3">
          <StatCard label="Endpoints" value={String(ENDPOINTS.length)} hint="Registered in gatewayv2" />
          <StatCard label="Auth header" value="x-api-key" tone="info" hint="Credential scope required" />
          <StatCard label="Response shape" value="Anthropic" hint="Upstream body, unrewritten" />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Authentication</CardTitle>
            <CardDescription>
              The raw key is read from the x-api-key header, falling back to the
              Authorization header. It is hashed and looked up before any provider is
              contacted.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="Send the key"
              language="bash"
              code={`curl http://localhost:8084/v1/models \\
  -H "x-api-key: $AIGW_API_KEY"`}
            />
            <div className="flex flex-wrap gap-2">
              {SCOPES.map((scope) => (
                <StatusBadge key={scope} tone="neutral">
                  {scope}
                </StatusBadge>
              ))}
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
              The gateway forwards the request body as sent, rewriting only the auth and
              version headers the target provider expects, and streams the provider&apos;s
              response straight back. It does not wrap the payload.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="Response"
              language="bash"
              code={`{
  "id": "msg_8f3a12",
  "type": "message",
  "role": "assistant",
  "model": "claude-sonnet-4-5",
  "content": [{ "type": "text", "text": "…" }],
  "stop_reason": "end_turn",
  "usage": { "input_tokens": 128, "output_tokens": 412 }
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
            <CardTitle>Rate limits and budgets</CardTitle>
            <CardDescription>
              Both are evaluated on the request path. A breach fails the call outright
              rather than degrading the response.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <CodeBlock
              label="Rejected requests"
              language="bash"
              code={`{ "type": "error", "error": { "type": "rate_limit_exceeded", "message": "…" } }
{ "type": "error", "error": { "type": "budget_exceeded",      "message": "…" } }`}
            />
            <CodeBlock
              label="Correlation"
              language="bash"
              code={`x-request-id: 8f3a12c4`}
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
