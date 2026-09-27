# Management Plane

Implements `docs/contracts/management-plane-openapi.yaml`. See `docs/adr/0003-management-plane-as-separate-service.md` for why this exists as its own service.

## Current state (Phase 2, in progress)

- `POST /api/tenants` allocates a `tenant_id`, VPN CIDR/port (`docs/adr/0004`), and a subdomain — but does **not** yet call k8s. `internal/k8s/client.go` is a stub (`panic("not implemented")`) intentionally, so wiring it up is a clear, isolated next step.
- Storage is in-memory (`internal/tenant/memstore.go`) — replace with Postgres before anything but local dev.
- No real auth — a single shared `MGMT_API_KEY` env var, checked by a middleware in `main.go`. Fine for one operator; not fine once more than one person touches this. See `docs/phases/phase-4-scale-ha-billing.md` for proper SSO.

## Run locally

```
export MGMT_API_KEY=dev-key
go run .
curl -H "X-Mgmt-Api-Key: dev-key" -X POST localhost:8090/api/tenants \
  -d '{"slug":"acme","display_name":"Acme Corp","plan":"trial"}'
```

## Next real step

Wire `internal/k8s.CreateTenantNamespace` + `InstallOrUpgradeRelease` into `createTenant`, using `charts/nexwall-controller` as the chart to install. That single change moves this from "reserves an allocation" to "actually provisions a customer."
