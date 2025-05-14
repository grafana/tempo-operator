#!/bin/bash

set -e

AZURE_RESOURCE_GROUP_NAME=ikanse-monolithic-azure
AZURE_RESOURCE_GROUP_LOCATION=eastus
AZURE_STORAGE_AZURE_ACCOUNTNAME="ikansemono"
SECRETNAME="azure-secret"
AZURE_ENV="AzureGlobal"

# Check if the storage account exists before attempting to delete it.
if az storage account show --name "$AZURE_STORAGE_AZURE_ACCOUNTNAME" --resource-group "$AZURE_RESOURCE_GROUP_NAME" &>/dev/null; then
    # Delete the storage account
    az storage account delete --name "$AZURE_STORAGE_AZURE_ACCOUNTNAME" --resource-group "$AZURE_RESOURCE_GROUP_NAME" --yes
    echo "Storage account '$AZURE_STORAGE_AZURE_ACCOUNTNAME' deleted successfully."
else
    echo "Storage account '$AZURE_STORAGE_AZURE_ACCOUNTNAME' does not exist."
fi

# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  az group delete --name $AZURE_RESOURCE_GROUP_NAME -y || { echo "Failed to delete resource group"; }
fi

#Wait for the resource group to be deleted for 30 seconds.
# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  sleep 30
fi

az group create --name $AZURE_RESOURCE_GROUP_NAME --location $AZURE_RESOURCE_GROUP_LOCATION || { echo "Failed to create resource group"; exit 1; }

az storage account create \
  --name $AZURE_STORAGE_AZURE_ACCOUNTNAME \
  --resource-group $AZURE_RESOURCE_GROUP_NAME \
  --location $AZURE_RESOURCE_GROUP_LOCATION \
  --sku Standard_RAGRS \
  --kind StorageV2 || { echo "Failed to create storage account"; exit 1; }

AZURE_STORAGE_AZURE_CONTAINER="ikansecntr"
AZURE_STORAGE_ACCOUNT_KEY=$(az storage account keys list --account-name $AZURE_STORAGE_AZURE_ACCOUNTNAME --resource-group $AZURE_RESOURCE_GROUP_NAME --query "[0].value") || { echo "Failed to list storage account keys"; exit 1; }

az storage container create --account-name $AZURE_STORAGE_AZURE_ACCOUNTNAME --resource-group $AZURE_RESOURCE_GROUP_NAME --fail-on-exist --name $AZURE_STORAGE_AZURE_CONTAINER --auth-mode login || { echo "Failed to create storage container"; exit 1; }

AZURE_STORAGE_ACCOUNT_KEY=$(az storage account keys list --account-name $AZURE_STORAGE_AZURE_ACCOUNTNAME --resource-group $AZURE_RESOURCE_GROUP_NAME --query "[0].value" | tr -d '"') || { echo "Failed to list storage account keys"; exit 1; }

kubectl create -n $NAMESPACE secret generic azure-secret \
  --from-literal=container="$AZURE_STORAGE_AZURE_CONTAINER" \
  --from-literal=account_name="$AZURE_STORAGE_AZURE_ACCOUNTNAME" \
  --from-literal=account_key=$AZURE_STORAGE_ACCOUNT_KEY || { echo "Failed to create secret"; exit 1; }

echo "Script executed successfully"
