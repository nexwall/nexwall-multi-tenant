# nexwall-controller Helm chart

One `helm install` = one fully isolated tenant. See `docs/adr/0007-one-pod-per-tenant.md` for why the stack is a single multi-container Pod (plus a separate TimescaleDB StatefulSet) and not one Deployment per component.

## Layout

| Template | What |
|---|---|
| `controller.yaml` | PVCs, the tenant Pod (init: `tun-setup`, `wait-db`, `route-files`; containers: vpn, api, ui, proxy, loki, promtail, prometheus, grafana), `-web` Service (8080), `-vpn` NodePort Service (UDP `vpnPort`) |
| `db-statefulset.yaml` | TimescaleDB + headless Service |
| `configmaps.yaml` | loki / promtail / prometheus / grafana provisioning, Traefik route files |
| `secrets.yaml` | `api.env` equivalent, db and grafana credentials |
| `ingress.yaml` | Host-based Ingress to the tenant's Traefik (TLS optional) |
| `dashboards/*.json` | Grafana dashboards, UIDs match the links hardcoded in nexwall-ui |

## Prerequisites on the cluster

- k3s `service-node-port-range` must include the tenant's `vpnPort` (ADR 0004 addendum).
- The four `ghcr.io/nexwall/nethsecurity-*:<tag>` images must exist on the worker's containerd (`podman save -m` + `k3s ctr images import`) until a registry is in place.
- Worker nodes need the `tun` kernel module (`/dev/net/tun`).

## Known gaps

1. Secrets default to `CHANGE_ME` — always override (never commit real values).
2. GeoIP: the api logs `error downloading geoip db file ... 401` (needs a MaxMind license). Non-fatal, not yet investigated.
3. Dashboards are placeholders wired to the right UIDs, not the full upstream dashboards.
4. No NetworkPolicy yet (tenant namespaces are not network-isolated from each other).
5. No probes/PodDisruptionBudget.

## Install

```
helm install tenant-<id>-<slug> charts/nexwall-controller -n tenant-<id>-<slug> --create-namespace -f values-<slug>.yaml
```
`values-<slug>.yaml` must satisfy `values.schema.json`.
