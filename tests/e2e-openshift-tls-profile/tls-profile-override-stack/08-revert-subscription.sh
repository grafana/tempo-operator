#!/bin/bash
# Revert the TLS_PROFILE and FEATURE_GATES overrides from the operator Subscription.
# Removing them from subscription config.env lets the CSV defaults take effect again.
set -euo pipefail

# Get the Tempo Operator namespace
TEMPO_OPERATOR_NAMESPACE=$(oc get pods -A \
    -l control-plane=controller-manager \
    -l app.kubernetes.io/name=tempo-operator \
    -o jsonpath='{.items[0].metadata.namespace}')

# Get the Tempo Operator subscription
TEMPO_OPERATOR_SUB=$(oc get subscription -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.items[0].metadata.name}')

echo "Reverting TLS_PROFILE and FEATURE_GATES overrides from subscription $TEMPO_OPERATOR_SUB"

# Get current env vars and remove TLS_PROFILE and FEATURE_GATES overrides
CURRENT_ENV=$(oc get subscription "$TEMPO_OPERATOR_SUB" -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.spec.config.env}' 2>/dev/null || echo "[]")

NEW_ENV=$(echo "$CURRENT_ENV" | python3 -c "
import json, sys
try:
    env = json.load(sys.stdin)
    if not isinstance(env, list):
        env = []
except:
    env = []
# Remove TLS_PROFILE and FEATURE_GATES (let CSV defaults take effect)
env = [e for e in env if e.get('name') not in ('TLS_PROFILE', 'FEATURE_GATES')]
print(json.dumps(env))
")

oc patch subscription "$TEMPO_OPERATOR_SUB" -n "$TEMPO_OPERATOR_NAMESPACE" \
    --type='merge' -p "{\"spec\": {\"config\": {\"env\": $NEW_ENV}}}"

# Wait for OLM to propagate and rollout
echo "Waiting for OLM to propagate the revert..."
for i in $(seq 1 30); do
  CURRENT_TLS=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TLS_PROFILE")].value}' 2>/dev/null || true)
  if [ -z "$CURRENT_TLS" ]; then
    echo "OLM removed TLS_PROFILE from deployment (attempt $i)"
    break
  fi
  echo "  Waiting for OLM propagation (attempt $i/30)..."
  sleep 10
done
oc rollout status deployment/tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" --timeout=300s

echo "PASS: TLS_PROFILE and FEATURE_GATES reverted from operator subscription"
