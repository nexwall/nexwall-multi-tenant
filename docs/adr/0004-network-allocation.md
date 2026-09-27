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
