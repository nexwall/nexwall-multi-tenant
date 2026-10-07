# Management Plane

Three things live in this one Go binary, on purpose:

1. **`/_mgmt/*`** (except the one route below) — provisioning/admin API for MSP staff. Implements `docs/contracts/management-plane-openapi.yaml`. Auth: `MGMT_API_KEY` header (`X-Mgmt-Api-Key`).
2. **`/_mgmt/tenants/:id/handoff-secret`** — the one route `nexwall-partner-multitenant` is allowed to call, gated by a *different* key (`PARTNER_API_KEY` / `X-Partner-Api-Key`), so a leaked `MGMT_API_KEY` can't be used to read every tenant's SSO signing secret. See `docs/adr/0009-handoff-secret-provisioning.md`.
3. **`/login`, `/logout`, `/sso/handoff`, everything else (`/`, `/api/*`, ...)** — the customer-facing side: a login broker + session-keyed reverse proxy into whichever tenant a browser is logged into (`docs/adr/0008`). `/login` replays a password; `/sso/handoff` forwards a partner-minted token instead — both end up creating the same kind of session.

## A real bug found and fixed here (2026-10-04)

The first version of this file built the `/_mgmt` auth check like this:

```go
mgmt := r.Group("/_mgmt")
mgmt.Use(apiKeyCheck)              // attached to `mgmt`...
api.RegisterRoutes(r, h)           // ...but this created its OWN r.Group("/_mgmt") internally,
                                    // with no middleware at all
```

Two separate `gin.RouterGroup` objects happened to share a URL prefix; only one of them had the auth check, and it had zero routes on it. Every `/_mgmt/tenants` call — the entire provisioning API — was reachable with no API key, and every manual test during development passed anyway, because the key was always sent out of habit, so a *missing* check was never exercised.

Fixed by having `main.go` create the group (with middleware) and pass that same `*gin.RouterGroup` into `api.RegisterRoutes`, which no longer calls `r.Group` itself. If `MGMT_API_KEY` isn't set, the routes are now not mounted at all, rather than mounted-but-unprotected.

## Current state

**Provisioning (`/_mgmt`)**: `POST /_mgmt/tenants` allocates a `tenant_id`, VPN CIDR/port (ADR 0004), a subdomain, and a handoff secret (ADR 0009) — but does **not** yet call k8s. `internal/k8s/client.go` is a stub (`panic("not implemented")`) intentionally. Storage is in-memory (`internal/tenant/memstore.go`) — replace with Postgres before anything but local dev.

**Login/proxy/handoff (`internal/auth/`)**: validated end-to-end against the real `demo` tenant on the cluster — `/login` (password replay), session cookie, and the reverse proxy serving `/`, `/grafana/*`, etc. all through `painel.nexwall.com.br` with no subdomain. `/sso/handoff` is wired and compiles/vets clean but **has not been exercised with a real token** — `nexwall-partner-multitenant`'s minting side doesn't exist yet.

## Run locally

```
export MGMT_API_KEY=dev-key
export PARTNER_API_KEY=dev-partner-key
go run .

# provisioning (MGMT_API_KEY)
curl -H "X-Mgmt-Api-Key: dev-key" -X POST localhost:8090/_mgmt/tenants \
  -d '{"slug":"acme","display_name":"Acme Corp","plan":"trial","admin_email":"admin@acme.com"}'

# fetching the handoff secret (PARTNER_API_KEY, deliberately a different one)
curl -H "X-Partner-Api-Key: dev-partner-key" localhost:8090/_mgmt/tenants/1/handoff-secret
```

## Next real steps

1. Wire `internal/k8s.CreateTenantNamespace` + `InstallOrUpgradeRelease` into `createTenant`, passing `handoffSecret` through to the Helm chart's `handoffSecret` value (already wired on the chart side).
2. Get a real handoff token from `nexwall-partner-multitenant` once its minting side exists, and prove `/sso/handoff` end-to-end the same way `/login` already is.
3. Replace the in-memory `Store`/`Sessions` before running more than one replica.
