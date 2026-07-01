#!/bin/bash
set -euo pipefail

log() {
    echo -e "\033[1;34m==>\033[0m \033[1m$1\033[0m"
}

PLATFORM=${PLATFORM:-auto}
if [ "$PLATFORM" = "auto" ]; then
    if kubectl api-resources --api-group=route.openshift.io &>/dev/null; then
        PLATFORM=openshift
    elif kind get clusters &>/dev/null 2>&1; then
        PLATFORM=kind
    else
        PLATFORM=kubernetes
    fi
    log "Detected platform: $PLATFORM"
else
    log "Using platform: $PLATFORM"
fi

case "$PLATFORM" in
    kind|kubernetes|openshift)
        log "Installing cert-manager"
        make cert-manager
        ;;
esac

export IMG_PREFIX=${IMG_PREFIX:-quay.io/$USER}
export OPERATOR_VERSION=$(date +%s).0.0
case "$PLATFORM" in
    openshift-olm)
        export OPERATOR_NAMESPACE=openshift-tempo-operator
        ;;
    *)
        export OPERATOR_NAMESPACE=tempo-operator-system
        ;;
esac
case "$PLATFORM" in
    openshift|openshift-olm)
        export BUNDLE_VARIANT=openshift
esac

log "Building operator image ${IMG_PREFIX}/tempo-operator:v${OPERATOR_VERSION}"
make docker-build

case "$PLATFORM" in
    kind)
        log "Loading image into kind cluster"
        kind load docker-image ${IMG_PREFIX}/tempo-operator:v${OPERATOR_VERSION}
        ;;
    *)
        log "Pushing image to registry"
        make docker-push
        ;;
esac

case "$PLATFORM" in
    openshift-olm)
        log "Building and pushing bundle"
        make bundle bundle-build bundle-push

        if kubectl get namespace "$OPERATOR_NAMESPACE" &>/dev/null; then
            log "Upgrading operator via OLM"
            make olm-upgrade
        else
            log "Deploying operator via OLM"
            kubectl create namespace "$OPERATOR_NAMESPACE"
            make olm-deploy
        fi
        ;;
    *)
        log "Deploying operator"
        make deploy
        ;;
esac

make reset

log "Waiting for operator rollout"
kubectl -n "$OPERATOR_NAMESPACE" rollout status deployment/tempo-operator-controller

log "Deploy complete"
