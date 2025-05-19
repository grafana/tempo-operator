#!/bin/bash

TEMPO_VERSION=$(grep -oP "docker.io/grafana/tempo:\K.*" config/manager/manager.yaml)
KUBE_MIN_VERSION=$(grep -oP 'kube-version: "\K([\d.]*)' .github/workflows/e2e.yaml | sed -n '1p')
KUBE_MAX_VERSION=$(grep -oP 'kube-version: "\K([\d.]*)' .github/workflows/e2e.yaml | sed -n '2p')

cat << EOF
### Components
- Tempo: [v${TEMPO_VERSION}](https://github.com/grafana/tempo/releases/tag/v${TEMPO_VERSION})

### Support
This release supports Kubernetes ${KUBE_MIN_VERSION} to ${KUBE_MAX_VERSION}.
EOF
