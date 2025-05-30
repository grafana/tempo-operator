#!/bin/bash

set -e

AZURE_RESOURCE_GROUP_NAME=ikanse-tempostack-azure

# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  az group delete --name $AZURE_RESOURCE_GROUP_NAME -y || { echo "Failed to delete resource group"; }
fi

#Wait for the resource group to be deleted for 30 seconds.
# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  sleep 30
fi

echo "Script executed successfully"
