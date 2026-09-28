"use client";

import {
  GatewayCredentialsBody,
  ProviderCredentialsBody,
  ServiceAccountsBody,
} from "@/lib/sections";
import { SpecView } from "@/lib/spec";

export function PoliciesBody() {
  return (
    <SpecView
      blurb="A visual policy builder. Rules are authored against a scope, compiled into an immutable snapshot, and propagated to the gateway hot path."
      needs={["GET /v1/policies", "POST /v1/policies", "GET /v1/policies/compiled"]}
      groups={[
        {
          title: "Scopes",
          items: ["organization", "team", "project", "user", "service account", "credential"],
        },
        {
          title: "Policy sections",
          items: [
            "Providers",
            "Models",
            "Rate limits",
            "Token budgets",
            "Spend budgets",
            "Request limits",
            "Project access",
            "Credential scopes",
            "Content controls",
          ],
        },
        {
          title: "Compiled preview",
          items: ["Policy version", "Rules", "Effective scope", "Last compiled", "Last propagated"],
        },
      ]}
      note="The gateway already resolves a compiled policy snapshot and enforces rate and budget checks against it, so the enforcement path exists. What is missing is authoring: there is no policy engine API to write, compile, or version a rule."
    />
  );
}

export function ModelAccessBody() {
  return (
    <SpecView
      blurb="Which members of a scope may call which models on which providers."
      needs={["GET /v1/model-access", "PUT /v1/model-access"]}
      groups={[
        { title: "Columns", items: ["scope", "model", "provider", "allowed", "effect", "updated"] },
        { title: "Actions", items: ["allow", "deny", "reset to provider default"] },
      ]}
      note="A deny here is a governance decision, not a routing hint: the gateway resolves policy before it selects a provider."
    />
  );
}

export function BudgetsLimitsBody() {
  return (
    <SpecView
      blurb="Rate limits and spend caps enforced in the request path, per scope."
      needs={["GET /v1/budgets", "GET /v1/limits", "PUT /v1/budgets"]}
      groups={[
        { title: "Budgets", items: ["token budgets", "spend budgets", "request limits"] },
        { title: "Rate limits", items: ["requests per minute", "tokens per minute", "window"] },
        { title: "Per scope", items: ["organization", "team", "project", "user", "service account"] },
        { title: "Enforcement", items: ["remaining", "exceeded", "reset at", "breach event"] },
      ]}
      note="Rate and budget checks run in the gateway hot path and reject requests when exceeded. Nothing persists the counters, so remaining and breach history cannot be displayed yet."
    />
  );
}

export function GuardrailsBody() {
  return (
    <SpecView
      blurb="Content controls applied to requests and responses."
      needs={["GET /v1/guardrails", "PUT /v1/guardrails"]}
      groups={[
        {
          title: "Controls",
          items: ["denied topics", "PII redaction", "retention override", "egress restriction"],
        },
        { title: "Per scope", items: ["organization", "team", "project", "user", "service account"] },
        { title: "Enforcement", items: ["action", "last evaluated", "matches"] },
      ]}
      note="These are the request-path counterpart to the Content controls section in the policy builder. Privacy modes and retention windows are not implemented, so no control here can currently change what is retained."
    />
  );
}

export function CredentialsBody() {
  return (
    <>
      <p className="dash-note">
        The three credential kinds are separate on purpose. A provider credential authenticates the
        gateway <em>to</em> a provider; a gateway credential authenticates a caller <em>of</em> the
        gateway; a service account is a non-human principal that keys attach to.
      </p>
      <p className="dash-note">
        Raw credential values are returned exactly once, in the response to the create or rotate call.
        They are stored hashed and cannot be read back through any endpoint. The tables below show
        only the key prefix or fingerprint.
      </p>

      <div className="dash-panel">
        <p className="dash-panel-title">Provider credentials</p>
        <p className="dash-note">
          Used by the gateway to reach Anthropic, OpenAI, or Gemini. Owner, fingerprint, and status
          only — the secret is never displayed.
        </p>
        <ProviderCredentialsBody />
      </div>

      <div className="dash-panel">
        <p className="dash-panel-title">Gateway credentials</p>
        <p className="dash-note">
          Owner, scope, team, project, status, created, expires, and last used. Actions: create,
          rotate, revoke, expire, restore.
        </p>
        <GatewayCredentialsBody />
      </div>

      <div className="dash-panel">
        <p className="dash-panel-title">Service account credentials</p>
        <p className="dash-note">
          Service accounts are the non-human principals. An API key bound to one is listed under
          API keys and carries its service account ID.
        </p>
        <ServiceAccountsBody />
      </div>
    </>
  );
}
