#!/bin/bash
set -euo pipefail

OUTPUT=$(kubectl debug -n "$NAMESPACE" tempo-simplest-ingester-0 \
  --image=registry.access.redhat.com/ubi9/ubi-minimal \
  --target=tempo \
  --container=verify-root-certs \
  --profile=sysadmin \
  --quiet \
  --attach \
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
')

echo "$OUTPUT"

if ! echo "$OUTPUT" | grep -q "^PASS$"; then
  echo "FAIL: no or not enough root certificates found in the Tempo image"
  exit 1
fi
