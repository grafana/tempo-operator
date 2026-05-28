#!/bin/bash
# Revert the TLS_PROFILE and FEATURE_GATES overrides from the operator.
# Supports operators installed via:
#   1. OLM subscription - removes env from subscription config so CSV defaults take effect
#   2. operator-sdk run bundle - reverts CSV deployment spec env vars
set -euo pipefail

# Get the Tempo Operator namespace
TEMPO_OPERATOR_NAMESPACE=$(oc get pods -A \
    -l control-plane=controller-manager \
    -l app.kubernetes.io/name=tempo-operator \
    -o jsonpath='{.items[0].metadata.namespace}')

# Detect install method: subscription > CSV > plain deployment
TEMPO_OPERATOR_SUB=$(oc get subscription -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

TEMPO_OPERATOR_CSV=""
if [ -z "$TEMPO_OPERATOR_SUB" ]; then
  TEMPO_OPERATOR_CSV=$(oc get csv -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep tempo || echo "")
fi

if [ -n "$TEMPO_OPERATOR_SUB" ]; then
  # OLM install with subscription: revert subscription
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

elif [ -n "$TEMPO_OPERATOR_CSV" ]; then
  # operator-sdk bundle install: revert CSV deployment spec env vars
  echo "No subscription found, reverting CSV $TEMPO_OPERATOR_CSV env vars in $TEMPO_OPERATOR_NAMESPACE"

  # Get current FEATURE_GATES and re-add openshift.clusterTLSPolicy
  CURRENT_FEATURE_GATES=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="FEATURE_GATES")].value}')

  ORIGINAL_FEATURE_GATES=$(echo "$CURRENT_FEATURE_GATES" | python3 -c "
import sys
gates = sys.stdin.read().strip()
parts = [g.strip() for g in gates.split(',') if g.strip()]
if 'openshift.clusterTLSPolicy' not in parts:
    parts.append('openshift.clusterTLSPolicy')
print(','.join(parts))
")

  echo "Restoring FEATURE_GATES to: $ORIGINAL_FEATURE_GATES"

  # Build the reverted env array using JSON patch type to avoid annotation size limits.
  NEW_ENV=$(oc get csv "$TEMPO_OPERATOR_CSV" -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.install.spec.deployments[0].spec.template.spec.containers[0].env}' | python3 -c "
import json, sys
fg = '$ORIGINAL_FEATURE_GATES'
env = json.load(sys.stdin)
env = [e for e in env if e.get('name') not in ('TLS_PROFILE', 'FEATURE_GATES')]
env.append({'name': 'FEATURE_GATES', 'value': fg})
print(json.dumps(env))
")

  oc patch csv "$TEMPO_OPERATOR_CSV" -n "$TEMPO_OPERATOR_NAMESPACE" --type json \
      -p "[{\"op\": \"replace\", \"path\": \"/spec/install/spec/deployments/0/spec/template/spec/containers/0/env\", \"value\": $NEW_ENV}]"

  # Wait for OLM to reconcile the CSV change to the deployment
  echo "Waiting for OLM to reconcile CSV revert to deployment..."
  for i in $(seq 1 30); do
    CURRENT_TLS=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TLS_PROFILE")].value}' 2>/dev/null || true)
    if [ -z "$CURRENT_TLS" ]; then
      echo "OLM removed TLS_PROFILE from deployment (attempt $i)"
      break
    fi
    echo "  Waiting for OLM reconciliation (attempt $i/30)..."
    sleep 10
  done

else
  echo "FAIL: No subscription or CSV found for tempo operator in $TEMPO_OPERATOR_NAMESPACE"
  exit 1
fi

oc rollout status deployment/tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" --timeout=300s

echo "PASS: TLS_PROFILE and FEATURE_GATES reverted from operator"
