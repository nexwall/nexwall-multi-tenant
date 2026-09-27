# Roadmap

Detailed per-phase docs live in `docs/phases/`. This is the one-page summary.

| Phase | Goal | Key exit criterion |
|---|---|---|
| 0 — Foundations | Repo scaffold exists | This commit |
| 1 — Cluster + Helm | k3s on 2 VMs, one tenant deployed by hand | A real firewall registers end-to-end through k3s |
| 2 — Management Plane MVP | Provision/suspend/delete tenants via API, not `helm install` by hand | One API call onboards a customer |
| 3 — Group policies | Close the NethSecurity gap: push firewall config to a group of units (tracked upstream, unimplemented: `NethServer/nethsecurity#1895`) | One action changes config on 10 units at once |
| 4 — Scale, HA, billing | Production posture | No single VM failure takes more than its own workload down |

## Explicitly not planned yet

- Shared-instance (non-isolated) tenancy for a cheap tier — revisit after Phase 1 gives real per-tenant cost data (see ADR 0002's consequences section).
- Cross-tenant aggregated dashboards — no shared metrics store across tenants by design (ADR 0003); would need deliberate re-design if it becomes a real requirement.
- IaC for the VMs themselves (Terraform/Pulumi) — the 2 starting VMs are provisioned outside this repo.
