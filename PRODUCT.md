# Tollseal

Tollseal is an AI gateway and governance console. Applications call one gateway
endpoint with a Tollseal key (an `aigw_` gateway key); the gateway authenticates the key, checks the
request against the organization's guardrails and policies, applies rate limits
and budgets, forwards it to the provider that serves the model, and records the
request: tokens, cost, latency, outcome, and (by privacy mode) the conversation.

## Who uses it

- **Platform and security owners** (the account owner, admins). They connect
  provider accounts, write policies and guardrails, set budgets, issue and
  rotate gateway keys, and read audit logs and security events.
- **Finance and team leads.** They read usage and cost by team, project, model
  and person, and watch budgets.
- **Employees and services.** They never open the console for their daily
  work; they call the gateway through the TypeScript SDK, the `aigw` CLI, or the
  OpenAI- and Anthropic-style REST endpoints with the key they were given.

The console is used at a desk, in long sessions, in light and in dark rooms.

## What is true about the product

- Provider kinds: OpenAI, Anthropic, Google Gemini, Azure OpenAI, Hugging Face,
  and custom OpenAI-compatible endpoints (Mistral connects this way).
- Policy rules: model allowlists and blocklists, rate limits, spend budgets and
  token budgets, scoped to the organization, a team, a project, a key or a
  service account.
- Guardrails: PII (email, phone, SSN), secrets and keys, blocked keywords and
  request-size limits, each set to block, redact or warn.
- Budgets page: spend, token and request caps on an hourly, daily, weekly or
  monthly window.
- Observability: live activity, usage, costs, models, providers, users, teams,
  projects and recorded conversations.
- Security: audit logs, security events and alert rules.
- Self-hosted: a Go gateway, control API and worker with Postgres, Redis and
  NATS, run as containers.

There are no published customers, traffic totals or certifications. Pages must
not claim any.

## Brand

Name: Tollseal (renamed from Sentinel; tollseal.com was unregistered when chosen).
Every provider sits behind one gateway key, and every request passes the toll
and leaves sealed. The interface reads as security printing: cool security
paper, intaglio indigo ink, banknote green for the one primary action, a foil
kept for the prism and the seal, guilloche linework, serial numbers and
microprint. It is positioned as the hub teams connect many providers to and call
through with one key (CLI, SDK, REST), not only as a security product.

Not yet renamed outside the web app: the backend, the `aigw` CLI (which still
accepts `SENTINEL_*` environment variables as a fallback to `AIGW_*`), the docs,
and the internal cookie and storage keys.
