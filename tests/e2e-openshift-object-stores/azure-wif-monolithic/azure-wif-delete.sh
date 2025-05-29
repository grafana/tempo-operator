#!/bin/bash

set -uo pipefail

AZURE_RESOURCE_GROUP_NAME=ikanse-monolithic-azure-wif

# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  az group delete --name $AZURE_RESOURCE_GROUP_NAME -y || { echo "Failed to delete resource group"; }
fi

#Wait for the resource group to be deleted for 30 seconds.
# Check if the resource group exists before attempting to delete it.
if [ "$(az group exists --name $AZURE_RESOURCE_GROUP_NAME)" == "true" ]; then
  sleep 30
fi

TEMPO_NAMESPACE="chainsaw-azurewif-mono"
TEMPO_NAME="azurewifmn"
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

# Delete the first federated credential
echo "Deleting federated credential 'chainsaw-azurewif-tempo'..."
az identity federated-credential delete \
  --name chainsaw-azurewif-mono \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --yes  || {
    echo "Warning: Failed to delete federated credential 'chainsaw-azurewif-tempo'. It might not exist."
}

# Delete the second federated credential
echo "Deleting federated credential 'chainsaw-azurewif-tempo-query-frontend'..."
az identity federated-credential delete \
  --name chainsaw-azurewif-monoo-query-frontend \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --yes || {
    echo "Warning: Failed to delete federated credential 'chainsaw-azurewif-tempo-query-frontend'. It might not exist."
}

# Delete role assignment
CLIENT_ID=$(az identity show \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --query clientId \
  -o tsv)

ROLE_ASSIGNMENT_ID=$(az role assignment list \
  --assignee "$CLIENT_ID" \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --query "[0].id" \
  -o tsv)

echo "Deleting role assignment '$ROLE_ASSIGNMENT_ID' for managed identity '$IDENTITY_NAME'..."
az role assignment delete \
  --ids "$ROLE_ASSIGNMENT_ID" || {
    echo "Warning: Failed to delete role assignment. It might have already been removed or permissions issues." 
  }

# Cleanup Azure Managed Identity
echo "Deleting Azure Managed Identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
az identity delete \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" || {
    echo "Warning: Failed to delete managed identity. It might not exist or still has associated resources (e.g., role assignments - though we tried to delete it)."
}

# Delete Kubernetes secret
oc delete secrets azure-secret -n $TEMPO_NAMESPACE

echo "Script executed successfully"
