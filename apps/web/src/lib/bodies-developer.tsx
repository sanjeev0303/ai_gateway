import { Mono } from "@/components/ui";

function Code({ children }: { readonly children: string }) {
  return <pre className="code-block mono">{children}</pre>;
}

export function QuickStartBody() {
  return (
    <>
      <p className="dash-note">
        Point your provider client at the gateway instead of the provider. The gateway authenticates
        the key, resolves the compiled policy snapshot, applies rate and budget checks, forwards the
        request, and streams the response back.
      </p>
      <Code>{`curl http://localhost:8084/v1/messages \\
  -H "x-api-key: $AIGW_GATEWAY_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "content-type: application/json" \\
  -H "x-routing-team: platform" \\
  -d '{
    "model": "claude-sonnet-4-5",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Summarize this quarter."}]
  }'`}</Code>
      <p className="dash-note">
        <Mono>x-routing-*</Mono> headers let the caller steer routing; the gateway records them on the
        request. Replace <Mono>x-api-key</Mono> with a <Mono>Authorization</Mono> header if that suits
        your client better.
      </p>
      <p className="dash-note">
        Gateway routes are <Mono>POST /v1/messages</Mono>, <Mono>POST /v1/messages/count_tokens</Mono>,{" "}
        <Mono>GET /v1/models</Mono>, <Mono>GET /v1/gateway/stats</Mono>, and{" "}
        <Mono>GET /v1/gateway/health</Mono>. Provider payloads pass through provider-native, so
        there is no universal request schema to learn.
      </p>
      <p className="dash-note">
        A gateway key is shown once at creation and cannot be read back. If you lose it, rotate it.
      </p>
    </>
  );
}

export function SdksBody() {
  return (
    <>
      <p className="dash-note">
        Two shared TypeScript packages ship in this monorepo. Both are private workspace packages, so
        they resolve from the repo rather than a public registry.
      </p>
      <Code>{`@ai-governance/api-client   typed client for the control + gateway API
@ai-governance/contracts     types generated from backend/api/openapi/openapi.yaml`}</Code>
      <p className="dash-note">
        Regenerate the contract types after an OpenAPI change:{" "}
        <Mono>npm run generate</Mono> in <Mono>packages/contracts</Mono>.
      </p>
    </>
  );
}

export function CliBody() {
  return (
    <>
      <p className="dash-note">
        <Mono>packages/cli</Mono> (<Mono>@ai-governance/cli</Mono>) is a declared workspace package
        but is not implemented yet — it currently carries only its manifest, with no binary, no
        commands, and no entry point.
      </p>
      <p className="dash-note">
        Until it exists, use the control API directly with <Mono>@/v1/auth/dev-login</Mono> for local
        sessions. Nothing on this page describes CLI behavior that does not exist.
      </p>
    </>
  );
}

export function ApiDocsBody() {
  return (
    <>
      <p className="dash-note">
        The control API serves its OpenAPI document and interactive reference under{" "}
        <Mono>/docs</Mono> on port 8081. The web app also hosts a readable guide at{" "}
        <Mono>/docs/api</Mono>.
      </p>
      <p className="dash-note">
        Be aware of a known gap: the OpenAPI spec currently documents only the health endpoints, not
        the implemented <Mono>/v1</Mono> routes. Treat the spec as incomplete and the route handlers
        as the source of truth until the contract is filled in.
      </p>
    </>
  );
}
