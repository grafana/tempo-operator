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
COUNT_13=$(echo "$CONFIG" | grep -c 'min_version: "1.3"' || true)
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

# --- 2. Gateway args verification ---
# Gateway uses Modern profile from subscription TLS_PROFILE=Modern.
MONO_GW_ARGS=$(kubectl get statefulset tempo-mono -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[?(@.name=="tempo-gateway")].args}')
echo "$MONO_GW_ARGS" | grep -- '--tls.min-version=VersionTLS13' || fail "gateway --tls.min-version should be VersionTLS13 (from subscription TLS_PROFILE=Modern)"
echo "PASS: gateway --tls.min-version=VersionTLS13"
echo "$MONO_GW_ARGS" | grep -- '--tls.cipher-suites=' || fail "gateway --tls.cipher-suites missing"
echo "PASS: gateway --tls.cipher-suites present"
echo "Gateway args: Modern profile from subscription TLS_PROFILE=Modern - OK"

# --- 3. Functional TLS checks (tls-scanner -host <service> -port <port>) ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono-gateway -port 8080 \
  || fail "TLS check failed on gateway HTTP:8080"
echo "PASS: gateway HTTP:8080 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-mono-gateway -port 8090 \
  || fail "TLS check failed on gateway gRPC:8090"
echo "PASS: gateway gRPC:8090 TLS functional"

# --- 4. Gateway TLS profile verification via direct nmap ---
# Note: tls-scanner -all-pods uses nmap -sV which can misdetect gateway HTTP port 8080
# as "tcpwrapped". Using nmap without -sV for gateway ports gives reliable TLS detection.
MONO_GW_IP=$(kubectl get pod -n $NAMESPACE tempo-mono-0 -o jsonpath='{.status.podIP}')

echo "=== nmap ssl-enum-ciphers: Monolithic Gateway ($MONO_GW_IP ports 8080,8090) ==="
NMAP_RESULT=$(kubectl exec tls-scanner -n $NAMESPACE -- nmap -Pn --script ssl-enum-ciphers -p 8080,8090 "$MONO_GW_IP")
echo "$NMAP_RESULT"

for port in 8080 8090; do
  PORT_SECTION=$(echo "$NMAP_RESULT" | grep -A 50 "${port}/tcp" || true)
  if [ -z "$PORT_SECTION" ]; then
    fail "Monolithic gateway: port $port not found in nmap output"
  fi
  echo "$PORT_SECTION" | grep "TLSv1.3" || fail "Monolithic gateway: port $port missing TLSv1.3 under Modern profile"
  if echo "$PORT_SECTION" | head -30 | grep -q "TLSv1.2"; then
    fail "Monolithic gateway: port $port still accepting TLSv1.2 under Modern profile"
  fi
  echo "PASS: Monolithic gateway port $port = Modern (TLSv1.3 only)"
done

echo "PASS: All TLS profile settings verified from subscription TLS_PROFILE=Modern - gateway HTTP=Modern, gateway gRPC=Modern, storage=VersionTLS13"
