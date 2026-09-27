# ADR 0005: Public entry point — VPS + WireGuard + nginx, not a public IP on the cluster nodes

**Status**: Accepted

## Context

The k3s cluster (master + workers) lives entirely on a private network with no public IP of its own. A pre-existing VPS (public IP, already running WireGuard for other internal services — an email server and a dev server, each as a WireGuard peer) is the only publicly reachable box in the infrastructure.

## Decision

- The cluster master joins the VPS's existing WireGuard hub as a new peer, `10.200.1.10/24` (VPS is `10.200.1.1`), added without disturbing the existing peers.
- nginx on the VPS (already serving other `*.nexwall.com.br` subdomains via certbot) gets one more site: `*.painel.nexwall.com.br` reverse-proxied to `http://10.200.1.10:80` (the master's Traefik ingress), passing the `Host` header through untouched so Traefik's own host-based routing (ADR 0004) still does the actual per-tenant dispatch.
- No public IP or NodePort is exposed directly on any cluster node for HTTP(S) traffic. The VPN port scheme from ADR 0004 (`20000+tenant_id` UDP) is a separate concern — OpenVPN traffic is NOT proxied through this nginx/WireGuard path in the current design; it still needs its own reachability plan (open item, see Consequences).

## Consequences

- One extra hop (VPS → WireGuard → Traefik) versus a cluster node with a public IP, but it means the cluster's private network topology never has to change if the VPS or its public IP changes.
- TLS termination point is now an open decision between two reasonable places: certbot on the VPS's nginx (consistent with how `license`/`updates`/`lists.nexwall.com.br` already work on this same box) versus cert-manager inside the cluster. Leaning toward the VPS/certbot option for consistency, not yet final.
- **Open item, not yet solved**: the per-tenant OpenVPN NodePort scheme (ADR 0004) assumed a public IP directly reachable on a cluster node. With the cluster fully private behind this VPS, each tenant's VPN port needs its own DNAT rule on the VPS's WireGuard interface (the existing `wg0.conf` already has a `PostUp`/`PostDown` DNAT pattern for the email/dev-server peers — the same pattern extends here, one rule pair per tenant, forwarding VPS-public-UDP-port → `10.200.1.10:tenant-nodeport`). Not yet scripted; must be done before any real customer's firewall unit tries to register.
- DNS: `*.painel.nexwall.com.br` must point at the VPS's public IP, not any cluster node's IP. No wildcard DNS record exists yet at the time of writing (domain is on Locaweb) — blocking item for actually reaching the cluster from the internet.
