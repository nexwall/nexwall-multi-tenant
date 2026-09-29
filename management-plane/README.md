# Management Plane

Two separate things live in this one Go binary, on purpose (see `docs/adr/0008-single-domain-identity-routing.md`):

1. **`/_mgmt/*`** — provisioning/admin API for MSP staff. Implements `docs/contracts/management-plane-openapi.yaml`. Auth: a single shared `MGMT_API_KEY` header.
2. **`/login`, `/logout`, everything else (`/`, `/api/*`, ...)** — the customer-facing side: a login broker + session-keyed reverse proxy into whichever tenant a browser is logged into. This is what makes `painel.nexwall.com.br` work as one hostname for every customer (`internal/auth/`).

These two are intentionally on disjoint path namespaces — `/_mgmt` vs. everything else — so a tenant's own proxied traffic (which uses paths like `/api/login`) can never collide with the provisioning API.

## Current state

**Provisioning (`/_mgmt`)**: `POST /_mgmt/tenants` allocates a `tenant_id`, VPN CIDR/port (`docs/adr/0004`), and a subdomain — but does **not** yet call k8s. `internal/k8s/client.go` is a stub (`panic("not implemented")`) intentionally, so wiring it up is a clear, isolated next step. Storage is in-memory (`internal/tenant/memstore.go`) — replace with Postgres before anything but local dev.

**Login/proxy (`internal/auth/`)**: real, and validated (`go build`/`go vet` pass against the actual `gin` + `client-go` dependencies). `POST /login` looks up the email in the same in-memory store's new `emailIndex` (populated from `admin_email` at tenant creation — see the "open item" in ADR 0008 about users added later inside a tenant), replays the password against that tenant's own `/api/login`, and on success sets an `nx_session` cookie. Every other request goes through `RequireSession` + `ProxyToTenant`, which forwards to `http://<release>-web.<namespace>.svc.cluster.local:8080` with the tenant's JWT injected. **Not yet tested against a real logged-in browser or a real tenant Service** — only compiled and vetted so far.

## Run locally

```
export MGMT_API_KEY=dev-key
go run .

# provisioning
curl -H "X-Mgmt-Api-Key: dev-key" -X POST localhost:8090/_mgmt/tenants \
  -d '{"slug":"acme","display_name":"Acme Corp","plan":"trial","admin_email":"admin@acme.com"}'

# login (will fail until a real tenant Service exists at InClusterWebAddr —
# this only works run inside the cluster, or against a port-forwarded tenant)
curl -c cookies.txt -X POST localhost:8090/login \
  -d '{"email":"admin@acme.com","password":"..."}'
```

## Next real steps

1. Wire `internal/k8s.CreateTenantNamespace` + `InstallOrUpgradeRelease` into `createTenant`, using `charts/nexwall-controller` — moves `/_mgmt/tenants` from "reserves an allocation" to "actually provisions a customer."
2. Deploy the Management Plane itself into the cluster and repoint the VPS nginx at it (today nginx forwards straight to a tenant's Traefik by Host header — needs to forward everything to the Management Plane instead, per ADR 0008).
3. Test the login/proxy path against the real `demo` tenant already running on the cluster.
