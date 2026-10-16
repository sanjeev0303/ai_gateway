# Sentinel AI Gateway — Known Limitations & Operational Considerations

**Date**: 2026-10-15  
**Scope**: Local Production-Readiness Environment (Excluding Cloud Infrastructure / CI/CD)  

---

## 1. Overview

While Sentinel runs as a fully functioning, multi-tenant system locally with complete database persistence, asynchronous telemetry ingestion, and real-time governance, several architectural distinctions exist between the local single-host environment and an enterprise cloud deployment.

---

## 2. Key Differences: Local vs. Production

### 2.1 Provider Integrations & Offline Simulation
- **Synthetic Keys**: Local test suites and seed data utilize deterministic dummy API keys (`sk-dummy-*`) to simulate upstream LLM providers (Anthropic, OpenAI, Gemini).
- **Network Isolation**: When offline or running without live provider credentials, model probe endpoints simulate success for known dummy keys and return `401 Unauthorized` for `sk-invalid-*`.
- **Live Upstream Requirements**: To route live requests through actual external providers (OpenAI, Anthropic, Google), valid external API keys must be configured in `platform/provider-accounts`.

### 2.2 Redis Caching & Invalidation
- **Single Node Redis**: The local Docker compose setup runs a single Redis 7 instance. Production environments require a Redis Sentinel or Redis Cluster deployment to provide high availability.
- **Cache Pre-warming**: On initial cold boot, the local gateway loads active policies on demand per tenant rather than maintaining a pre-warmed snapshot across all organizations.

### 2.3 Rate Limiter Precision & Sliding Windows
- **Local Clock Skew**: The Redis Lua rate-limiting script relies on `redis.replicate_commands()` and system time. In local environments with host machine sleep/wake cycles, timestamps may temporarily jump, resulting in brief rate-limit reset discrepancies.
- **Window Granularity**: Local rate limits use 1-minute sliding windows. Extremely high burst traffic (>1,000 req/sec) on a single local core may experience CPU throttling before reaching Redis capacity.

### 2.4 Database Connection Pool & Concurrency
- **Pool Bounds**: Local PostgreSQL is configured with `max_connections = 100`. The backend Go services (`control-api`, `gateway-v2`, `worker`) are configured with bounded connection pools (max 25-50 connections each).
- **Connection Saturation**: Running parallel load tests with over 100 concurrent workers against the local PostgreSQL container may cause transient `connection refused` or `pool exhausted` errors if pool sizes are not tuned to local hardware limits.

### 2.5 Message Bus (NATS JetStream)
- **Local Persistence**: NATS is configured to use memory or local disk storage in the container. Container removal (`docker compose down -v`) clears unconsumed JetStream queues unless mounted volumes are preserved.
- **Worker Concurrency**: The local worker runs with a single process consumer pool. In production, worker instances can be horizontally scaled using NATS queue groups.

### 2.6 Next.js Server Components & Cookies
- **Local Static Generation**: Certain Next.js 16 pages use dynamic request cookies (`session.ts`). During build time, Next.js logs dynamic server usage warnings for pages reading headers/cookies outside request scope, which is expected Next.js 16 behavior for dynamic dashboard routes.
- **Reverse Proxy Headers**: The local stack runs without an external ingress (e.g., NGINX, Cloudflare). Client IP addresses are derived directly from Docker network interfaces (`172.x.x.x`).

---

## 3. Production Deployment Prerequisites (Out of Scope for Local Readiness)

When transitioning Sentinel from local readiness to cloud production, the following infrastructure components must be established:
1. **Managed Secret Storage**: Integration with AWS Secrets Manager, HashiCorp Vault, or GCP Secret Manager for encryption at rest of provider master keys.
2. **Horizontal Autoscaling (HPA)**: Kubernetes deployment manifests for stateless `gateway-v2` pods and `control-api` pods with CPU/latency-based autoscaling.
3. **Managed Database & Cache**: Amazon RDS Aurora PostgreSQL (Multi-AZ) and Amazon ElastiCache Redis Cluster.
4. **Distributed Tracing & APM**: OpenTelemetry Collector export to Datadog, Prometheus/Grafana, or Google Cloud Trace.
5. **TLS Ingress & WAF**: Cloudflare or AWS ALB with SSL termination, DDoS protection, and IP reputation filtering.
