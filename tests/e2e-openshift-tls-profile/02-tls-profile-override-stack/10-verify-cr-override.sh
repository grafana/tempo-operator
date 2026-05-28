#!/bin/bash
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# --- 0. Wait for the operator to fully reconcile the CR update ---
# The operator may trigger rolling restarts after the CR update in step-09.
echo "Waiting for deployment rollouts to complete after CR update..."
kubectl rollout status deploy/tempo-simplest-distributor -n $NAMESPACE --timeout=300s
kubectl rollout status deploy/tempo-simplest-gateway -n $NAMESPACE --timeout=300s
echo "Rollouts complete."

# --- 1. ConfigMap verification ---
# After reverting the subscription, the operator-level TLS profile is back to the default
# (Intermediate from the APIServer clusterTLSPolicy). The per-CR storage minVersion: "1.3"
# override should force VersionTLS13 on the storage connection.
CONFIG=$(kubectl get configmap tempo-simplest -n $NAMESPACE -o jsonpath='{.data.tempo\.yaml}')
echo "$CONFIG"

# Verify storage TLS uses per-CR override (VersionTLS13)
echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found - storage TLS not configured"
echo "$CONFIG" | grep 'tls_min_version: VersionTLS13' || fail "storage tls_min_version should be VersionTLS13 (from per-CR minVersion override)"
if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS12'; then
  fail "storage still using VersionTLS12, per-CR override should set VersionTLS13"
fi
echo "ConfigMap: storage TLS minVersion=VersionTLS13 (from per-CR minVersion override) - OK"

# --- 2. Functional TLS checks ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8080 \
  || fail "TLS check failed on gateway:8080"
echo "PASS: gateway:8080 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8090 \
  || fail "TLS check failed on gateway:8090"
echo "PASS: gateway:8090 TLS functional"

echo "PASS: Per-CR TLS overrides verified - storage=VersionTLS13"
