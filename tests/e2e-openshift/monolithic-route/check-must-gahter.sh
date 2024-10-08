#!/bin/bash

# Check if must gather directory exists
MUST_GATHER_DIR=/tmp/monolithic-route
mkdir -p $MUST_GATHER_DIR

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest -- /usr/bin/must-gather --operator-namespace tempo-operator

# Define required files and directories
REQUIRED_ITEMS=(
  "event-filter.html"
  "timestamp"
  "*sha*/deployment-tempo-operator-controller.yaml"
  "*sha*/olm/operator-servicemeshoperator-openshift-operators.yaml"
  "*sha*/olm/installplan-install-*.yaml"
  "*sha*/olm/clusterserviceversion-tempo-operator-*.yaml"
  "*sha*/olm/operator-opentelemetry-product-openshift-opentelemetry-operator.yaml"
  "*sha*/olm/operator-tempo-operator-tempo-operator.yaml"
  "*sha*/olm/operator-tempo-product-openshift-tempo-operator.yaml"
  "*sha*/olm/subscription-tempo-operator-*-sub.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/tempomonolithic-mono-route.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/service-tempo-mono-route-jaegerui.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/configmap-tempo-mono-route-serving-cabundle.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/statefulset-tempo-mono-route.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/service-tempo-mono-route.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/route-tempo-mono-route-jaegerui.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/configmap-tempo-mono-route-config.yaml"
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/serviceaccount-tempo-mono-route.yaml"
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