# Phase 0 — Foundations

**Status**: Done. Superseded by Phase 1, which is now in progress.
**Goal**: repo scaffold exists, nothing deployed yet.

## Done
- [x] Architecture decided and recorded (`docs/adr/0001`–`0004`)
- [x] Repo structure created: Helm chart skeleton, Management Plane skeleton, infra scripts, CI skeletons
- [x] Management Plane API contract drafted (`docs/contracts/management-plane-openapi.yaml`)

## Not done (blocking Phase 1)
- [ ] 2 VMs provisioned (owner: you — infra not yet set up per this conversation)
- [ ] DNS: wildcard `*.painel.<domain>` pointed at the future Ingress IP
- [ ] GitHub repo created and this scaffold pushed (owner: you)
- [ ] Container registry access confirmed from both VMs (`ghcr.io/nexwall/*` — already proven reachable from the WAN VM in `nexwall-controller`'s deployment)

## Exit criteria
Both VMs reachable by SSH, DNS wildcard resolving, repo pushed to GitHub with CI passing on the skeleton (lint-only at this stage, nothing to test yet).

## Handoff to Phase 1

All blocking items above were resolved during real deployment — see `docs/adr/0001-orchestrator-k3s.md`'s addendum and `infra/k3s/README.md`'s "Real deployment notes" for what actually happened (private-network node-IP requirement, the taint-does-not-survive-a-restart gotcha, and the WireGuard/nginx public entry point from ADR 0005 that wasn't anticipated in this phase's original scope). Read those before starting Phase 1 work, not just this file.
