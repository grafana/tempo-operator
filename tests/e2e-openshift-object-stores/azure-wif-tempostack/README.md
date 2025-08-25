# TempoStack with Azure Workload Identity Federation (WIF)

This configuration blueprint demonstrates how to deploy TempoStack with Azure Workload Identity Federation for secure access to Azure Blob Storage. This setup showcases enterprise-grade cloud security with Azure Managed Identity, OIDC federation, and token-based authentication without storing long-term credentials.

## Overview

This test validates a cloud-native security stack featuring:
- **Azure Workload Identity Federation**: OpenShift service accounts federated with Azure Managed Identity
- **Temporary Token Authentication**: Short-lived Azure AD tokens for Blob Storage access
- **Managed Identity Integration**: Azure-native identity management with federated trust
- **Zero Credential Storage**: No Azure credentials stored in Kubernetes secrets
- **Enterprise Security**: Role-based access control with automated credential management

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│   TempoStack    │───▶│   Azure AD / Entra   │───▶│ Azure Blob      │
│ ┌─────────────┐ │    │ ┌─────────────────┐  │    │ Storage         │
│ │ Service     │ │    │ │ Token Exchange  │  │    │ ┌─────────────┐ │
│ │ Accounts    │ │    │ │ OIDC → AAD      │  │    │ │ Container   │ │
│ │ + JWT       │ │    │ │ Managed Idenity │  │    │ │ Access      │ │
│ └─────────────┘ │    │ └─────────────────┘  │    │ └─────────────┘ │
└─────────────────┘    └──────────────────────┘    └─────────────────┘
          │                        ▲
          │ ┌──────────────────────┴─────────────────────┐
          │ │         OpenShift OIDC Provider            │
          └▶│ ┌─────────────────┐ ┌─────────────────┐    │
            │ │ Service Account │ │ OIDC Issuer     │    │
            │ │ JWT Tokens      │ │ Federation      │    │
            │ └─────────────────┘ └─────────────────┘    │
            └────────────────────────────────────────────┘
                                   ▲
                          ┌──────────────────────┐
                          │ Azure Managed Identity│
                          │ - Federated Creds     │
                          │ - Storage Blob Role   │
                          │ - Trust Relationship  │
                          └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+) on Azure with OIDC federation capabilities
- Azure subscription with permissions for resource group and identity management
- Tempo Operator installed
- Azure CLI configured
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Create Azure Infrastructure and Identity Resources

Run the automated setup script to create Azure resources and configure Workload Identity Federation:

```bash
./azure-wif-create.sh
```

**Script Functionality from [`azure-wif-create.sh`](./azure-wif-create.sh)**:

#### Azure Resource Group and Storage Account
```bash
AZURE_RESOURCE_GROUP_NAME=example-user-tempostack-azure-wif
AZURE_STORAGE_AZURE_ACCOUNTNAME="example-usertempowif"
AZURE_STORAGE_AZURE_CONTAINER="example-usercntr"
```

- Creates dedicated resource group for TempoStack resources
- Provisions Standard_RAGRS storage account for geo-redundancy
- Creates blob container for trace storage

#### Managed Identity Creation
```bash
IDENTITY_NAME="$OCP_CLUSTER_NAME-$TEMPO_NAME-azure-cloud-credentials"
az identity create --name "$IDENTITY_NAME" --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME"
```

#### Federated Credential Configuration
Creates federated credentials for OpenShift service accounts:
```bash
# Primary service account
TEMPO_SA_SUBJECT="system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}"

# Query frontend service account  
TEMPO_SA_QUERY_FRONTEND_SUBJECT="system:serviceaccount:${TEMPO_NAMESPACE}:tempo-${TEMPO_NAME}-query-frontend"
```

#### RBAC Assignment
```bash
az role assignment create \
  --assignee "$ASSIGNEE_NAME" \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID"
```

#### Secret Creation
Creates Kubernetes secret with WIF configuration:
- `container`: Azure Blob Storage container name
- `account_name`: Azure Storage account name
- `client_id`: Managed Identity client ID
- `audience`: Token exchange audience (api://AzureADTokenExchange)
- `tenant_id`: Azure AD tenant ID

### Step 2: Deploy TempoStack with Azure WIF

Apply TempoStack configuration for Azure Blob Storage with WIF:

```bash
oc apply -f install-tempostack.yaml
```

**Key WIF Configuration from [`install-tempostack.yaml`](./install-tempostack.yaml)**:
- **Azure Storage Type**: Configured for Azure Blob Storage with WIF authentication
- **Storage Secret**: References `azure-secret` containing managed identity configuration
- **OpenShift Route**: External access for Jaeger UI
- **Resource Allocation**: 4Gi memory and 2000m CPU for production workloads

### Step 3: Verify TempoStack Readiness

Wait for TempoStack to initialize with Azure WIF authentication:

```bash
oc get --namespace chainsaw-azurewif-tempo tempo azurewiftm -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

### Step 4: Generate Sample Traces

Create traces to validate WIF-authenticated Blob Storage:

```bash
oc apply -f generate-traces.yaml
```

**Reference**: [`generate-traces.yaml`](./generate-traces.yaml)

### Step 5: Verify Traces with Azure WIF Storage

Validate that traces are properly stored in Azure Blob Storage via WIF:

```bash
oc apply -f verify-traces.yaml
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

### Step 6: Cleanup Azure Resources

After testing, clean up Azure infrastructure:

```bash
./azure-wif-delete.sh
```

**Reference**: [`azure-wif-delete.sh`](./azure-wif-delete.sh)

## Key Features Demonstrated

### 1. **Azure Workload Identity Federation**
- **Token Exchange**: OpenShift JWT tokens exchanged for Azure AD access tokens
- **Federated Trust**: OpenShift OIDC provider trusted by Azure AD
- **Managed Identity**: Azure-native identity without password management
- **Audience Validation**: Secure token exchange with audience restrictions

### 2. **Enterprise Cloud Security**
- **Zero Secret Storage**: No Azure credentials stored in Kubernetes
- **Temporary Tokens**: Short-lived access tokens with automatic refresh
- **Role-Based Access**: Azure RBAC for fine-grained storage permissions
- **Audit Integration**: Complete Azure Activity Log tracking

### 3. **OpenShift Integration**
- **Service Account Mapping**: Direct mapping of K8s service accounts to Azure identities
- **OIDC Federation**: Seamless integration with OpenShift's identity provider
- **Namespace Isolation**: Scope-specific identity and access controls
- **Automatic Token Refresh**: Background token renewal without service interruption

### 4. **Production Azure Patterns**
- **Geo-Redundant Storage**: RAGRS storage for high availability
- **Container Organization**: Structured blob container layout
- **Regional Deployment**: Co-located compute and storage for performance
- **Cost Optimization**: Efficient storage tiers and access patterns

## Azure Workload Identity Flow

### Token Exchange Process

1. **Service Account Token**: TempoStack pod receives OpenShift service account JWT
2. **Federated Credential Lookup**: Azure AD validates JWT against federated credentials
3. **Token Exchange**: JWT exchanged for Azure AD access token via OIDC
4. **Managed Identity Access**: Access token represents Azure Managed Identity
5. **Blob Storage Access**: Managed Identity accesses storage with assigned RBAC roles
6. **Automatic Refresh**: Tokens automatically renewed before expiration

### Security Benefits

- **Credential-free Architecture**: No stored credentials can be compromised
- **Identity Federation**: Leverages existing OpenShift identity infrastructure
- **Conditional Access**: Support for Azure Conditional Access policies
- **Comprehensive Auditing**: Azure Activity Log tracks all access

## Azure Configuration Details

### Managed Identity Components

```bash
# Managed Identity Creation
az identity create \
  --name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --location "$OIDC_REGION"

# Federated Credential Configuration
az identity federated-credential create \
  --name chainsaw-azurewif-tempo \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME" \
  --issuer "$CLUSTER_ISSUER" \
  --subject "system:serviceaccount:tempo-namespace:tempo-service-account" \
  --audiences "api://AzureADTokenExchange"
```

### RBAC Configuration

```bash
# Storage Blob Data Contributor role assignment
az role assignment create \
  --assignee "$MANAGED_IDENTITY_CLIENT_ID" \
  --role "Storage Blob Data Contributor" \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP"
```

### Required Azure Permissions

- **Resource Group**: Contributor access for resource creation
- **Managed Identity**: User Access Administrator for identity management
- **Storage Account**: Storage Account Contributor for account operations
- **RBAC**: Role assignment permissions for identity-to-storage mapping

## Monitoring and Troubleshooting

### Verify Azure WIF Configuration

```bash
# Check Azure secret configuration
oc get secret azure-secret -o yaml

# Verify managed identity client ID
oc get secret azure-secret -o jsonpath='{.data.client_id}' | base64 -d

# Check federated credentials
az identity federated-credential list \
  --identity-name "$IDENTITY_NAME" \
  --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME"
```

### Validate Azure Resources

```bash
# List managed identities
az identity list --resource-group "$OCP_OIDC_RESOURCE_GROUP_NAME"

# Check storage account
az storage account show \
  --name "$AZURE_STORAGE_AZURE_ACCOUNTNAME" \
  --resource-group "$AZURE_RESOURCE_GROUP_NAME"

# Verify role assignments
az role assignment list --assignee "$MANAGED_IDENTITY_CLIENT_ID"
```

### Debug WIF Authentication

```bash
# Check component logs for Azure authentication
oc logs -l app.kubernetes.io/component=compactor | grep -i azure

# Monitor token exchange operations
oc logs -l app.kubernetes.io/component=ingester | grep -i "token\|azure"

# Verify blob storage access
oc logs -l app.kubernetes.io/component=querier | grep -i "storage\|blob"
```

### Common WIF Issues

1. **Federated Credential Mismatch**:
   ```bash
   # Verify OIDC issuer URL
   oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}'
   
   # Check service account subject format
   kubectl get serviceaccount tempo-azurewiftm -o yaml
   ```

2. **Azure RBAC Permission Errors**:
   ```bash
   # Check role assignments
   az role assignment list --assignee "$CLIENT_ID" --output table
   
   # Verify storage account permissions
   az storage account show --name "$STORAGE_ACCOUNT" --query "primaryEndpoints"
   ```

3. **Token Exchange Failures**:
   ```bash
   # Monitor Azure AD sign-in logs
   az monitor activity-log list --correlation-id "$CORRELATION_ID"
   
   # Check managed identity access
   az rest --method GET --uri "https://management.azure.com/subscriptions/$SUB_ID/providers/Microsoft.ManagedIdentity/userAssignedIdentities"
   ```

## Production Considerations

### 1. **Security Hardening**
- Implement least-privilege Azure RBAC roles
- Use Azure Key Vault for additional secret management
- Enable Azure Security Center monitoring
- Regular review of federated credential configurations

### 2. **High Availability**
- Deploy across multiple Azure availability zones
- Configure Azure Blob Storage geo-replication
- Monitor Azure service health and SLA compliance
- Implement circuit breakers for Azure API failures

### 3. **Cost Optimization**
- Use Azure Blob Storage lifecycle management
- Monitor data transfer costs between regions
- Optimize storage account tiers based on access patterns
- Consider Azure Reserved Capacity for predictable workloads

### 4. **Operational Excellence**
- Automate Azure resource lifecycle with ARM templates
- Monitor WIF token usage and renewal patterns
- Implement automated testing of WIF authentication flow
- Document disaster recovery procedures for Azure dependencies

## Related Configurations

- [AWS STS Integration](../aws-sts-tempostack/README.md) - AWS equivalent of Azure WIF
- [GCP Workload Identity](../gcp-wif-tempostack/README.md) - GCP workload identity federation
- [Basic Azure Storage](../tempostack-azure/README.md) - Azure with traditional credentials
- [Basic TempoStack](../../e2e/compatibility/README.md) - Local storage baseline

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/azure-wif-tempostack
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires Azure subscription access and proper OIDC federation setup between OpenShift and Azure AD.