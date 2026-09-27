#!/bin/bash
# Run on VM1 (master). Installs k3s server, cert-manager, and prints the
# join token VM2 needs for install-worker.sh.
#
# Usage: ./install-master.sh
set -euo pipefail

echo "== Installing k3s server =="
curl -sfL https://get.k3s.io | sh -s - server \
    --write-kubeconfig-mode 644 \
    --disable traefik=false

echo "== Waiting for node Ready =="
until kubectl get nodes 2>/dev/null | grep -q " Ready"; do sleep 2; done
kubectl get nodes

echo "== Installing cert-manager =="
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
kubectl -n cert-manager wait --for=condition=Available --timeout=120s deployment --all

cat <<'EOF'

== Next steps ==
1. Create a ClusterIssuer for the wildcard cert (DNS-01 challenge — provider
   depends on where painel.nexwall.com.br's DNS is hosted; not scripted here
   since it needs provider credentials as a Secret first).
2. Copy the join token below to VM2 and run install-worker.sh there:

EOF
echo "K3S_URL=https://$(hostname -I | awk '{print $1}'):6443"
echo "K3S_TOKEN=$(cat /var/lib/rancher/k3s/server/node-token)"
