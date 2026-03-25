#!/bin/bash
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# --- 0. Wait for the operator to fully reconcile the CR update ---
# The operator may trigger a rolling restart after the CR update in step-09.
echo "Waiting for StatefulSet rollout to complete after CR update..."
kubectl rollout status statefulset/tempo-mono -n $NAMESPACE --timeout=300s
echo "StatefulSet rollout complete."

# --- 1. ConfigMap verification ---
# After reverting the subscription, the operator-level TLS profile is back to Intermediate (from APIServer).
# The per-CR minVersion: "1.3" should override this to Modern on the specified components.
CONFIG=$(kubectl get configmap tempo-mono-config -n $NAMESPACE -o jsonpath='{.data.tempo\.yaml}')
echo "$CONFIG"

# Verify both gRPC and HTTP receivers use per-CR override (min_version "1.3")
COUNT_13=$(echo "$CONFIG" | grep -c 'min_version: "1.3"')
if [[ "$COUNT_13" -ne 2 ]]; then
  fail "expected 2 occurrences of min_version 1.3 (gRPC + HTTP receivers from per-CR minVersion override), found $COUNT_13"
fi
echo "ConfigMap: gRPC receiver=1.3, HTTP receiver=1.3 (from per-CR minVersion override) - OK"

# Verify no receiver is still using the default 1.2
if echo "$CONFIG" | grep -q 'min_version: "1.2"'; then
  fail "found min_version 1.2 in receiver config, per-CR override should set 1.3"
fi

# Verify storage S3 TLS uses per-CR override (tls_min_version: VersionTLS13)
echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found"
echo "$CONFIG" | grep 'tls_min_version: VersionTLS13' || fail "storage tls_min_version should be VersionTLS13 (from per-CR minVersion override)"
if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS12'; then
  fail "storage still using VersionTLS12, per-CR override should set VersionTLS13"
fi
echo "ConfigMap: storage TLS minVersion=VersionTLS13 (from per-CR minVersion override) - OK"

# --- 2. Functional TLS checks ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono -port 4317 \
  || fail "TLS check failed on monolithic gRPC:4317"
echo "PASS: monolithic gRPC:4317 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono -port 4318 \
  || fail "TLS check failed on monolithic HTTP:4318"
echo "PASS: monolithic HTTP:4318 TLS functional"

echo "PASS: All per-CR TLS overrides verified - gRPC=Modern, HTTP=Modern, storage=VersionTLS13"
