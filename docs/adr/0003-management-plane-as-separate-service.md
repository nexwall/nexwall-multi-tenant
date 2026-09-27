# ADR 0003: Management Plane as a new, separate service

**Status**: Accepted

## Context

With isolated-per-tenant controllers (ADR 0002), nothing today provisions them, tracks which exist, or gives the MSP a cross-customer view. This capability doesn't exist in `nexwall-controller` and can't be bolted onto it without violating ADR 0002's isolation goal.

## Decision

Build a new, small Go service — `management-plane/` — that talks to the k8s API (client-go), not to any tenant's controller API directly. It owns tenant lifecycle (create/suspend/delete) and aggregates health/status by querying the k8s API for each tenant's namespace, not by calling into tenant databases.

Its own state (tenant registry, plan/billing metadata) lives in its own database — separate from every tenant's TimescaleDB, consistent with ADR 0002.

## Consequences

- Clean separation: `nexwall-controller` never needs to know it's being run multi-tenant. It stays exactly what it is upstream.
- The Management Plane's API is the actual product surface for "central management like Sophos Central" — see `docs/contracts/management-plane-openapi.yaml`. This is the highest-leverage new code in the whole project.
- Cross-tenant reporting/dashboards (e.g. "total attacks blocked across all customers") requires the Management Plane to query each tenant's Grafana/Prometheus individually and aggregate — there is intentionally no shared metrics store across tenants in Phase 1–2. Revisit if/when this becomes a real product requirement.
