# Phase 2 — Management Plane MVP

**Status**: In progress. Routes are wired (`management-plane/internal/api/handlers.go`),
allocation logic is real (`internal/tenant/model.go` implements ADR 0004's
scheme correctly), but storage is in-memory (`internal/tenant/memstore.go`)
and every k8s-touching method in `internal/k8s/client.go` is a literal
`panic("not implemented")` stub. Nothing deploys to k8s yet — `createTenant`
today only reserves an allocation.
**Goal**: tenants created/suspended/deleted via API instead of `helm install`
by hand.
**Repos touched**: this repo only (`management-plane/`).
**Reference material**: `multi-tenant-design/my-nethesis-reference.md` in
`dev-nethsec-reference` — read this before starting 2.2, it's directly
relevant to the data model below.

---

## 2.1 — Wire `internal/k8s` for real

This is the single change the `management-plane/README.md` itself names as
"the next real step" — it turns the service from "reserves an allocation"
into "actually provisions a customer."

- **Study**: `client-go`'s own examples for in-cluster clients (the
  `NewInClusterClient` stub already assumes this pattern and has the
  ServiceAccount/RBAC scope already decided — see `infra/k3s/README.md`
  step 5 for the exact ClusterRole needed: `namespaces`, `deployments`,
  `statefulsets`, `services`, `secrets`, `persistentvolumeclaims`,
  `ingresses`). Also study the Helm Go SDK
  (`helm.sh/helm/v3/pkg/action`) — specifically `action.Install` and
  `action.Upgrade`, since `InstallOrUpgradeRelease`'s stub signature
  already implies both need to be handled (idempotent install-or-upgrade).
- **Build new**: all three method bodies currently panicking:
  - `CreateTenantNamespace` — Namespace + ResourceQuota + NetworkPolicy.
    The chart's `values.yaml` `resources:` block already has real per-plan
    numbers from Phase 1's pilot; the ResourceQuota should sum these,
    not invent new numbers.
  - `InstallOrUpgradeRelease` — render `values.yaml` from a `tenant.Tenant`
    struct (already has every field `values.schema.json` requires:
    `tenantId`, `slug`, `vpnCidr`, `vpnPort`, `subdomain`), call
    `action.Install` if the release doesn't exist yet, else
    `action.Upgrade`.
  - `GetTenantStatus` — list Pods/Deployments in the tenant's namespace via
    `clientset.AppsV1().Deployments(ns).List(...)`, map to the
    `DeploymentStatus` shape already defined, matching the
    `TenantStatus`/`components` shape already specified in
    `docs/contracts/management-plane-openapi.yaml`.
  - No existing reference for any of this in `NethServer/my` or upstream
    NethSecurity — neither manages Kubernetes deployments of anything, this
    capability is entirely Nexwall-specific (per ADR 0003, this is
    explicitly "the highest-leverage new code in the whole project").

- **Wire into handlers**: replace the three `TODO(phase-2)` comments in
  `internal/api/handlers.go` (`createTenant`, `deleteTenant`,
  `suspendTenant`) with real calls into the new `k8s.Client` methods. Order
  matters, per the comments already left there: on delete, trigger
  namespace deletion *before* removing the store row (never orphan a
  running namespace); on suspend, scale to 0 *before* flipping status
  (status must reflect reality, not intent).

**Acceptance criteria**: `POST /tenants` results in a real, running pilot
stack reachable at its subdomain within the same request/response cycle (or
async with a `provisioning` → `active` status transition, whichever is
simpler for MVP — polling `GET /tenants/:id/status` until `healthy: true`
is acceptable for Phase 2, don't over-engineer webhooks/events yet).

---

## 2.2 — Postgres-backed store, replacing `MemStore`

- **Study**: `NethServer/my`'s `backend/entities/local_customers.go`,
  `local_systems.go`, and especially `local_systems_suspend_test.go` for
  the **pattern** of how they model a suspend-with-retained-data operation
  against a real schema — including their soft-delete convention
  (`deleted_at IS NULL` filtering, seen directly in
  `collect/methods/rebranding.go`'s `getSystemOrgID` query). Note the
  license flag from the reference doc: **read this for the pattern, do not
  copy the code** — `my`'s backend is AGPL-3.0, ours is GPL-3.0, and this
  is a network service.
- **Reimplement** (pattern, not code): a `tenants` table matching
  `tenant.Tenant`'s existing fields, but consider adopting `my`'s
  soft-delete pattern (`deleted_at` column) instead of the OpenAPI
  contract's current hard-`DELETE`. This is worth a deliberate decision,
  not a silent change — hard-delete matches what's documented today in
  `docs/contracts/management-plane-openapi.yaml`
  (`DELETE /tenants/{tenantId}` → "Permanently delete"); soft-delete would
  need a contract update first (add a `deleted_at` field to the `Tenant`
  schema, decide whether `GET /tenants` should exclude soft-deleted rows by
  default). Recommend soft-delete for auditability — a suspended tenant
  that's later hard-deleted loses the `tenant_id`/network-allocation
  history that ADR 0004 explicitly cares about not reusing.
- **Build new**: implement the same `Store` interface already defined in
  `internal/api/handlers.go` against Postgres (`database/sql` +
  `lib/pq`/`pgx`, or an ORM if the team prefers — nothing here needs to
  match `my`'s tooling choices) instead of the map+mutex in `memstore.go`.
  Migrations: a `migrations/` folder with plain numbered SQL files is
  enough for this scale — `my`'s `backend/database/migrations/` is a
  reasonable structural reference for naming conventions only.
- **Acceptance criteria**: `MemStore` deleted entirely (not left as a
  fallback — `main.go`'s `tenant.NewMemStore()` call replaced with a real
  DB-backed store), a restart of the Management Plane process does not lose
  any tenant.

---

## 2.3 — Cert secret provisioning hook

Carried over from Phase 1.2's TODO: whichever mechanism was chosen there
(Helm post-install hook or reflector) needs to actually fire as part of
`CreateTenantNamespace` (2.1), not as a separate manual step. If reflector
was chosen in 1.2, this task may already be done automatically — verify,
don't assume.

---

## 2.4 — Auth: static API key (deliberately minimal)

**Status: done, already in `main.go`.** A single `MGMT_API_KEY` env var,
checked in middleware. Nothing to build here for Phase 2 — explicitly
adequate for internal operators indefinitely (ADR 0006 moved partner-
facing SSO/RBAC to `nexwall-partner-multitenant`). Do not scope-creep
accounts/RBAC into this service. One addition IS required before the
Partner Program integrates: support a second, dedicated API key (e.g.
`PARTNER_API_KEY`) so the Partner Program never shares the key operators
use by hand.

---

## 2.5 — Pilot customers through the API

- 2–3 real pilot customers created **through this API**, not by hand via
  `helm install` (which was Phase 1's method) — this is the actual proof
  that 2.1–2.3 work end-to-end together, not just individually.

**Acceptance criteria (= Phase 2 exit criteria, unchanged)**: an MSP
operator onboards a new customer with one API call (or one CLI command
wrapping it), sees all customers' status in one `GET /tenants` list, and
suspends a non-paying customer without touching `kubectl` directly.

## Carries into later phases

- The `Store` interface from 2.2, once Postgres-backed, is the natural home
  for any plan metadata `management-plane` itself needs; billing and
  entitlements live in `nexwall-partner-multitenant` (ADR 0006).
- 2.1's namespace-per-tenant provisioning is the foundation Phase 3's
  `helm upgrade` rollout (for the group-policy feature, once built in
  `nexwall-controller`) reuses unchanged.
