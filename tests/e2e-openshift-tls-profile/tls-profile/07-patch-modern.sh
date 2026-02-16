#!/bin/bash
set -euo pipefail

# Patch APIServer to Modern profile
oc patch apiserver cluster --type merge \
  -p '{"spec":{"tlsSecurityProfile":{"type":"Modern","modern":{}}}}'

# Poll ConfigMap for Modern settings (tls_min_version: VersionTLS13)
echo "Waiting for ConfigMap to update with Modern profile..."
for i in $(seq 1 60); do
  CONFIG=$(kubectl get configmap tempo-simplest -n $NAMESPACE -o jsonpath='{.data.tempo\.yaml}' 2>/dev/null || echo "")
  if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS13'; then
    echo "ConfigMap updated with Modern profile"
    break
  fi
  if [ $i -eq 60 ]; then
    echo "FAIL: ConfigMap not updated after 5 minutes"
    exit 1
  fi
  sleep 5
done

# Wait for operator restart (SecurityProfileWatcher triggers graceful restart on TLS profile change).
# Do this early because the operator must restart before it reconciles operands with new TLS settings.
echo "Waiting for operator restart..."
kubectl rollout status deployment/tempo-operator-controller -n openshift-tempo-operator --timeout=5m || true
echo "Operator rollout check done"

# First pass: try to wait for component rollouts, but tolerate timeouts.
# MCO node reconciliation may drain nodes and evict pods concurrently, causing rollout timeouts.
echo "First rollout pass (tolerating timeouts due to concurrent MCO node reconciliation)..."
for deploy in tempo-simplest-gateway tempo-simplest-distributor tempo-simplest-querier tempo-simplest-query-frontend tempo-simplest-compactor; do
  echo "Waiting for $deploy rollout..."
  kubectl rollout status deployment/$deploy -n $NAMESPACE --timeout=3m || echo "  $deploy rollout not yet complete (will retry after node reconciliation)"
done
echo "Waiting for ingester rollout..."
kubectl rollout status statefulset/tempo-simplest-ingester -n $NAMESPACE --timeout=3m || echo "  ingester rollout not yet complete (will retry after node reconciliation)"

# Wait for all nodes to finish MCO rollout (APIServer patch triggers node-by-node reconciliation).
# Nodes go SchedulingDisabled during drain, which evicts pods and causes network disruptions.
echo "Waiting for all nodes to be Ready and schedulable..."
for i in $(seq 1 90); do
  NOT_READY=$(kubectl get nodes --no-headers | grep -c "SchedulingDisabled" || true)
  if [ "$NOT_READY" -eq 0 ]; then
    echo "All nodes Ready and schedulable"
    break
  fi
  if [ $i -eq 90 ]; then
    echo "WARNING: Some nodes still SchedulingDisabled after 15 minutes, proceeding anyway"
    kubectl get nodes
    break
  fi
  echo "  Nodes with SchedulingDisabled: $NOT_READY (attempt $i/90)"
  sleep 10
done

# Second pass: wait for all Tempo pods to be Running and Ready after node reconciliation.
# This pass uses strict error checking - failures here are real failures.
echo "Second rollout pass (post node reconciliation, strict)..."
for deploy in tempo-simplest-gateway tempo-simplest-distributor tempo-simplest-querier tempo-simplest-query-frontend tempo-simplest-compactor; do
  echo "Waiting for $deploy rollout..."
  kubectl rollout status deployment/$deploy -n $NAMESPACE --timeout=5m
done
echo "Waiting for ingester rollout..."
kubectl rollout status statefulset/tempo-simplest-ingester -n $NAMESPACE --timeout=5m

# Ensure no terminating pods remain (old pods from pre-rollout)
echo "Waiting for terminating pods to clear..."
for i in $(seq 1 30); do
  TERMINATING=$(kubectl get pods -n $NAMESPACE --field-selector=status.phase!=Running,status.phase!=Succeeded,status.phase!=Failed --no-headers 2>/dev/null | grep -c "Terminating" || true)
  if [ "$TERMINATING" -eq 0 ]; then
    echo "No terminating pods"
    break
  fi
  echo "  Terminating pods: $TERMINATING (attempt $i/30)"
  sleep 5
done

# Brief stabilization period
sleep 10
echo "All components rolled out with Modern profile"
