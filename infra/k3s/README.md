# infra/k3s

Bootstrap scripts for the 2-VM starting cluster (ADR 0001). Expansion is `install-worker.sh` on any new VM pointed at the same master — no change to these scripts needed.

## Order of operations

1. `./install-master.sh` on VM1. Note the `K3S_URL`/`K3S_TOKEN` it prints.
2. `K3S_URL=... K3S_TOKEN=... ./install-worker.sh` on VM2.
3. Create the DNS-01 `ClusterIssuer` for cert-manager by hand (provider-specific — not scripted, see the comment at the end of `install-master.sh`).
4. Request the wildcard cert (`*.painel.nexwall.com.br`) as a `Certificate` resource with `secretName: wildcard-painel-tls` — this is the secret every tenant's `Ingress` (see `charts/nexwall-controller/templates/ingress.yaml`) expects to already exist in its namespace. Until multi-namespace cert sharing is set up (e.g. via `cert-manager`'s `ClusterIssuer` + a Helm hook copying the secret per-namespace, or [reflector](https://github.com/emberstack/kubernetes-reflector)), each new tenant namespace needs that secret copied into it manually or by the Management Plane at provisioning time (Phase 2 work item).
5. Create the Management Plane's own ServiceAccount + ClusterRole (scoped to `namespaces`, `deployments`, `statefulsets`, `services`, `secrets`, `persistentvolumeclaims`, `ingresses` — nothing else) before Phase 2 wires up `internal/k8s`.

## Node roles

Per ADR 0001: no taint on the master while there are only 2 nodes. Once a 3rd node joins:

```
kubectl taint nodes <master-node-name> node-role.kubernetes.io/master=:NoSchedule
```

## Not yet scripted (deliberately out of scope for Phase 0/1)

- HA control-plane (3 masters) — Phase 4.
- Terraform/IaC for VM provisioning itself — VMs are created by hand for now; the 2 starting VMs already exist outside this repo's scope.

## Real deployment notes (from the actual master + slave1 setup)

- **Node IP / internal network**: cluster nodes talk to each other over a dedicated private interface (`ens34`, `10.10.1.0/24`), not the VM's public-facing interface. Every node's `/etc/rancher/k3s/config.yaml` sets `node-ip` (its `10.10.1.x` address), `node-external-ip` (its `192.168.0.x` address, unused by the cluster itself today but kept for visibility), and `flannel-iface: ens34`. Workers join via `server: https://10.10.1.<master>:6443`, not the public/external address.
- **Taint does not survive a k3s restart**: `node-taint` in `config.yaml` is a first-registration-only setting. After *any* restart of the master's `k3s` service (including a VM reboot), check and reapply:
  ```
  kubectl describe node <master> | grep Taints
  kubectl taint nodes <master> node-role.kubernetes.io/control-plane=:NoSchedule --overwrite
  ```
  This does not evict already-running pods that lack a toleration (they keep running); it only blocks *new* scheduling. So a missed re-taint after a reboot is a silent risk (a tenant could get scheduled onto the master), not an immediate outage — but it should be checked every time.
- **Public entry point**: see `docs/adr/0005-public-entry-point-wireguard-nginx.md` — the cluster has no public IP of its own. A separate VPS reaches the master over WireGuard (`10.200.1.0/24`) and nginx there reverse-proxies to the master's Traefik.
