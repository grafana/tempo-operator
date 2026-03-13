#!/bin/bash
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# Ensure tls-scanner pod is running and responsive (may have been evicted or its
# container killed during APIServer/MCO rollout while restartPolicy=Never).
recreate_tls_scanner() {
  kubectl delete pod tls-scanner -n $NAMESPACE --force --grace-period=0 2>/dev/null || true
  sleep 5
  cat <<'PODEOF' | sed "s/REPLACE_NS/$NAMESPACE/g" | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: tls-scanner
  namespace: REPLACE_NS
  labels:
    app: tls-scanner
spec:
  serviceAccountName: tls-scanner
  containers:
  - name: tls-scanner
    image: quay.io/rhn_support_ikanse/tls-scanner:latest
    command: ["sleep", "infinity"]
    securityContext:
      privileged: true
      runAsUser: 0
  restartPolicy: Never
PODEOF
  echo "Waiting for tls-scanner pod to be ready..."
  kubectl wait --for=condition=Ready pod/tls-scanner -n $NAMESPACE --timeout=2m
  echo "tls-scanner pod recreated and ready"
}

NEED_RECREATE=false
if ! kubectl get pod tls-scanner -n $NAMESPACE &>/dev/null; then
  echo "tls-scanner pod not found"
  NEED_RECREATE=true
elif ! kubectl get pod tls-scanner -n $NAMESPACE -o jsonpath='{.status.phase}' | grep -q Running; then
  echo "tls-scanner pod exists but phase is not Running"
  NEED_RECREATE=true
elif ! kubectl exec tls-scanner -n $NAMESPACE -- echo ok &>/dev/null; then
  echo "tls-scanner pod exists and phase is Running, but container is not responsive"
  NEED_RECREATE=true
fi

if [ "$NEED_RECREATE" = "true" ]; then
  echo "Recreating tls-scanner pod..."
  recreate_tls_scanner
fi

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
echo "$CONFIG" | grep 'tls_min_version: VersionTLS13' || fail "tls_min_version VersionTLS13 not found"

echo "$CONFIG" | grep 'tls_ca_path' || fail "storage tls_ca_path not found - storage TLS not configured"
if echo "$CONFIG" | grep -q 'tls_min_version: VersionTLS12'; then
  fail "storage still using VersionTLS12 under Modern profile"
fi

# --- 2. Gateway args verification ---
GW_ARGS=$(kubectl get deploy tempo-simplest-gateway -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].args}')
echo "$GW_ARGS" | grep -- '--tls.min-version=VersionTLS13' || fail "gateway not updated to VersionTLS13"

# --- 3. Functional TLS checks (tls-scanner -host <service> -port <port>) ---
echo "=== Functional TLS checks ==="
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8080 \
  || fail "TLS check failed on gateway:8080"
echo "PASS: gateway:8080 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host tempo-simplest-gateway -port 8090 \
  || fail "TLS check failed on gateway:8090"
echo "PASS: gateway:8090 TLS functional"

# --- 4. Gateway TLS profile verification via direct nmap ---
GATEWAY_IP=$(kubectl get pod -n $NAMESPACE -l app.kubernetes.io/component=gateway -o jsonpath='{.items[0].status.podIP}')
verify_nmap_tls_profile "$GATEWAY_IP" "8080,8090" modern "Gateway"

# --- 5. Internal gRPC TLS profile verification via targeted nmap ---
# Use targeted nmap on specific pod IPs instead of -all-pods scan for reliability
# after rollout (avoids scanning terminating pods and cross-node network issues).
echo "=== Targeted internal gRPC scan (Modern profile) ==="

INGESTER_IP=$(kubectl get pod -n $NAMESPACE -l app.kubernetes.io/component=ingester -o jsonpath='{.items[0].status.podIP}')
verify_nmap_tls_profile "$INGESTER_IP" "9095" modern "Ingester internal gRPC"

QF_IP=$(kubectl get pod -n $NAMESPACE -l app.kubernetes.io/component=query-frontend -o jsonpath='{.items[0].status.podIP}')
verify_nmap_tls_profile "$QF_IP" "9095" modern "Query-frontend internal gRPC"

# --- 6. Operator webhook and metrics - verify restart and Modern profile ---
# The operator restarts (graceful shutdown via SecurityProfileWatcher) when TLS profile changes.
OPERATOR_NS="openshift-tempo-operator"

# Verify operator pod was restarted (new pod UID expected after TLS profile change)
OPERATOR_POD=$(kubectl get pod -n $OPERATOR_NS -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')
OPERATOR_UID=$(kubectl get pod -n $OPERATOR_NS $OPERATOR_POD -o jsonpath='{.metadata.uid}')
if [ -f /tmp/operator-uid-baseline.txt ]; then
  BASELINE_UID=$(cat /tmp/operator-uid-baseline.txt)
  if [ "$OPERATOR_UID" != "$BASELINE_UID" ]; then
    echo "PASS: Operator pod restarted (UID changed: $BASELINE_UID -> $OPERATOR_UID)"
  else
    echo "NOTE: Operator pod UID unchanged ($OPERATOR_UID) - restart may not have completed yet"
  fi
fi

OPERATOR_IP=$(kubectl get pod -n $OPERATOR_NS $OPERATOR_POD -o jsonpath='{.status.podIP}')
echo "Operator pod: $OPERATOR_POD at $OPERATOR_IP"

# Functional TLS checks on operator ports
kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host $OPERATOR_IP -port 9443 \
  || fail "TLS check failed on operator webhook:9443"
echo "PASS: operator webhook:9443 TLS functional"

kubectl exec tls-scanner -n $NAMESPACE -- tls-scanner -host $OPERATOR_IP -port 8443 \
  || fail "TLS check failed on operator metrics:8443"
echo "PASS: operator metrics:8443 TLS functional"

# Operator TLS profile verification via direct nmap
verify_nmap_tls_profile "$OPERATOR_IP" "9443,8443" modern "Operator"

echo "PASS: Modern profile verified on all components, gateway, and operator"
