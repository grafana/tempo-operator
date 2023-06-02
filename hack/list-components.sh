#!/bin/bash

echo "### Components"
cat config/overlays/community/controller_manager_config.yaml | grep -E "docker.io|quay.io" | sed 's/^ /-/'
