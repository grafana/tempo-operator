#!/bin/bash

# Create a temporary directory to store must-gather
MUST_GATHER_DIR=$(mktemp -d)

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest -- /usr/bin/must-gather --operator-namespace $temponamespace

# Define required files and directories
REQUIRED_ITEMS=(
  "event-filter.html"
  "timestamp"
  "*sha*/deployment-tempo-operator-controller.yaml"
  "*sha*/olm/installplan-install-*"
  "*sha*/olm/clusterserviceversion-tempo-operator-*.yaml"
  "*sha*/olm/operator-opentelemetry-product-openshift-opentelemetry-operator.yaml"
  "*sha*/olm/operator-*-tempo-operator.yaml"
  "*sha*/olm/subscription-tempo-*.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-distributor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-ingester.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-distributor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-querier.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/configmap-tempo-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-compactor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-querier.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/tempostack-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/serviceaccount-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/statefulset-tempo-simplest-ingester.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/route-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-gossip-ring.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/configmap-tempo-simplest-ca-bundle.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/serviceaccount-tempo-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-compactor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-query-frontend-discovery.yaml"
  "*sha*/tempo-operator-controller-*"
)

# Verify each required item
for item in "${REQUIRED_ITEMS[@]}"; do
  if ! find "$MUST_GATHER_DIR" -path "$MUST_GATHER_DIR/$item" -print -quit | grep -q .; then
    echo "Missing: $item"
    exit 1
  else
    echo "Found: $item"
  fi
done

# Cleanup the must-gather directory
rm -rf $MUST_GATHER_DIR
