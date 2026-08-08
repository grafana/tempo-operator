#!/bin/bash
set -euo pipefail

log() {
    echo -e "\033[1;34m==>\033[0m \033[1m$1\033[0m"
}

OTEL_OPERATOR_VERSION=v0.156.0

if kubectl get crd opentelemetrycollectors.opentelemetry.io &>/dev/null; then
    log "OpenTelemetry operator is already installed, skipping"
    exit 0
fi

log "Deploying OpenTelemetry operator ${OTEL_OPERATOR_VERSION}"
kubectl apply -f "https://github.com/open-telemetry/opentelemetry-operator/releases/download/${OTEL_OPERATOR_VERSION}/opentelemetry-operator.yaml"

log "Waiting for deployment to be created"
kubectl wait --for=create deployment/opentelemetry-operator-controller-manager -n opentelemetry-operator-system --timeout=5m

log "Waiting for rollout to complete"
kubectl rollout status deployment opentelemetry-operator-controller-manager -n opentelemetry-operator-system --timeout=5m

log "OpenTelemetry operator deployed successfully"
