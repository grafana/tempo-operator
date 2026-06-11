#!/bin/bash

set -e

# Create Azure storage bucket
AZURE_RESOURCE_GROUP_NAME=ikanse-monolithic-azure-wif
AZURE_RESOURCE_GROUP_LOCATION=eastus
AZURE_STORAGE_AZURE_ACCOUNTNAME="ikansemonowif"
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

if az identity show --name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" &>/dev/null; then
    echo "Managed identity '$IDENTITY_NAME' already exists, reusing it."
    az identity show --name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" | jq
else
    echo "Creating managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
    az identity create \
      --name "$IDENTITY_NAME" \
      --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
      --location "$OIDC_REGION" \
      --subscription "$SUBSCRIPTION_ID" | jq
fi

FEDERATED_CRED_NAME="chainsaw-azurewif-mono"
echo "Creating federated credentials for subject '$TEMPO_SA_SUBJECT' in managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
if az identity federated-credential show --name "$FEDERATED_CRED_NAME" --identity-name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" &>/dev/null; then
    echo "Federated credential '$FEDERATED_CRED_NAME' already exists, deleting and recreating..."
    az identity federated-credential delete --name "$FEDERATED_CRED_NAME" --identity-name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" --yes
fi
az identity federated-credential create \
  --name "$FEDERATED_CRED_NAME" \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --issuer "$CLUSTER_ISSUER" \
  --subject "$TEMPO_SA_SUBJECT" \
  --audiences "$AUDIENCE" | jq

FEDERATED_CRED_QF_NAME="chainsaw-azurewif-mono-query-frontend"
echo "Creating federated credentials for subject '$TEMPO_SA_QUERY_FRONTEND_SUBJECT' in managed identity '$IDENTITY_NAME' in resource group '$OCP_OIDC_RESOURCE_GROUP_NAME'..."
if az identity federated-credential show --name "$FEDERATED_CRED_QF_NAME" --identity-name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" &>/dev/null; then
    echo "Federated credential '$FEDERATED_CRED_QF_NAME' already exists, deleting and recreating..."
    az identity federated-credential delete --name "$FEDERATED_CRED_QF_NAME" --identity-name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" --yes
fi
az identity federated-credential create \
  --name "$FEDERATED_CRED_QF_NAME" \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --issuer "$CLUSTER_ISSUER" \
  --subject "$TEMPO_SA_QUERY_FRONTEND_SUBJECT" \
  --audiences "$AUDIENCE" | jq

echo "Wait for Azure resources"
sleep 20

ASSIGNEE_NAME=$(az ad sp list --all --filter "servicePrincipalType eq 'ManagedIdentity'" | jq -r --arg idName "$IDENTITY_NAME" '.[] | select(.displayName == $idName) | .appId')
echo "Assignee name is $ASSIGNEE_NAME"

EXISTING_ROLE=$(az role assignment list \
  --assignee "$ASSIGNEE_NAME" \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --query "[0].id" -o tsv 2>/dev/null || true)

if [ -z "$EXISTING_ROLE" ]; then
    echo "Assigning role Storage Blob Data Contributor to managed identity's '$IDENTITY_NAME' service principal '$ASSIGNEE_NAME'"
    az role assignment create \
      --assignee "$ASSIGNEE_NAME" \
      --role "Storage Blob Data Contributor" \
      --scope "/subscriptions/$SUBSCRIPTION_ID" | jq
else
    echo "Role 'Storage Blob Data Contributor' already assigned to '$ASSIGNEE_NAME', skipping."
fi

# Fetch the Client ID of the existing managed identity
CLIENT_ID=$(az identity show \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --query clientId \
  -o tsv)

# Create Kubernetes secret for Azure WIF (delete first if it exists to ensure fresh values)
kubectl delete secret azure-secret -n "$TEMPO_NAMESPACE" --ignore-not-found=true
kubectl create -n "$TEMPO_NAMESPACE" secret generic azure-secret \
  --from-literal=container="$AZURE_STORAGE_AZURE_CONTAINER" \
  --from-literal=account_name="$AZURE_STORAGE_AZURE_ACCOUNTNAME" \
  --from-literal=client_id="$CLIENT_ID" \
  --from-literal=audience="$AUDIENCE" \
  --from-literal=tenant_id="$TENANT_ID" || { echo "Failed to create secret"; exit 1; }

echo "Script executed successfully"
