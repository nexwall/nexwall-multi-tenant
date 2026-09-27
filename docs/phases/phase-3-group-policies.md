# Phase 3 — Group-level firewall policy push

**Goal**: close the real product gap found during evaluation — NethSecurity's own controller has no way to push a policy to a group of units (confirmed: not in `nexwall-controller`'s code, not in upstream, and openly tracked as unimplemented in `NethServer/nethsecurity#1895`).

This work happens **inside `nexwall-controller`** (the API/UI repo), not here — this repo only needs to package and deploy it once it exists. Tracked here because it's the single highest-value differentiator identified so far and the roadmap would be incomplete without it.

## Work items (in `nexwall-controller`)
1. New `Policy` model, associated with `UnitGroup` (which already exists, currently RBAC-only — see the controller's `api/models/unit.go`).
2. Push mechanism: reuse the existing VPN tunnel + the same authenticated channel already used for SSH key distribution and remote updates (`Actions → Update systems` today is the closest existing analog — a per-unit action triggerable in bulk; generalize it to accept an arbitrary config payload).
3. UI: a "Policies" view under Unit Groups, matching the pattern of the existing `UnitGroupsView.vue`.

## Work items (in this repo)
- No infra changes required — this ships as a `nexwall-controller`/`nexwall-ui` image update, rolled out via `helm upgrade` per tenant, same as any other version bump.

## Exit criteria
An MSP operator applies one firewall rule change to a group of 10 units in one action, from one tenant's controller UI.
