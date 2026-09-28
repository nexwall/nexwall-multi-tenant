# Roadmap

Detailed per-phase docs live in `docs/phases/`. This is the one-page summary.

| Phase | Status | Goal | Key exit criterion |
|---|---|---|---|
| 0 — Foundations | Done | Repo scaffold exists | This commit |
| 1 — Cluster + Helm | In progress | k3s on 2 VMs, one tenant deployed by hand | A real firewall registers end-to-end through k3s |
| 2 — Management Plane MVP | In progress | Provision/suspend/delete tenants via API, not `helm install` by hand | One API call onboards a customer |
| 3 — Group policies | Not started | Close the NethSecurity gap: push firewall config to a group of units (tracked upstream, unimplemented: `NethServer/nethsecurity#1895`) | One action changes config on 10 units at once |
| 4 — Scale and HA | Not started | Production posture | No single VM failure takes more than its own workload down |

Each phase doc in `docs/phases/` now includes, per task, what to **study**
in existing code (ours or upstream Nethesis/NethSecurity), what to
**reimplement** (an existing pattern, adapted), and what has **no
reference and must be built new** — written for someone picking up
implementation directly. Background research behind several of these
(the `NethServer/my` analysis behind Phase 2's data model and Phase 4's
Mimir section, the upstream backlog tracked for Phase 3) lives in the
separate `dev-nethsec-reference` repo, linked from each phase doc where
relevant.

## Explicitly not planned yet

- Shared-instance (non-isolated) tenancy for a cheap tier — revisit after Phase 1 gives real per-tenant cost data (see ADR 0002's consequences section).
- Cross-tenant aggregated dashboards — no shared metrics store across tenants by design (ADR 0003); would need deliberate re-design if it becomes a real requirement.
- IaC for the VMs themselves (Terraform/Pulumi) — the 2 starting VMs are provisioned outside this repo.

## Related system

The partner/reseller business layer (hierarchy, entitlements, billing,
partner SSO) is a separate project: `nexwall-partner-multitenant`
(`partner.nexwall.com.br`, its own cluster). See
`docs/adr/0006-partner-program-separate-service.md`.
