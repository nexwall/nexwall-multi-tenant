# ADR 0002: Tenancy model — one full controller stack per customer

**Status**: Accepted

## Context

Two ways to serve multiple customers from `nexwall-controller`:

**A. Shared instance, logical tenancy** — one controller, one database, a `tenant_id` column added everywhere, every query filtered by it.

**B. Isolated instance per tenant** — each customer gets their own full stack (vpn, db, api, ui, proxy, monitoring) in its own k8s Namespace, on shared VMs.

## Decision

**B.** Confirmed directly in the upstream `ns8-nethsecurity-controller` README: *"Each node can host multiple controller instances"* — this is an already-intended deployment pattern, not something we're forcing onto the software.

Deciding factor: `nexwall-controller`'s schema and Go routes have **zero** tenant-scoping today (verified against `api/models/unit.go`, `api/main.go` — no `tenant_id` anywhere, `UnitGroup` is RBAC-only, scoped to users not tenants). Option A means auditing and rewriting every query in the API before shipping a single paying customer. Option B uses the code exactly as it exists.

## Consequences

- **Isolation**: a bug in one tenant's controller cannot leak another tenant's data — there is no shared table to leak from. Blast radius of a compromise is one customer.
- **Cost**: ~10 containers and their resource requests per customer, not shared. At meaningful scale (50+ customers) this is revisited — likely via A for cost-sensitive small customers, keeping B as a "dedicated/enterprise" tier. Out of scope until Phase 4+.
- **The Management Plane (see ADR 0003) becomes mandatory**, not optional — with N independent controller databases and no shared tenant table, there is no other way to get an MSP-wide view across customers.
