# ADR 0006: Partner/reseller business layer is a separate service, not part of Management Plane

**Status**: Accepted

## Context

Evaluating `NethServer/my` (Nethesis's own account/billing/RBAC/reseller-
hierarchy system, see `dev-nethsec-reference/multi-tenant-design/
my-nethesis-reference.md`) surfaced a real question: does `management-plane`
grow to include multi-level reseller hierarchy, billing, and partner SSO —
or does that stay out of scope, as Phase 2's OpenAPI contract currently
assumes (flat tenant list, static API key, no org hierarchy)?

## Decision

Two separate systems, on separate clusters:

- **`nexwall-multi-tenant`** (this repo) stays exactly as scoped in Phases
  0–3: infrastructure orchestration only. `management-plane` provisions/
  suspends/deletes isolated `nexwall-controller` stacks for direct
  customers. No reseller concept, no billing, no partner-facing auth. Its
  existing static-API-key auth (Phase 2.4) is fine indefinitely for this
  narrower scope — it's an internal tool for Nexwall's own operators, not a
  partner-facing product.
- **`nexwall-partner-multitenant`** (new repo, new cluster) — the "Nexwall
  Partner Program," serving `partner.nexwall.com.br`. This is where a
  `my`-inspired business layer actually belongs: Distributor → Reseller →
  Customer hierarchy, entitlements, billing, partner-facing SSO/RBAC.
  Partners log into this system directly.

## Why this isn't just "move Phase 4's SSO/billing into a new folder"

`my`'s own architecture is comparatively passive: Nethesis's products run on
*customers' own hardware*, so `my` mostly records inventory/heartbeat from
units that phone home to it — it never provisions infrastructure. **We host
every customer's controller stack ourselves**, so `nexwall-partner-
multitenant` cannot just be a bookkeeping copy of `my` — when a partner
onboards a new end customer, that action must **call this repo's Management
Plane API** (`POST /tenants`, per `docs/contracts/management-plane-
openapi.yaml`) to actually provision the real stack in this cluster. That
cross-cluster call is new design surface with no equivalent in `my` at all,
and needs its own service-to-service auth (distinct from partner-facing
human auth, and distinct from `management-plane`'s current static
`MGMT_API_KEY`, which was never meant to be handed to another service).

## Consequences

- `docs/phases/phase-4-scale-ha-billing.md` in this repo no longer includes
  SSO or billing as tasks — moved to `nexwall-partner-multitenant`'s own
  roadmap. This repo's own "proper SSO" need (if any — an internal ops team
  is a much smaller user base than a partner network) is now optional, not
  a hard Phase 4 requirement.
- `management-plane`'s `POST /tenants` becomes a semi-public API surface
  it wasn't originally designed as — called by a second service, not only
  used by hand/CLI. Its current static-API-key auth should be scoped to a
  dedicated key issued to `nexwall-partner-multitenant` specifically
  (rotatable independently of any human operator's key), not the same key
  a person types by hand — worth doing before the Partner Program's first
  real call, not deferred to a later hardening pass.
- Two clusters means two sets of infra operational overhead (two k3s
  bootstraps, two sets of ADR-0001/0005-style network decisions) —
  accepted, since the two systems have genuinely different audiences
  (internal ops vs. external partners) and different security postures
  (an external-facing partner portal is a materially different attack
  surface than an internal provisioning tool, and isolating them limits
  blast radius the same way ADR 0002 isolates customer tenants from each
  other).
