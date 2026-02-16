#!/bin/bash
set -euo pipefail

# Detect FIPS mode from machineconfig
IS_FIPS=false
if kubectl get machineconfig 99-master-fips -o jsonpath='{.spec.fips}' 2>/dev/null | grep -q true; then
  IS_FIPS=true
  echo "FIPS mode detected - adjusting TLS verification"
fi

fail() { echo "FAIL: $1"; exit 1; }

# Verify TLS profile detected by tls-scanner matches expected profile for a given port.
# Args: $1=scan_output, $2=port, $3=expected_profile (intermediate|modern), $4=description, $5=min_count (default 1)
verify_tls_profile() {
  local scan_output="$1" port="$2" expected="$3" description="$4" min_count="${5:-1}"

  local port_lines
  port_lines=$(echo "$scan_output" | grep "Found TLS information for port $port" || true)
  if [ -z "$port_lines" ]; then
    fail "$description: no TLS information found for port $port"
  fi

  local count
  count=$(echo "$port_lines" | wc -l | tr -d ' ')
  if [ "$count" -lt "$min_count" ]; then
    fail "$description: expected at least $min_count endpoints on port $port, found $count"
  fi

  if [ "$expected" = "intermediate" ]; then
    local tls12_count tls13_count
    tls12_count=$(echo "$port_lines" | grep -c "TLSv1.2" || true)
    tls13_count=$(echo "$port_lines" | grep -c "TLSv1.3" || true)
    if [ "$tls12_count" -lt "$min_count" ]; then
      fail "$description: expected TLSv1.2 on all $min_count endpoints (port $port), found $tls12_count"
    fi
    if [ "$tls13_count" -lt "$min_count" ]; then
      fail "$description: expected TLSv1.3 on all $min_count endpoints (port $port), found $tls13_count"
    fi
  elif [ "$expected" = "modern" ]; then
    local tls13_count
    tls13_count=$(echo "$port_lines" | grep -c "TLSv1.3" || true)
    if [ "$tls13_count" -lt "$min_count" ]; then
      fail "$description: expected TLSv1.3 on all $min_count endpoints (port $port), found $tls13_count"
    fi
    if echo "$port_lines" | grep -q "TLSv1.2"; then
      fail "$description: port $port still accepting TLSv1.2 under Modern profile"
    fi
  fi

  echo "PASS: $description (port $port, profile=$expected, endpoints=$count)"
}

# --- 1. ConfigMap verification ---
CONFIG=$(kubectl get configmap tempo-mono-config -n $NAMESPACE -o jsonpath='{.data.tempo\.yaml}')
echo "$CONFIG"

# Verify both gRPC and HTTP receivers use Modern profile (min_version "1.3") from subscription TLS_PROFILE
COUNT_13=$(echo "$CONFIG" | grep -c 'min_version: "1.3"')
if [[ "$COUNT_13" -ne 2 ]]; then
  fail "expected 2 occurrences of min_version 1.3 (gRPC + HTTP receivers from subscription TLS_PROFILE=Modern), found $COUNT_13"
fi
echo "ConfigMap: gRPC receiver=1.3, HTTP receiver=1.3 (from subscription TLS_PROFILE=Modern) - OK"

# Verify no receiver is still using the default 1.2
if echo "$CONFIG" | grep -q 'min_version: "1.2"'; then
  fail "found min_version 1.2 in receiver config, all receivers should use Modern profile from subscription"
fi

# Verify storage S3 TLS uses Modern profile (VersionTLS13) from subscription
echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found"
echo "$CONFIG" | grep 'tls_min_version: VersionTLS13' || fail "storage tls_min_version should be VersionTLS13 (from subscription TLS_PROFILE=Modern)"
if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS12'; then
  fail "storage still using VersionTLS12, should use Modern profile from subscription"
fi
echo "ConfigMap: storage TLS minVersion=VersionTLS13 (from subscription TLS_PROFILE=Modern) - OK"

# --- 2. Functional TLS checks (tls-scanner -host <service> -port <port>) ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono -port 4317 \
  || fail "TLS check failed on monolithic gRPC:4317"
echo "PASS: monolithic gRPC:4317 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono -port 4318 \
  || fail "TLS check failed on monolithic HTTP:4318"
echo "PASS: monolithic HTTP:4318 TLS functional"

# --- 3. Comprehensive scan + profile verification ---
echo "=== Scanning all Tempo pods in $NAMESPACE ==="
SCAN_OUTPUT=$(kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner \
  -all-pods \
  -namespace-filter $NAMESPACE \
  -json-file /tmp/monolithic-scan.json 2>&1) || true
echo "$SCAN_OUTPUT"

kubectl cp $NAMESPACE/tls-scanner:/tmp/monolithic-scan.json /tmp/monolithic-scan.json 2>/dev/null || true

# Verify port 4317 (gRPC): Modern profile (TLSv1.3 only) from subscription TLS_PROFILE=Modern
verify_tls_profile "$SCAN_OUTPUT" 4317 modern "Monolithic gRPC (subscription TLS_PROFILE=Modern)"

# Verify port 4318 (HTTP): Modern profile (TLSv1.3 only) from subscription TLS_PROFILE=Modern
verify_tls_profile "$SCAN_OUTPUT" 4318 modern "Monolithic HTTP (subscription TLS_PROFILE=Modern)"

echo "PASS: All TLS profile settings verified from subscription TLS_PROFILE=Modern - gRPC=Modern, HTTP=Modern, storage=VersionTLS13"
