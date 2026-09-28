# Phase 4 — Scale-out and HA

(Filename keeps `billing` for link stability; billing/SSO moved out, see ADR 0006.)

**Status**: Not started — correctly deferred until Phase 1–3 give real
operational data (per-tenant actual resource usage, real customer count) to
size these decisions against, rather than guessing now.
**Goal**: production posture for real growth, not just "it works."
**Repos touched**: this repo only (`infra/`, new platform-observability chart).
**Reference material**: `multi-tenant-design/my-nethesis-reference.md` in
`dev-nethsec-reference` is directly relevant to 4.4 below.

> **Scope change (ADR 0006)**: SSO and billing are no longer part of this
> repo. They belong to `nexwall-partner-multitenant` (the Partner Program,
> `partner.nexwall.com.br`, separate cluster). See the section "Moved out of
> this phase" below.

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

## Moved out of this phase

**SSO and billing** (previously 4.3 and 4.4) now live in
`nexwall-partner-multitenant`, per `docs/adr/0006-partner-program-separate-
service.md`. What stays relevant to this repo:

- `management-plane`'s static `MGMT_API_KEY` remains adequate for internal
  operators. Optional hardening (individual operator accounts + audit log)
  is a nice-to-have here, not a phase exit requirement.
- **Required before the Partner Program's first real call**: issue a
  dedicated, independently rotatable API key to `nexwall-partner-
  multitenant` — do not reuse the key operators type by hand. This is a
  small task (env var + middleware already exist), but it must land before
  integration, not after.
- The `nexwall-license` package / `license.nexwall.com.br` question (fold
  into a Postgres-backed store vs. stay a separate service) is now a
  Partner Program decision, since entitlements live there.

## 4.3 — Cost-tier revisit (shared-instance option, ADR 0002 consequence)

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

## 4.4 — Platform observability (cluster-level, not per-tenant)

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

4.1 and 4.2 are infrastructure and can proceed independently. 4.3 and 4.4's
cross-tenant half both depend on a real "is this needed yet" decision — don't
schedule them by default; schedule them when their trigger condition (50+
customers; a real cross-tenant reporting request) actually occurs.

**Acceptance criteria (= Phase 4 exit criteria)**: no single VM's failure
takes down more than its own workload. Onboarding and billing are handled by
the Partner Program (`nexwall-partner-multitenant`), which calls this
repo's Management Plane API — operators no longer use `kubectl` for either.
