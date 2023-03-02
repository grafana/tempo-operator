#!/bin/bash

if kubectl api-resources --api-group=route.openshift.io | grep -q Route; then
    echo "OpenShift Routes are already installed."
else
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/router_rbac.yaml
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/route_crd.yaml
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/router.yaml
    kubectl wait --for=condition=available deployment/ingress-router -n openshift-ingress --timeout=5m
fi
