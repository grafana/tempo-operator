#!/bin/bash
set -e

if [[ -z $OPERATOR_VERSION ]]; then
    echo "OPERATOR_VERSION isn't set. Skipping process."
    exit 1
fi

echo "Updating the Tempo images version"
sed -i "s~docker.io/grafana/tempo:.*~docker.io/grafana/tempo:${TEMPO_VERSION}~gi" config/manager/controller_manager_config.yaml
sed -i "s~docker.io/grafana/tempo-query:.*~docker.io/grafana/tempo-query:${TEMPO_VERSION}~gi" config/manager/controller_manager_config.yaml

echo "Generating the bundle"
USER=os-observability VERSION=${OPERATOR_VERSION} USER=tempo-operator make bundle
