#!/bin/bash
set -euo pipefail
oc patch apiserver cluster --type json \
  -p '[{"op":"remove","path":"/spec/tlsSecurityProfile"}]'
echo "APIServer TLS profile reverted to default (nil)"

# Wait for the operator to restart and its webhook to become available.
# The APIServer revert triggers SecurityProfileWatcher to restart the operator,
# and chainsaw cleanup will fail if the webhook is unavailable when deleting the TempoStack.
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
