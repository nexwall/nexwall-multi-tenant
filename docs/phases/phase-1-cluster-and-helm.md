# Phase 1 — Cluster bootstrap + one tenant via Helm

**Goal**: k3s running on both VMs, one real customer stack deployed and reachable through it — no Management Plane yet, tenant created by hand.

## Work items
1. `infra/k3s/install-master.sh` on VM1, `infra/k3s/install-worker.sh` on VM2 (join token from VM1).
2. Install cert-manager + Traefik Ingress (Traefik ships with k3s by default — confirm version, upgrade if needed) + wildcard cert via DNS-01.
3. Finish `charts/nexwall-controller/templates/` — every resource currently produced by `dev-nexwall.sh`'s `podman run` calls, translated 1:1 to k8s (see chart README for the mapping table already done in this scaffold).
4. `helm install tenant-pilot charts/nexwall-controller -f values-pilot.yaml` — first real deploy.
5. Manually verify: unit registration flow (the same `ns-plug` → `/units/register` flow already validated in the single-VM dev setup) works through the Ingress + tenant-scoped VPN port.
6. Move the api/ui/proxy/vpn image build (already in `nexwall-controller`'s GitHub Actions) — no change needed here, this repo only consumes those tags.

## Exit criteria
One paying-customer-grade stack reachable at `pilot.painel.<domain>`, a real firewall registered against it end-to-end (UI login, unit online, metrics/logs flowing — same checks already validated in the single-VM dev environment, now through k3s).
