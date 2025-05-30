#!/bin/bash

set -e

# Create Azure storage bucket
AZURE_RESOURCE_GROUP_NAME=ikanse-tempostack-azure-wif
AZURE_RESOURCE_GROUP_LOCATION=eastus
AZURE_STORAGE_AZURE_ACCOUNTNAME="ikansetempowif"
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

# Create Azure Managed Identity
echo "Creating Azure Managed Identiy"

TEMPO_NAMESPACE="chainsaw-azurewif-tempo"
TEMPO_NAME="azurewiftm"
OCP_CLUSTER_NAME=$(oc get infrastructure cluster -o json | jq -r .status.infrastructureName)
OCP_RESOURCE_GROUP_NAME=$(oc get infrastructure cluster -o json | jq -r .status.platformStatus.azure.resourceGroupName)
OCP_OIDC_RESOURCE_GROUP_NAME="$OCP_RESOURCE_GROUP_NAME-oidc"
SUBSCRIPTION_ID="$(az account show | jq -r '.id')"
TENANT_ID="$(az account show | jq -r '.tenantId')"
IDENTITY_NAME="$OCP_CLUSTER_NAME-$TEMPO_NAME-azure-cloud-credentials"
TEMPO_SA_SUBJECT="system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}"
TEMPO_SA_QUERY_FRONTEND_SUBJECT="system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend"
CLUSTER_ISSUER=$(oc get authentication cluster -o json | jq -r .spec.serviceAccountIssuer)
OIDC_REGION=$(az group show --name "$OCP_OIDC_RESOURCE_GROUP_NAME" --query location -o tsv)
AUDIENCE="api://AzureADTokenExchange"

echo "Creating managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
az identity create \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --location "$OIDC_REGION" \
  --subscription "$SUBSCRIPTION_ID" | jq

echo "Creating federated credentials for subject '$TEMPO_SA_SUBJECT' in managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
az identity federated-credential create \
  --name chainsaw-azurewif-tempo \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --issuer "$CLUSTER_ISSUER" \
  --subject "$TEMPO_SA_SUBJECT" \
  --audiences "$AUDIENCE" | jq

echo "Creating federated credentials for subject '$TEMPO_SA_QUERY_FRONTEND_SUBJECT' in managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
az identity federated-credential create \
  --name chainsaw-azurewif-tempo-query-frontend \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --issuer "$CLUSTER_ISSUER" \
  --subject "$TEMPO_SA_QUERY_FRONTEND_SUBJECT" \
  --audiences "$AUDIENCE" | jq

echo "Wait for Azure resources"
sleep 20

ASSIGNEE_NAME=$(az ad sp list --all --filter "servicePrincipalType eq 'ManagedIdentity'" | jq -r --arg idName "$IDENTITY_NAME" '.[] | select(.displayName == $idName) | .appId')
echo "Assignee name is $ASSIGNEE_NAME"

echo "Assigning role Storage Blob Data Contributor to managed identity's '$IDENTITY_NAME' service principal '$ASSIGNEE_NAME'"
az role assignment create \
  --assignee "$ASSIGNEE_NAME" \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID" | jq

# Fetch the Client ID of the existing managed identity
CLIENT_ID=$(az identity show \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --query clientId \
  -o tsv)

# Create Kubernetes secret for Azure WIF
kubectl create -n $TEMPO_NAMESPACE secret generic azure-secret \
  --from-literal=container="$AZURE_STORAGE_AZURE_CONTAINER" \
  --from-literal=account_name="$AZURE_STORAGE_AZURE_ACCOUNTNAME" \
  --from-literal=client_id="$CLIENT_ID" \
  --from-literal=audience="$AUDIENCE" \
  --from-literal=tenant_id="$TENANT_ID" || { echo "Failed to create secret"; exit 1; }

  echo "Script executed successfully"
