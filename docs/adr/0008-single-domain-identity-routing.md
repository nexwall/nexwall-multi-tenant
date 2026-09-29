# ADR 0008: Single-domain login, identity-based routing (not per-tenant subdomains)

**Status**: Accepted

## Context

ADR 0004 gave every tenant its own subdomain (`<slug>.painel.nexwall.com.br`), requiring a wildcard DNS record. The registrar in use (Locaweb) does not accept a wildcard on this domain — only a single `painel.nexwall.com.br` record.

Separately: the product goal is an MSP-facing "central" experience. Researched how Sophos Central actually does this (the explicit reference point), since assuming its mechanics without checking would be guessing: it is **one hostname** (`central.sophos.com`), login by email/password, and a `whoami` call returns the account's tenant ID, which scopes every subsequent API call (`X-Tenant-ID` header) — not a subdomain per customer. Worth noting for calibration: Sophos users openly ask for a smoother multi-tenant switcher (no "account picker" today, people resort to separate browser profiles) — so matching this bar is realistic, exceeding it on the account-switching UX is a real opportunity, not a stretch goal.

## Decision

Single public hostname (`painel.nexwall.com.br`, one A record). The Management Plane becomes the thing nginx forwards all traffic to — not a fixed tenant's Traefik. It owns:

1. **Identity directory**: `email -> [tenant_id]`, populated when a tenant is provisioned (`admin_email` in `TenantCreateRequest`, already in the OpenAPI contract) and, going forward, whenever a tenant adds a user (an integration point into `nexwall-controller`'s own Users feature, not yet built — tracked as an open item below).
2. **Login broker, not a credential store**: `POST /login` takes email+password, looks up the tenant(s) for that email, and **replays the same credentials against that tenant's own `/login`** (`nexwall-controller`'s existing bcrypt+gin-jwt code, unmodified). No password is ever stored in the Management Plane. On success it holds the tenant's JWT server-side, keyed by a session cookie.
3. **Dynamic reverse proxy**: once a session exists, every request is forwarded to that tenant's in-cluster Service (`tenant-<id>-<slug>-web.tenant-<id>-<slug>.svc.cluster.local:8080`), at the root path — not a URL prefix, so `nexwall-ui`'s asset paths need no change.
4. **Multi-tenant accounts**: if an email maps to more than one tenant, a picker is shown before proxying starts. Switching tenants later just replaces the session's target, no re-login needed if the same email/password already succeeded against both.

Per-tenant subdomains (ADR 0004) are not removed — the Ingress from ADR 0004/0007 still exists and still works as direct access (useful for debugging a single tenant, or a future enterprise tier that wants its own URL). It is just no longer the primary path a human uses to log in.

## Consequences

- Solves the Locaweb wildcard limitation outright — no CoreDNS/NS-delegation workaround needed (an approach that was proposed and then dropped in favor of this).
- The Management Plane goes from a provisioning API (Phase 2 as scoped) to also being a stateful session-holding reverse proxy — a meaningfully bigger piece of software than originally scoped. It needs a session store (in-memory is fine for one Management Plane replica; revisit before running more than one).
- Password changes, resets, and MFA all still happen inside each tenant's own `nexwall-controller` (unchanged) — the Management Plane never owns a password, only brokers the login call. This keeps `nexwall-controller` exactly as upstream, consistent with ADR 0002.
- **Open item**: today only the tenant's initial `admin_email` (set at provisioning) is known to the Management Plane's directory. A user created later inside a tenant's own Users page has no path into this directory yet — they could log into that tenant directly via its subdomain, but not through the single-domain portal. Needs either a webhook from `nexwall-controller` on user creation, or a periodic sync; not designed yet.
- **Open item**: nginx on the VPS currently forwards `painel.nexwall.com.br` straight to the master's Traefik (which today resolves by `Host`, matching only `demo.painel...`). It must be repointed at the Management Plane's own Service/Ingress instead — a config change, not yet made.
