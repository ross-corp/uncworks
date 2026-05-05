#!/usr/bin/env bash
# hack/uncworks-status.sh — Emit a compact UNCWORKS system snapshot.
# Used by the Claude Code UserPromptSubmit hook so Claude always has current context.
# Output is prepended to every user message automatically — keep it short.
set -euo pipefail

NS=uncworks

echo "=== UNCWORKS STATUS ==="

# Cluster health
POD_OUTPUT=$(kubectl get pods -n "$NS" --no-headers 2>/dev/null || true)
if [ -z "$POD_OUTPUT" ]; then
  CLUSTER="not_installed"
else
  CLUSTER=$(echo "$POD_OUTPUT" | \
    awk '!/Completed|Succeeded|Running/{if($0!="")bad++} END{print (bad>0?"degraded":"running")}')
fi
echo "Cluster: $CLUSTER"

# Core deployments (ready/desired)
kubectl get deployments -n "$NS" \
  -l 'app.kubernetes.io/name=aot' \
  --no-headers 2>/dev/null | \
  awk '{printf "  %-30s %s/%s\n", $1, $2, $3}' || true

# LiteLLM
LITELLM=$(kubectl get deployment litellm -n "$NS" --no-headers 2>/dev/null | awk '{print $2"/"$3}')
echo "LiteLLM: $LITELLM"

# Recent runs (last 5)
echo "Runs (recent):"
kubectl get agentruns -n "$NS" \
  --sort-by=.metadata.creationTimestamp \
  --no-headers 2>/dev/null | tail -5 | \
  awk '{printf "  %-20s %-12s %s\n", $1, $3, $4}' || true

# Stale/competing workers in other namespaces
EXTRA_WORKERS=$(kubectl get pods -A --no-headers 2>/dev/null | \
  grep 'worker' | grep -v "^$NS " | grep Running | wc -l | tr -d ' ' || echo 0)
if [ "$EXTRA_WORKERS" -gt 0 ]; then
  echo "WARNING: $EXTRA_WORKERS worker pod(s) running outside $NS namespace"
  kubectl get pods -A --no-headers 2>/dev/null | grep 'worker' | grep -v "^$NS "
fi

# Temporal: any stuck workflows
STUCK=$(temporal workflow list --namespace default --address localhost:7233 \
  --query 'ExecutionStatus="Running"' 2>/dev/null | grep -c 'AgentRunWorkflow' || true)
echo "Temporal running workflows: ${STUCK:-0}"

# Log tail (last 5 lines, errors only)
LOGFILE="$HOME/Library/Logs/UNCWORKS/uncworks.log"
if [ -f "$LOGFILE" ]; then
  ERRORS=$(grep -i 'error\|panic\|fatal' "$LOGFILE" | tail -3 || true)
  if [ -n "$ERRORS" ]; then
    echo "Recent log errors:"
    echo "$ERRORS" | sed 's/^/  /'
  fi
fi

echo "======================"
