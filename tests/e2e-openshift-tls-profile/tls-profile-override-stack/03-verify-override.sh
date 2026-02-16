#!/bin/bash
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# Verify TLS profile from tls-scanner -all-pods output for a given port.
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

# Verify TLS profile via direct nmap ssl-enum-ciphers scan on a specific IP:port.
# Uses nmap without -sV to avoid "tcpwrapped" misdetection on HTTP ports.
# Args: $1=ip, $2=ports (comma-separated), $3=expected_profile, $4=description
verify_nmap_tls_profile() {
  local ip="$1" ports="$2" expected="$3" description="$4"

  echo "=== nmap ssl-enum-ciphers: $description ($ip ports $ports) ==="
  local result
  result=$(kubectl exec tls-scanner -n $NAMESPACE -- nmap -Pn --script ssl-enum-ciphers -p "$ports" "$ip")
  echo "$result"

  for port in ${ports//,/ }; do
    local port_section
    port_section=$(echo "$result" | grep -A 50 "${port}/tcp" || true)
    if [ -z "$port_section" ]; then
      fail "$description: port $port not found in nmap output"
    fi

    if [ "$expected" = "intermediate" ]; then
      echo "$port_section" | grep "TLSv1.2" || fail "$description: port $port missing TLSv1.2"
      echo "$port_section" | grep "TLSv1.3" || fail "$description: port $port missing TLSv1.3"
    elif [ "$expected" = "modern" ]; then
      echo "$port_section" | grep "TLSv1.3" || fail "$description: port $port missing TLSv1.3"
      if echo "$port_section" | head -30 | grep -q "TLSv1.2"; then
        fail "$description: port $port still accepting TLSv1.2 under Modern profile"
      fi
    fi
  done

  echo "PASS: $description (ports $ports, profile=$expected)"
}

# --- 1. ConfigMap verification ---
CONFIG=$(kubectl get configmap tempo-simplest -n $NAMESPACE -o jsonpath='{.data.tempo\.yaml}')
echo "$CONFIG"

# Verify storage TLS overridden to 1.3
echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found - storage TLS not configured"
echo "$CONFIG" | grep 'tls_min_version: 1.3' || fail "storage tls_min_version should be 1.3"
if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS12'; then
  fail "storage still using VersionTLS12, should be overridden to 1.3"
fi
echo "ConfigMap: storage TLS minVersion=1.3 - OK"

# Verify distributor receivers use cluster-wide TLS profile (Intermediate, no CR override).
# With gateway enabled, traces flow through the gateway, not the distributor receiver ports.
echo "$CONFIG" | grep 'min_version: "1.2"' || fail "distributor receiver min_version should be 1.2 (cluster default)"
echo "ConfigMap: distributor receiver TLS minVersion=1.2 (cluster default) - OK"

# --- 2. Gateway args verification ---
# Gateway uses the cluster-wide TLS profile (Intermediate by default).
GW_ARGS=$(kubectl get deploy tempo-simplest-gateway -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].args}')
echo "$GW_ARGS" | grep -- '--tls.min-version=VersionTLS12' || fail "gateway --tls.min-version missing"
echo "$GW_ARGS" | grep -- '--tls.cipher-suites=' || fail "gateway --tls.cipher-suites missing"
echo "Gateway args: cluster default Intermediate - OK"

# --- 3. Functional TLS checks ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8080 \
  || fail "TLS check failed on gateway:8080"
echo "PASS: gateway:8080 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8090 \
  || fail "TLS check failed on gateway:8090"
echo "PASS: gateway:8090 TLS functional"

# --- 4. Gateway TLS profile verification via direct nmap ---
# Gateway uses the cluster-wide TLS profile (Intermediate).
GATEWAY_IP=$(kubectl get pod -n $NAMESPACE -l app.kubernetes.io/component=gateway -o jsonpath='{.items[0].status.podIP}')
verify_nmap_tls_profile "$GATEWAY_IP" "8080,8090" intermediate "Gateway (cluster default)"

# --- 5. Internal gRPC TLS profile verification via -all-pods ---
echo "=== Scanning all Tempo pods in $NAMESPACE ==="
SCAN_OUTPUT=$(kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner \
  -all-pods \
  -namespace-filter $NAMESPACE \
  -json-file /tmp/override-scan.json 2>&1) || true
echo "$SCAN_OUTPUT"

kubectl cp $NAMESPACE/tls-scanner:/tmp/override-scan.json /tmp/override-scan.json 2>/dev/null || true

# Internal gRPC (port 9095) uses cluster-wide TLS profile (Intermediate)
verify_tls_profile "$SCAN_OUTPUT" 9095 intermediate "Internal gRPC (cluster default)" 2

echo "PASS: TLS overrides verified - storage=1.3 (CR override), gateway=Intermediate (cluster default), internal gRPC=Intermediate (cluster default), distributor receivers=Intermediate (cluster default)"
