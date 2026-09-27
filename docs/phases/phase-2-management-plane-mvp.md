# Phase 2 — Management Plane MVP

**Goal**: tenants created/suspended/deleted via API instead of `helm install` by hand.

## Work items
1. Implement `POST /tenants` in `management-plane/`: allocates the next `tenant_id` (see ADR 0004), renders `values.yaml` from the Helm chart's `values.schema.json`, calls Helm via its Go SDK (`helm.sh/helm/v3/pkg/action`), creates the Namespace first with the correct `ResourceQuota`/`NetworkPolicy`.
2. Implement `GET /tenants`, `GET /tenants/:id/status` (pod health via client-go), `POST /tenants/:id/suspend` (scale replicas to 0), `DELETE /tenants/:id`.
3. Management Plane's own datastore (Postgres, its own instance — not a tenant's TimescaleDB) for the tenant registry + plan metadata.
4. Basic auth/SSO for MSP operators (who is allowed to call this API at all) — deliberately minimal in this phase (static API keys are fine); proper SSO is Phase 4.
5. 2–3 real pilot customers created through this API instead of by hand.

## Exit criteria
An MSP operator can onboard a new customer with one API call (or one CLI command wrapping it), see all customers' status in one list, and suspend a non-paying customer without touching `kubectl` directly.
