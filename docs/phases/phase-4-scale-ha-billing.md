# Phase 4 — Scale-out, HA, billing

**Status**: Not started — correctly deferred until Phase 1–3 give real
operational data (per-tenant actual resource usage, real customer count) to
size these decisions against, rather than guessing now.
**Goal**: production posture for real growth, not just "it works."
**Repos touched**: this repo (`infra/`, new platform-observability chart),
`management-plane/` (SSO, billing).
**Reference material**: `multi-tenant-design/my-nethesis-reference.md` in
`dev-nethsec-reference` is directly relevant to 4.3 and 4.6 below.

---

## 4.1 — Scale-out workers

- **Reimplement**: `infra/k3s/install-worker.sh` already handles this —
  "expansion is `install-worker.sh` on any new VM pointed at the same
  master, no change to these scripts needed" (per `infra/k3s/README.md`).
  Nothing new to build for the join itself.
- **Build new**: the one real task here is operational, not scripting — once
  a 3rd node exists, apply the master's `NoSchedule` taint (already decided
  in ADR 0001, already done early per its addendum) is confirmed present,
  and verify tenant pods actually schedule onto the new worker (check
  `kubectl get pods -o wide` across all tenant namespaces after the join,
  don't just trust the scheduler silently).

## 4.2 — HA control-plane (1 → 3 masters)

- **Study**: k3s's own docs for embedded-etcd HA mode (`--cluster-init` and
  joining additional server nodes) — this is generic k3s operational
  knowledge, no Nethesis-specific reference exists or is needed.
- **Build new**: a load balancer in front of the k8s API itself (the 3
  masters' `:6443`) — currently workers and any API clients point at one
  master's address per `infra/k3s/README.md`'s join instructions; this
  needs to become a VIP or a small nginx/HAProxy in front of all 3.
  Consider whether the existing VPS from ADR 0005 is the right place for
  this LB (consistent with its existing role) or whether a dedicated
  internal LB is cleaner — this decision affects whether a control-plane
  LB failure and a public-ingress failure become correlated, worth thinking
  through explicitly rather than defaulting to "reuse the VPS for
  everything."

## 4.3 — Proper SSO for the Management Plane

- **Study**: `NethServer/my`'s `DESIGN.md` in full — the token-exchange flow
  diagram at the top (`Vue Frontend → Logto IdP → Access Token → POST
  /auth/exchange → Go Backend → Custom JWT → Frontend`) is exactly the
  shape needed here: an external IdP handles login, the backend mints its
  own JWT embedding RBAC claims rather than passing the IdP's raw token
  around everywhere.
- **Reimplement** (pattern, not code — same AGPL caveat as Phase 2.2
  applies, `my`'s backend is AGPL-3.0): the token-exchange endpoint and JWT-
  embedding approach. IdP choice is open — `my` uses Logto, but nothing
  about this pattern requires that specific product; Keycloak, Authentik,
  or Zitadel are equally valid choices and should be evaluated on their own
  merits (self-hosting story, license, feature fit) rather than copying
  Nethesis's vendor choice by default.
- **Build new**: the RBAC role/permission model itself. `my`'s
  Owner→Distributor→Reseller→Customer hierarchy is *their* business
  hierarchy (Nethesis is the Owner of a multi-level channel business) — our
  own MSP structure may be simpler (likely just "MSP staff" with maybe
  Admin/Support-style technical roles, no reseller channel to model unless
  Nexwall itself grows a reseller program). Don't import `my`'s full org
  hierarchy wholesale; design our own roles against our actual org
  structure, using `my`'s *separation* of organization-role permissions
  from user-role permissions as the useful idea, not their specific role
  names.

**Acceptance criteria**: MSP staff have individual accounts (no more shared
`MGMT_API_KEY`), and every Management Plane action is attributable to a
specific person in an audit log.

## 4.4 — Billing integration

- **Build new**: entirely new — wire the `plan`/usage metadata already
  modeled in the Management Plane's Postgres store (Phase 2.2) to an actual
  billing provider (Stripe or a Brazil-specific provider given the
  `.com.br` domain and infrastructure already in place — worth checking
  what the existing `license.nexwall.com.br` subdomain, referenced in ADR
  0005, already integrates with, since that may already have a billing/
  payment relationship that should be reused rather than standing up a
  second one).
- **Study**: `NethServer/my`'s `entitlements.go` for the *concept* of tying
  subscription/plan state directly into the same system-of-record as
  tenant/system data, rather than a fully separate billing microservice —
  worth considering whether `firewall-msp`'s existing `nexwall-license`
  package (which already talks to `license.nexwall.com.br`) should be
  folded into the Management Plane's own data model at this point, now that
  a real Postgres store exists (Phase 2.2), instead of remaining a separate
  bespoke license server. This is an architecture decision worth its own
  ADR before implementation starts, not something to decide inline.

## 4.5 — Cost-tier revisit (shared-instance option, ADR 0002 consequence)

- **Study**: re-read ADR 0002's "Consequences" section — Option A (shared
  instance, logical tenancy, `tenant_id` column everywhere) was explicitly
  deferred, not rejected: "revisited at meaningful scale (50+ customers)...
  likely via A for cost-sensitive small customers, keeping B as a
  dedicated/enterprise tier." By Phase 4, real per-tenant cost data from
  Phases 1–3 should exist — use it to actually decide whether this
  threshold has been reached, rather than defaulting to "not yet" again.
- **Build new**: if pursued, this means the `tenant_id`-column-everywhere
  audit ADR 0002 explicitly avoided doing for `nexwall-controller`'s schema
  and routes becomes real work at this point — scope it as its own
  mini-project (likely its own set of ADRs) rather than a bullet in this
  phase doc when the time comes. Do not start this speculatively before the
  50+ customer threshold is actually approached.

## 4.6 — Platform observability (cluster-level, not per-tenant)

- **Study**: `NethServer/my`'s `services/mimir/` — this is the direct
  answer to the gap our own `ROADMAP.md` flags under "Explicitly not
  planned yet": *"Cross-tenant aggregated dashboards — no shared metrics
  store across tenants by design (ADR 0003); would need deliberate
  re-design if it becomes a real requirement."* Read `services/mimir/
  README.md` and `my.yaml` for their actual multi-tenant configuration
  (the `X-Scope-OrgID` header convention) before deciding this phase's
  approach.
- **Two distinct things are both called "observability" here — don't
  conflate them**:
  1. **Platform observability** (this task, as originally scoped): Loki/
     Prometheus/Grafana *for the cluster itself* — node health, k3s
     control-plane, Management Plane's own metrics. Separate from every
     tenant's own monitoring stack (already deployed per-tenant since
     Phase 1). No `my` reference needed for this half — it's standard k8s
     cluster monitoring, well-documented generically.
  2. **Cross-tenant aggregated dashboards** (the ADR 0003 gap, e.g. "total
     attacks blocked across all customers"): this is where Mimir's
     multi-tenancy is actually the relevant reference. If pursued, each
     tenant's existing per-namespace Prometheus (Phase 1's chart already
     deploys one per tenant) would `remote_write` into one shared Mimir
     cluster with each tenant's `tenant_id` as its `X-Scope-OrgID` — giving
     genuine cross-tenant queries without the Management Plane having to
     fan out and query N separate Prometheus/Grafana instances (the
     fallback ADR 0003 currently describes).
- **Build new**: whichever of the two above are pursued, the deployment of
  Mimir itself (or the fan-out query approach, if that's chosen instead) is
  new work either way — `my`'s Mimir usage proves the pattern works at
  Nethesis's scale, but the actual chart/config for our cluster is ours to
  build.
- **Decision needed before starting**: is cross-tenant reporting actually a
  real product requirement by this point, or still hypothetical? ADR 0003
  explicitly says revisit only "if/when this becomes a real product
  requirement" — don't build 4.6's second half speculatively.

## Order of execution

4.1 and 4.2 are infrastructure and can proceed independently of the rest.
4.3 should land before or alongside 4.4 (billing needs attributable users
for audit trails). 4.5 and 4.6's cross-tenant half both depend on a real
"is this needed yet" decision — don't schedule them by default, schedule
them when their trigger condition (50+ customers; a real cross-tenant
reporting request) actually occurs.

**Acceptance criteria (= Phase 4 exit criteria, unchanged)**: no single VM's
failure takes down more than its own workload; onboarding, billing, and
support are run by non-engineers through the Management Plane and a billing
dashboard, not through `kubectl`.
