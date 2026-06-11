#!/bin/bash
set -euo pipefail
oc patch apiserver cluster --type json \
  -p '[{"op":"remove","path":"/spec/tlsSecurityProfile"}]'
echo "APIServer TLS profile reverted to default (nil)"

# Wait for the operator to restart (SecurityProfileWatcher triggers graceful restart on TLS profile change).
echo "Waiting for operator restart after revert..."
kubectl rollout status deployment/tempo-operator-controller -n openshift-tempo-operator --timeout=5m || true
echo "Operator rollout check done"

# Wait for all nodes to finish MCP reconciliation triggered by the APIServer revert.
# Same pattern as 07-patch-modern.sh: the APIServer revert causes another full node-by-node
# drain cycle. If MCP is still draining nodes during chainsaw cleanup, the operator pod can be
# evicted making the admission webhook unavailable when chainsaw deletes TempoStack/TempoMonolithic CRs.
echo "Waiting for all nodes to be Ready and schedulable after APIServer revert..."
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

# Wait for MachineConfigPool to report fully updated for both pools.
echo "Waiting for MachineConfigPools to finish updating..."
for pool in master worker; do
  for i in $(seq 1 60); do
    UPDATED=$(kubectl get mcp "$pool" -o jsonpath='{.status.conditions[?(@.type=="Updated")].status}' 2>/dev/null || echo "")
    UPDATING=$(kubectl get mcp "$pool" -o jsonpath='{.status.conditions[?(@.type=="Updating")].status}' 2>/dev/null || echo "")
    if [ "$UPDATED" = "True" ] && [ "$UPDATING" = "False" ]; then
      echo "MCP $pool: UPDATED=True UPDATING=False"
      break
    fi
    if [ $i -eq 60 ]; then
      echo "WARNING: MCP $pool not fully updated after 10 minutes (UPDATED=$UPDATED UPDATING=$UPDATING)"
      break
    fi
    echo "  MCP $pool: UPDATED=$UPDATED UPDATING=$UPDATING (attempt $i/60)"
    sleep 10
  done
done

# After MCP stabilizes the operator pod may have been re-evicted; wait for its rollout again.
echo "Waiting for operator deployment to be stable after MCP stabilization..."
kubectl rollout status deployment/tempo-operator-controller -n openshift-tempo-operator --timeout=5m

# Wait for the operator webhook endpoint to be available and remain stable.
# The operator must be Running and its webhook ready before chainsaw cleanup deletes TempoStack/TempoMonolithic CRs.
echo "Waiting for operator webhook to become available after revert..."
for i in $(seq 1 60); do
  ENDPOINTS=$(kubectl get endpoints tempo-operator-controller-service -n openshift-tempo-operator -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || echo "")
  if [ -n "$ENDPOINTS" ]; then
    echo "Operator webhook endpoint available at $ENDPOINTS"
    break
  fi
  if [ $i -eq 60 ]; then
    echo "WARNING: Operator webhook not available after 5 minutes, cleanup may fail"
  fi
  sleep 5
done

# Brief stabilization before chainsaw cleanup begins.
sleep 10
echo "APIServer revert complete and operator stable"
