# ADR 0004: Network allocation — VPN CIDR/port, HTTP subdomain routing

**Status**: Accepted

## Context

Official NethSecurity docs warn that each controller's OpenVPN network must not overlap the internal networks of the units connecting to it. With multiple controller instances on shared infrastructure, they also must not overlap **each other** or collide on the public IP.

## Decision

| Resource | Scheme | Example (tenant_id=7) |
|---|---|---|
| VPN CIDR | `10.<100 + tenant_id>.0.0/16` | `10.107.0.0/16` |
| VPN port (NodePort, UDP) | `20000 + tenant_id` | `20007/udp` |
| HTTP(S) | One shared Ingress, `Host`-based routing | `acme.painel.nexwall.com.br` |
| TLS | One wildcard cert, cert-manager + Let's Encrypt DNS-01 | `*.painel.nexwall.com.br` |

`tenant_id` is a small integer assigned once at tenant creation by the Management Plane (auto-increment in its own DB) and is never reused after a tenant is deleted, to avoid CIDR/port reuse racing a still-connecting old unit.

## Consequences

- 1000 tenants fit in ports `20000`–`20999` on a single public IP before a second IP is needed — comfortable headroom for the 2-VM starting point and well beyond.
- HTTP scales independently of the VPN scheme — the Ingress handles unlimited tenants on one IP via SNI/Host routing, no per-tenant port needed.
- `tenant_id` must be threaded through the Helm chart's `values.yaml` (`vpnCidr`, `vpnPort`, `subdomain`) — see `docs/contracts/` for the schema that enforces this at install time.

## Addendum (found while deploying the first real tenant)

1. **NodePort range**: the scheme above puts VPN NodePorts at `20000+tenant_id`, but k3s only accepts 30000–32767 by default and the Service is rejected. The master must run with `kube-apiserver-arg: service-node-port-range=20000-32767` (in `/etc/rancher/k3s/config.yaml`; see `infra/k3s/README.md`). Verified: `demo` (tenant 1) got `20001/UDP`.
2. **CIDR scheme caps at 99 tenants, not 1000**: `10.<100+tenant_id>.0.0/16` reaches `10.200.0.0/16` at tenant 100, colliding with the WireGuard network (`10.200.1.0/24`) used to reach the cluster (ADR 0005). Valid range today is `tenant_id` 1–99. Must be redesigned (e.g. smaller per-tenant blocks from a range that avoids 10.10/10.42/10.43/10.200) before the 100th tenant; the Management Plane must refuse to allocate beyond 99 until then.
3. Verified on the real cluster: OpenVPN bound `:20001/UDP` with pool `10.101.0.2+`, promtail bound `10.101.0.1:1514`.
