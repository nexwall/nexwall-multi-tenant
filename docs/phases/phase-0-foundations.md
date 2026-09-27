# Phase 0 — Foundations (current)

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
