# ADR 0001: Orchestrator — k3s

**Status**: Accepted
**Context**: We need to run N isolated customer stacks (currently 10 containers each) across a small, growing number of VMs, starting at 2 (1 master + 1 worker).

## Options considered

| Option | Multi-VM clustering | Tenant isolation primitives | Expansion |
|---|---|---|---|
| Podman pods (current dev setup) | None — pods are per-host | Manual (bind mounts, ports by hand) | Reinstall/reconfigure per VM |
| Docker Swarm | Yes, `docker swarm join` | Weak — overlay networks, no real NetworkPolicy | Simple but low ongoing investment from Docker Inc. |
| k3s (Kubernetes) | Yes, `k3s agent` join | `Namespace`, `NetworkPolicy`, `ResourceQuota`, `LimitRange` — all native | `k3s agent` on a new node, one command |

## Decision

k3s. Multi-tenancy needs real isolation primitives (Namespace + NetworkPolicy + ResourceQuota), which only Kubernetes provides natively among the options above. k3s specifically because it ships as a single ~70MB binary, needs no external etcd for small clusters (uses embedded SQLite/dqlite), and is the de facto standard for small/edge k8s clusters — appropriate for a 2-VM starting point that must scale to N workers without a re-platform.

## Consequences

- Every component from here on (Helm chart, Ingress, Secrets) is k8s-native, not Podman-native. `dev-nexwall.sh` (the Podman dev harness in `nexwall-controller`) stays as the local single-tenant dev tool; it is not the deployment mechanism for this repo.
- We take on k8s operational complexity (RBAC, etcd/dqlite backups, node upgrades) earlier than strictly necessary for 2 customers — accepted as the cost of not re-platforming at 10 customers.

## Addendum (from real deployment, master VM)

Revised mid-flight against actual hardware: the master VM started at 2GB RAM, where the control-plane alone (coredns, traefik, local-path-provisioner, metrics-server) already consumed 61% of allocatable memory. Even after upgrading to 4GB, we chose to **taint the master `NoSchedule` immediately**, not wait for a 3rd node as originally planned above — with per-tenant stacks requesting ~700Mi and bursting past 2.7Gi at limits, there was never real headroom to share the master with tenant workload.

**Operational gotcha found**: `node-taint` in `/etc/rancher/k3s/config.yaml` only applies the first time a node registers — it is silently dropped on subsequent `systemctl restart k3s` / reboots. The taint must be re-applied imperatively (`kubectl taint nodes <master> node-role.kubernetes.io/control-plane=:NoSchedule --overwrite`) after any master restart until this is scripted into a startup check. See `infra/k3s/README.md`.
