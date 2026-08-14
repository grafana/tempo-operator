#!/bin/bash
set -euo pipefail

log() {
    echo -e "\033[1;34m==>\033[0m \033[1m$1\033[0m"
}

PLATFORM=${PLATFORM:-auto}
if [ "$PLATFORM" = "auto" ]; then
    if kubectl get clusterversion.config.openshift.io &>/dev/null; then
        PLATFORM=openshift-olm
    else
        PLATFORM=kubernetes
    fi
    log "Detected platform: $PLATFORM"
else
    log "Using platform: $PLATFORM"
fi

if kubectl get crd opentelemetrycollectors.opentelemetry.io &>/dev/null; then
    log "OpenTelemetry operator is already installed, skipping"
    exit 0
fi

case "$PLATFORM" in
    openshift-olm)
        log "Deploying OpenTelemetry operator via OLM"
        kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-opentelemetry-operator
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: openshift-opentelemetry-operator
  namespace: openshift-opentelemetry-operator
spec:
  upgradeStrategy: Default
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: opentelemetry-product
  namespace: openshift-opentelemetry-operator
spec:
  channel: stable
  installPlanApproval: Automatic
  name: opentelemetry-product
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
        ;;
    *)
        OTEL_OPERATOR_VERSION=${OTEL_OPERATOR_VERSION:-v0.156.0}
        log "Deploying OpenTelemetry operator ${OTEL_OPERATOR_VERSION}"
        kubectl apply -f "https://github.com/open-telemetry/opentelemetry-operator/releases/download/${OTEL_OPERATOR_VERSION}/opentelemetry-operator.yaml"
        ;;
esac

case "$PLATFORM" in
    openshift-olm)
        log "Waiting for deployment to be created"
        kubectl -n openshift-opentelemetry-operator wait --for=create deployment/opentelemetry-operator-controller-manager --timeout=5m

        log "Waiting for rollout to complete"
        kubectl -n openshift-opentelemetry-operator rollout status deployment/opentelemetry-operator-controller-manager --timeout=5m
        ;;
    *)
        log "Waiting for deployment to be created"
        kubectl -n opentelemetry-operator-system wait --for=create deployment/opentelemetry-operator-controller-manager --timeout=5m

        log "Waiting for rollout to complete"
        kubectl -n opentelemetry-operator-system rollout status deployment/opentelemetry-operator-controller-manager --timeout=5m
        ;;
esac

log "OpenTelemetry operator deployed successfully"
