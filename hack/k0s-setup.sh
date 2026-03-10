#!/usr/bin/env bash
# k0s-setup.sh — Initialize a single-node k0s cluster with kine (SQLite) for AOT local development.
# Usage: sudo ./hack/k0s-setup.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="${SCRIPT_DIR}/k0s-config.yaml"
KUBECONFIG_OUT="${SCRIPT_DIR}/../kubeconfig"
REAL_USER="${SUDO_USER:-$(whoami)}"

echo "==> Checking k0s..."
if ! command -v k0s &>/dev/null; then
  echo "ERROR: k0s not found. Install it first: https://docs.k0sproject.io/stable/install/"
  exit 1
fi

echo "==> k0s version: $(k0s version)"

# Stop existing cluster if running
if k0s status &>/dev/null; then
  echo "==> Stopping existing k0s cluster..."
  k0s stop || true
  k0s reset || true
fi

echo "==> Installing k0s controller with worker (single-node)..."
k0s install controller --single --config "${CONFIG}"

echo "==> Starting k0s..."
k0s start

echo "==> Waiting for k0s to be ready..."
for i in $(seq 1 90); do
  if k0s kubectl get nodes 2>/dev/null | grep -q "Ready"; then
    break
  fi
  echo "    waiting... (${i}/90)"
  sleep 2
done

echo "==> Generating kubeconfig..."
k0s kubeconfig admin > "${KUBECONFIG_OUT}"
chown "${REAL_USER}:${REAL_USER}" "${KUBECONFIG_OUT}"
chmod 600 "${KUBECONFIG_OUT}"

echo "==> Cluster ready!"
echo "    export KUBECONFIG=${KUBECONFIG_OUT}"
k0s kubectl get nodes
