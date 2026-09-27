# Phase 4 — Scale-out, HA, billing

**Goal**: production posture for real growth, not just "it works."

## Work items
1. **Scale-out workers**: additional VMs joined as `k3s agent`s; the master gets `NoSchedule` tainted once ≥3 nodes exist, so control-plane never competes with tenant workload for resources.
2. **HA control-plane**: move from 1 master to 3 (k3s embedded etcd HA mode), fronted by a load balancer for the k8s API itself.
3. **Proper SSO** for the Management Plane (replacing Phase 2's static API keys) — OIDC, so MSP staff have individual accounts and audit trails.
4. **Billing integration**: plan/usage metadata already modeled in the Management Plane's datastore (Phase 2) gets wired to an actual billing provider.
5. **Cost-tier revisit** (ADR 0002 consequence): for small/free-tier customers, evaluate the shared-instance model (Option A from ADR 0002) as a cheaper tier alongside the isolated tier, now that the isolated model is proven and the team understands the real per-tenant cost.
6. **Platform observability**: Loki/Prometheus/Grafana *for the cluster itself* (node health, k3s control-plane, Management Plane) — separate from each tenant's own monitoring stack (already deployed per-tenant since Phase 1).

## Exit criteria
No single VM's failure takes down more than its own workload; onboarding, billing, and support are run by non-engineers through the Management Plane and a billing dashboard, not through `kubectl`.
