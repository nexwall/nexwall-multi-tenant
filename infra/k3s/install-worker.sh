#!/bin/bash
# Run on VM2 (worker), or any additional worker VM per ADR 0001's expansion
# path. Needs K3S_URL and K3S_TOKEN printed by install-master.sh.
#
# Usage: K3S_URL=https://<master-ip>:6443 K3S_TOKEN=<token> ./install-worker.sh
set -euo pipefail

: "${K3S_URL:?Set K3S_URL to https://<master-ip>:6443}"
: "${K3S_TOKEN:?Set K3S_TOKEN from install-master.sh's output}"

echo "== Installing k3s agent, joining ${K3S_URL} =="
curl -sfL https://get.k3s.io | K3S_URL="${K3S_URL}" K3S_TOKEN="${K3S_TOKEN}" sh -

echo "== Done. Verify from the master with: kubectl get nodes =="
