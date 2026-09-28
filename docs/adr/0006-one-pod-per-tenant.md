# ADR 0006: One multi-container Pod per tenant (not one Deployment per component)

**Status**: Accepted — validated on the real cluster (tenant `demo`, revision 2, 8/8 containers, 0 restarts)

## Context

The first version of the Helm chart translated each `podman run` from `nexwall-controller`'s `dev.sh` into its own k8s Deployment. Reading the controller's own scripts showed that cannot work:

- `proxy/entrypoint.sh` hardcodes its upstreams as `http://127.0.0.1:${api_port}/` and `http://127.0.0.1:${ui_port}/`.
- `vpn/handle-connection` writes Traefik routes to `https://<unit-vpn-ip>:9090/`. Unit VPN addresses are only routable from the network namespace that owns the tunnel interface, so Traefik must live there.
- Units ship syslog to the VPN server address; promtail must listen on the tunnel interface.
- The vpn image's `ip` is a no-op shim: OpenVPN never configures its own interface, something outside must create and address `tunsec` (dev.sh did it on the host).

## Decision

One Deployment per tenant whose Pod runs `vpn`, `api`, `ui`, `proxy`, `loki`, `promtail`, `prometheus`, `grafana` as containers of the same Pod (shared netns), mirroring the podman pod. Init containers: `tun-setup` (creates `tunsec`, assigns `<vpn-cidr>.0.1/16`), `wait-db`, `route-files` (drops the grafana/prometheus Traefik routes into the proxy's watched dir on the shared volume). TimescaleDB stays a separate StatefulSet (real state, own lifecycle), reached by Service DNS.

## Consequences

- Strategy is `Recreate`: RWO volumes and a single tunnel owner mean two replicas can never coexist. A tenant has brief downtime on upgrade — acceptable, and it is per-tenant.
- Components cannot scale independently; the tenant Pod's requests are the sum of all containers (~700Mi). That is the real unit of capacity planning.
- Ports inside the shared netns must not collide: ui=3000, so grafana is moved to 3009.
- Bugs found only because we ran it for real (all fixed in the chart): api needs `OVPN_NETMASK` (its entrypoint writes it into `conf.env` for the vpn hooks; dev harness never set it); api state (`tokens`, `credentials`, `data`, `secrets`) needs a PVC; Prometheus with `--web.route-prefix` needs the prefix in the Grafana datasource URL and in its own self-scrape `metrics_path`; Grafana dashboards reference datasources by fixed `uid`, which provisioning must set.
