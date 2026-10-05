# ADR 0009: Handoff secret provisioning — generated here, consumed by two other repos

**Status**: Accepted

## Context

`nexwall-controller`'s `docs/adr/0001-sso-handoff-from-partner-program.md` (implemented 2026-10-04) verifies partner-originated login tokens with a per-tenant HMAC secret, read from `SSO_HANDOFF_SECRET`. `nexwall-partner-multitenant`'s `docs/adr/0006` mints those tokens, needing the *same* secret for the *same* tenant. Nothing generates or distributes this secret yet. It has to live somewhere that both sides can reach, and that is this repo: the Management Plane already owns tenant provisioning (ADR 0003) and already renders this tenant's Helm values (ADR 0007's `secrets.yaml`).

## Decision

1. **Generated once, at `POST /_mgmt/tenants`**, using `nexwall-multitenant-sso/handoff.GenerateSecret()` — the same package both consuming repos already import, so there's no risk of a format mismatch invented independently on either side.
2. **Stored in the Tenant record, never returned by `GET /_mgmt/tenants` or `GET /_mgmt/tenants/:id`** — those responses omit it entirely. A dedicated `GET /_mgmt/tenants/:id/handoff-secret` exists for the one legitimate reason to read it back: `nexwall-partner-multitenant` fetching it to mint a token. Gated by a separate `PARTNER_API_KEY`, distinct from the general `MGMT_API_KEY` admin key (matching the distinction `nexwall-partner-multitenant`'s own ADR 0003 already asks for).
3. **Passed into the tenant's Helm values as `handoffSecret`**, rendered to `SSO_HANDOFF_SECRET` in `secrets.yaml` — the tenant's own `nexwall-controller` never has to be told about this out of band.
4. **The Management Plane's own customer-facing side gains `GET /sso/handoff`** (distinct from `/login`): takes `?token=...&tenant=<slug>`, looks up that tenant, forwards the token verbatim to that tenant's in-cluster `/sso/handoff`, and on success wraps the returned JWT in a normal session exactly like `/login` does after a password replay (ADR 0008). This is what `nexwall-partner-multitenant` actually redirects a browser to — it never talks to a tenant's cluster Service directly.

## Consequences

- Three repos now share one secret per tenant, generated in exactly one place. Rotation means re-generating here and re-deploying the tenant's Helm release — not yet automated, tracked as an open item.
- `/sso/handoff`'s forwarding step means the Management Plane briefly holds a tenant JWT in a request-scoped variable, same as it already does for password-replay logins — no new trust boundary, just a second way to arrive at the same internal state (a `Session`).
- `PARTNER_API_KEY` is a second static shared key, same weakness as `MGMT_API_KEY` (ADR 0003's open item about proper auth is unresolved for both, not just one).
