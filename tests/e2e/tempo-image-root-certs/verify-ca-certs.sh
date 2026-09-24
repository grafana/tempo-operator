#!/bin/bash
set -euo pipefail

POD=tempo-simplest-ingester-0
CONTAINER=verify-root-certs

# Start ephemeral container with ubi to check if the tempo container has root certificates installed
kubectl debug -n "$NAMESPACE" "$POD" \
  --image=registry.access.redhat.com/ubi9/ubi-minimal \
  --target=tempo \
  --container="$CONTAINER" \
  --profile=sysadmin \
  --quiet \
  -- /bin/bash -c '
for bundle in /etc/ssl/certs/ca-certificates.crt \
              /etc/pki/tls/certs/ca-bundle.crt \
              /etc/ssl/ca-bundle.pem \
              /etc/pki/tls/cacert.pem \
              /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem \
              /etc/ssl/cert.pem; do
  path="/proc/1/root$bundle"
  if [ ! -f "$path" ]; then
    echo "$bundle: skipped (not a readable file)"
    continue
  fi

  count=$(grep -c "BEGIN CERTIFICATE" "$path" || true)
  echo "$bundle: $count certificates"
  if [ "$count" -gt 100 ]; then
    echo "PASS"
    break
  fi
done
'

# Wait for the ephemeral container to terminate.
for _ in $(seq 1 60); do
  reason=$(kubectl get pod -n "$NAMESPACE" "$POD" \
    -o jsonpath="{.status.ephemeralContainerStatuses[?(@.name==\"$CONTAINER\")].state.terminated.reason}")
  [ -n "$reason" ] && break
  sleep 5
done

OUTPUT=$(kubectl logs -n "$NAMESPACE" "$POD" -c "$CONTAINER")
echo "$OUTPUT"

if ! echo "$OUTPUT" | grep -q "^PASS$"; then
  echo "FAIL: no or not enough root certificates found in the Tempo image"
  exit 1
fi
