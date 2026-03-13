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
echo "$CONFIG" | grep 'tls_min_version: VersionTLS12' || fail "tls_min_version VersionTLS12 not found"
echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found - storage TLS not configured"

# --- 2. Gateway args verification ---
GW_ARGS=$(kubectl get deploy tempo-simplest-gateway -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].args}')
echo "$GW_ARGS" | grep -- '--tls.min-version=VersionTLS12' || fail "gateway --tls.min-version missing"
echo "$GW_ARGS" | grep -- '--tls.cipher-suites=' || fail "gateway --tls.cipher-suites missing"

# --- 3. Functional TLS checks (tls-scanner -host <service> -port <port>) ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8080 \
  || fail "TLS check failed on gateway:8080"
echo "PASS: gateway:8080 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8090 \
  || fail "TLS check failed on gateway:8090"
echo "PASS: gateway:8090 TLS functional"

# --- 4. Gateway TLS profile verification via direct nmap ---
# Note: tls-scanner -all-pods uses nmap -sV which misdetects gateway HTTP port 8080
# as "tcpwrapped". Using nmap without -sV for gateway ports gives reliable TLS detection.
GATEWAY_IP=$(kubectl get pod -n $NAMESPACE -l app.kubernetes.io/component=gateway -o jsonpath='{.items[0].status.podIP}')
verify_nmap_tls_profile "$GATEWAY_IP" "8080,8090" intermediate "Gateway"

# --- 5. Internal gRPC TLS profile verification via -all-pods ---
echo "=== Scanning all Tempo pods in $NAMESPACE ==="
SCAN_OUTPUT=$(kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner \
  -all-pods \
  -namespace-filter $NAMESPACE \
  -json-file /tmp/intermediate-scan.json 2>&1) || true
echo "$SCAN_OUTPUT"

kubectl cp $NAMESPACE/tls-scanner:/tmp/intermediate-scan.json /tmp/intermediate-scan.json 2>/dev/null || true

# Verify internal gRPC TLS profile matches Intermediate on components with port 9095 (ingester, query-frontend)
verify_tls_profile "$SCAN_OUTPUT" 9095 intermediate "Internal gRPC" 2

# --- 6. Operator webhook and metrics ---
OPERATOR_NS="openshift-tempo-operator"
OPERATOR_IP=$(kubectl get pod -n $OPERATOR_NS -l control-plane=controller-manager -o jsonpath='{.items[0].status.podIP}')
OPERATOR_POD=$(kubectl get pod -n $OPERATOR_NS -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')
echo "Operator pod: $OPERATOR_POD at $OPERATOR_IP"

# Save operator pod UID to verify restart after TLS profile change to Modern
OPERATOR_UID=$(kubectl get pod -n $OPERATOR_NS $OPERATOR_POD -o jsonpath='{.metadata.uid}')
echo "$OPERATOR_UID" > /tmp/operator-uid-baseline.txt
echo "Operator UID baseline: $OPERATOR_UID"

# Functional TLS checks on operator ports
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host $OPERATOR_IP -port 9443 \
  || fail "TLS check failed on operator webhook:9443"
echo "PASS: operator webhook:9443 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host $OPERATOR_IP -port 8443 \
  || fail "TLS check failed on operator metrics:8443"
echo "PASS: operator metrics:8443 TLS functional"

# Operator TLS profile verification via direct nmap
verify_nmap_tls_profile "$OPERATOR_IP" "9443,8443" intermediate "Operator"

echo "PASS: Intermediate profile verified on all components, gateway, storage, and operator"
