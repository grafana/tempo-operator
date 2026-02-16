#!/bin/bash
# Verify that extraConfig settings from the TempoStack CR are applied to the ConfigMap.
# Uses grep-based checks instead of exact YAML matching so it works regardless of
# TLS profile (community uses Modern/TLS1.3, OpenShift uses Intermediate/TLS1.2).
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# Check both tempo-query-frontend.yaml and tempo.yaml keys
for key in tempo-query-frontend.yaml tempo.yaml; do
  echo "=== Checking $key ==="
  CONFIG=$(kubectl get configmap tempo-simplest -n $NAMESPACE -o jsonpath="{.data.${key//./\\.}}")

  [ -z "$CONFIG" ] && fail "$key not found in ConfigMap"

  # Verify extraConfig overrides are applied
  echo "$CONFIG" | grep -q 'query_timeout: 180s' \
    || fail "$key: extraConfig querier.search.query_timeout=180s not applied"
  echo "$CONFIG" | grep -q 'max_retries: 3' \
    || fail "$key: extraConfig query_frontend.max_retries=3 not applied"
  echo "$CONFIG" | grep -q 'http_server_read_timeout: 10m' \
    || fail "$key: extraConfig server.http_server_read_timeout=10m not applied"
  echo "$CONFIG" | grep -q 'http_server_write_timeout: 10m' \
    || fail "$key: extraConfig server.http_server_write_timeout=10m not applied"

  echo "PASS: $key has all extraConfig overrides"
done

# Verify tempo-query.yaml exists (Jaeger query is enabled)
QUERY_CONFIG=$(kubectl get configmap tempo-simplest -n $NAMESPACE -o jsonpath='{.data.tempo-query\.yaml}')
[ -z "$QUERY_CONFIG" ] && fail "tempo-query.yaml not found in ConfigMap"
echo "$QUERY_CONFIG" | grep -q 'backend: localhost:3200' \
  || fail "tempo-query.yaml: backend not set"
echo "PASS: tempo-query.yaml present and configured"

echo "PASS: All extraConfig settings verified"
