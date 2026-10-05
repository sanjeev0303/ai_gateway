# Data Source Migration

## Overview
This document tracks the transition from the `lib/demo` mock data layer to the real PostgreSQL-backed API layer via `packages/api-client`. 

## Mock Data Removal
- All files under `app/src/lib/demo` have been completely disconnected from the dashboard pages.
- The `useDemoStore()` React hook has been removed from all data table and metrics dashboard components.
- Fabricated marketing statistics and metrics (e.g. 96.4% uptime, $112,785 cost savings) have been removed from the frontend and replaced with honest live counts from the platform APIs.
- Specifically, the Overview page now draws from the `/v1/usage/summary` and `/v1/usage/timeseries` endpoints.

## Real Data Adoption
- **Usage & Costs**: Populated via pre-aggregated SQL read models exposed on the `/v1/usage/*` surface.
- **Provider Registry**: The dashboard now lists real providers via `GET /v1/provider-accounts` and their associated health check states.
- **Model Registry**: Models are retrieved via `GET /v1/models` and synchronized with upstreams using `POST /v1/models/sync`.
- **RBAC & Organization**: Projects, teams, and members are natively managed through the `/v1/organizations`, `/v1/projects`, and `/v1/teams` endpoints, tied to the user's session token.

## Known Limitations
- The `govern/model-access` page correctly displays an empty state indicating that the feature is not yet fully plumbed to the backend. No mock data is shown here either.
- Certain list endpoints currently rely on frontend pagination; implementing cursor-based pagination is tracked for future phases.
