#!/usr/bin/env bash
# k0s-teardown.sh — Stop and reset the local k0s cluster.
# Usage: sudo ./hack/k0s-teardown.sh
set -euo pipefail

echo "==> Stopping k0s..."
k0s stop || true

echo "==> Resetting k0s..."
k0s reset --yes || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
rm -f "${SCRIPT_DIR}/../kubeconfig"

echo "==> Cluster torn down."
