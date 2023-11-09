#!/bin/bash

TEMPO_VERSION=$(cat config/manager/manager.yaml | grep -oP "docker.io/grafana/tempo:\K.*")

cat << EOF
### Components
- Tempo: [v${TEMPO_VERSION}](https://github.com/grafana/tempo/releases/tag/v${TEMPO_VERSION})
EOF
