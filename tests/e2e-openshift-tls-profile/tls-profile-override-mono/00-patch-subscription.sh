#!/bin/bash
# Patch the operator to set TLS_PROFILE=Modern via env var.
# Also overrides FEATURE_GATES to remove openshift.clusterTLSPolicy since that takes precedence
# over TLS_PROFILE by fetching the profile from the APIServer CR instead.
# Supports operators installed via:
#   1. OLM subscription (e.g. from catalog)
#   2. operator-sdk run bundle (CSV exists but no subscription)
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

# Get current FEATURE_GATES from the deployment
CURRENT_FEATURE_GATES=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="FEATURE_GATES")].value}')

# Remove openshift.clusterTLSPolicy from FEATURE_GATES
NEW_FEATURE_GATES=$(echo "$CURRENT_FEATURE_GATES" | python3 -c "
import sys
gates = sys.stdin.read().strip()
filtered = ','.join(g.strip() for g in gates.split(',') if g.strip() != 'openshift.clusterTLSPolicy')
print(filtered)
")

echo "Current FEATURE_GATES: $CURRENT_FEATURE_GATES"
echo "New FEATURE_GATES: $NEW_FEATURE_GATES"

if [ -n "$TEMPO_OPERATOR_SUB" ]; then
  # OLM install with subscription: patch subscription
  echo "Patching subscription $TEMPO_OPERATOR_SUB in $TEMPO_OPERATOR_NAMESPACE with TLS_PROFILE=Modern"

  # Get current subscription config.env
  CURRENT_ENV=$(oc get subscription "$TEMPO_OPERATOR_SUB" -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.config.env}' 2>/dev/null || echo "[]")

  # Build new env array with TLS_PROFILE and modified FEATURE_GATES
  NEW_ENV=$(echo "$CURRENT_ENV" | python3 -c "
import json, sys
fg = '$NEW_FEATURE_GATES'
try:
    env = json.load(sys.stdin)
    if not isinstance(env, list):
        env = []
except:
    env = []
# Remove existing TLS_PROFILE and FEATURE_GATES if present
env = [e for e in env if e.get('name') not in ('TLS_PROFILE', 'FEATURE_GATES')]
env.append({'name': 'TLS_PROFILE', 'value': 'Modern'})
env.append({'name': 'FEATURE_GATES', 'value': fg})
print(json.dumps(env))
")

  oc patch subscription "$TEMPO_OPERATOR_SUB" -n "$TEMPO_OPERATOR_NAMESPACE" \
      --type='merge' -p "{\"spec\": {\"config\": {\"env\": $NEW_ENV}}}"

  # Wait for OLM to propagate the subscription change to the deployment.
  echo "Waiting for OLM to update the deployment with TLS_PROFILE=Modern..."
  for i in $(seq 1 30); do
    CURRENT_TLS=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TLS_PROFILE")].value}' 2>/dev/null || true)
    if [ "$CURRENT_TLS" = "Modern" ]; then
      echo "OLM propagated TLS_PROFILE=Modern to deployment (attempt $i)"
      break
    fi
    echo "  Waiting for OLM propagation (attempt $i/30)..."
    sleep 10
  done

elif [ -n "$TEMPO_OPERATOR_CSV" ]; then
  # operator-sdk bundle install: patch CSV deployment spec (OLM reconciles deployment from CSV)
  echo "No subscription found, patching CSV $TEMPO_OPERATOR_CSV in $TEMPO_OPERATOR_NAMESPACE with TLS_PROFILE=Modern"

  # Build the new env array for the CSV patch using JSON patch type to avoid annotation size limits.
  NEW_ENV=$(oc get csv "$TEMPO_OPERATOR_CSV" -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.install.spec.deployments[0].spec.template.spec.containers[0].env}' | python3 -c "
import json, sys
fg = '$NEW_FEATURE_GATES'
env = json.load(sys.stdin)
env = [e for e in env if e.get('name') not in ('TLS_PROFILE', 'FEATURE_GATES')]
env.append({'name': 'TLS_PROFILE', 'value': 'Modern'})
env.append({'name': 'FEATURE_GATES', 'value': fg})
print(json.dumps(env))
")

  oc patch csv "$TEMPO_OPERATOR_CSV" -n "$TEMPO_OPERATOR_NAMESPACE" --type json \
      -p "[{\"op\": \"replace\", \"path\": \"/spec/install/spec/deployments/0/spec/template/spec/containers/0/env\", \"value\": $NEW_ENV}]"

  # Wait for OLM to reconcile the CSV change to the deployment
  echo "Waiting for OLM to reconcile CSV change to deployment..."
  for i in $(seq 1 30); do
    CURRENT_TLS=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
      -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TLS_PROFILE")].value}' 2>/dev/null || true)
    if [ "$CURRENT_TLS" = "Modern" ]; then
      echo "OLM reconciled TLS_PROFILE=Modern to deployment (attempt $i)"
      break
    fi
    echo "  Waiting for OLM reconciliation (attempt $i/30)..."
    sleep 10
  done

else
  echo "FAIL: No subscription or CSV found for tempo operator in $TEMPO_OPERATOR_NAMESPACE"
  exit 1
fi

# Wait for the operator deployment rollout to complete with the new env vars
echo "Waiting for operator deployment rollout..."
oc rollout status deployment/tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" --timeout=300s

# Verify the TLS_PROFILE env var is set in the deployment
TLS_PROFILE_VALUE=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="TLS_PROFILE")].value}')
if [ "$TLS_PROFILE_VALUE" != "Modern" ]; then
    echo "FAIL: TLS_PROFILE env var not set to Modern, got: '$TLS_PROFILE_VALUE'"
    exit 1
fi

# Verify openshift.clusterTLSPolicy is no longer in FEATURE_GATES
FEATURE_GATES=$(oc get deployment tempo-operator-controller -n "$TEMPO_OPERATOR_NAMESPACE" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="FEATURE_GATES")].value}')
if echo "$FEATURE_GATES" | grep -q "openshift.clusterTLSPolicy"; then
    echo "FAIL: openshift.clusterTLSPolicy still present in FEATURE_GATES: $FEATURE_GATES"
    exit 1
fi
echo "Deployment FEATURE_GATES: $FEATURE_GATES"

if [ -n "$TEMPO_OPERATOR_SUB" ] || [ -n "$TEMPO_OPERATOR_CSV" ]; then
  # Verify the CSV is still healthy (for OLM-managed installs)
  CSV_NAME="${TEMPO_OPERATOR_CSV:-$(oc get csv -n "$TEMPO_OPERATOR_NAMESPACE" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep tempo)}"
  if oc get csv "$CSV_NAME" -n "$TEMPO_OPERATOR_NAMESPACE" -o jsonpath='{.status.phase}' | grep -qi "Succeeded"; then
      echo "CSV $CSV_NAME is healthy"
  else
      echo "Operator CSV update failed, exiting with error."
      exit 1
  fi
fi

echo "PASS: Operator patched with TLS_PROFILE=Modern (openshift.clusterTLSPolicy disabled)"
