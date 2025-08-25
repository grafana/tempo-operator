# TempoMonolithic with Azure Workload Identity Federation

This configuration blueprint demonstrates TempoMonolithic deployment with Azure Workload Identity Federation (WIF) for secure, credential-free access to Azure Blob Storage. This setup provides enterprise-grade cloud security using Azure AD managed identities and OIDC federation, eliminating the need for stored credentials.

## Overview

This test validates Azure WIF integration featuring:
- **Azure Workload Identity Federation**: Secure authentication without stored credentials
- **Managed Identity Integration**: Azure AD managed identities for cloud resource access
- **OIDC Federation**: OpenShift service accounts federated with Azure AD
- **Blob Storage Backend**: Azure Blob Storage for trace data persistence
- **Automated Infrastructure**: Complete Azure resource lifecycle management

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ TempoMonolithic         │───▶│   Azure AD               │───▶│ Azure Blob Storage      │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │    │ ┌─────────────────────┐ │
│ │ Service Accounts    │ │    │ │ Managed Identity    │  │    │ │ Storage Account     │ │
│ │ - tempo-azurewifmn  │ │    │ │ - Federated Creds   │  │    │ │ - example-usermonowif     │ │
│ │ - tempo-...-qf      │ │    │ │ - RBAC Assignments  │  │    │ │ - Container: example-user │ │
│ └─────────────────────┘ │    │ └─────────────────────┘  │    │ └─────────────────────┘ │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘

┌─────────────────────────┐    ┌──────────────────────────┐
│ OpenShift OIDC          │───▶│ Azure Resource Group     │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │
│ │ Service Account     │ │    │ │ Resource Group      │  │
│ │ JWT Tokens          │ │    │ │ - example-user-mono-wif   │  │
│ │ - Federated         │ │    │ │ - East US           │  │
│ └─────────────────────┘ │    │ └─────────────────────┘  │
└─────────────────────────┘    └──────────────────────────┘

WIF Authentication Flow:
1. ServiceAccount JWT → Azure AD Token Exchange
2. Managed Identity → RBAC Authorization  
3. Blob Storage → Secure Data Access
```

## Prerequisites

- OpenShift cluster on Azure (ARO) or with Azure integration
- Azure subscription with sufficient permissions
- Azure CLI configured with appropriate credentials
- Tempo Operator installed
- Understanding of Azure AD and Workload Identity Federation

## Key Configuration Elements

### TempoMonolithic Configuration
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: azurewifmn
spec:
  storage:
    traces:
      backend: azure
      azure:
        secret: azure-secret
  jaegerui:
    enabled: true
    route:
      enabled: true
```

### Azure WIF Infrastructure
The setup script creates:
- **Resource Group**: `example-user-monolithic-azure-wif`
- **Storage Account**: `example-usermonowif` with Standard_RAGRS SKU
- **Container**: `example-usercntr` for trace storage
- **Managed Identity**: Federated with OpenShift service accounts
- **RBAC Assignments**: Storage Blob Data Contributor role

### Kubernetes Secret
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-secret
stringData:
  container: "example-usercntr"
  account_name: "example-usermonowif"
  client_id: "{managed-identity-client-id}"
  audience: "api://AzureADTokenExchange"
  tenant_id: "{azure-tenant-id}"
```

## Step-by-Step Deployment

### 1. Create Azure Infrastructure
```bash
./azure-wif-create.sh
```
Creates storage account, managed identity, and configures federation.

### 2. Deploy TempoMonolithic
```bash
oc apply -f install-monolithic.yaml
```

### 3. Generate and Verify Traces
```bash
oc apply -f generate-traces.yaml
oc apply -f verify-traces.yaml
```

### 4. Cleanup Resources
```bash
./azure-wif-delete.sh
```

## Security Features

### Workload Identity Federation
- **No Stored Credentials**: Uses managed identity for authentication
- **Automatic Token Exchange**: OpenShift JWT tokens exchanged for Azure AD tokens
- **RBAC Integration**: Azure role-based access control for storage resources
- **Audit Trail**: Complete Azure AD authentication and authorization logging

### Managed Identity Benefits
- **Credential Rotation**: Automatic credential lifecycle management
- **Least Privilege**: Granular RBAC permissions for storage access
- **Federation Security**: Secure trust relationship between OpenShift and Azure AD

## Production Considerations

### Security Best Practices
- Use dedicated resource groups for tenant isolation
- Implement least-privilege RBAC assignments
- Enable Azure Storage encryption and access logging
- Monitor federated credential usage and access patterns

### High Availability
- Configure geo-redundant storage (GRS/RA-GRS)
- Deploy across multiple Azure availability zones
- Implement cross-region disaster recovery strategies

### Cost Optimization
- Use appropriate storage tiers for trace data lifecycle
- Configure blob lifecycle policies for automated tier transitions
- Monitor storage costs and optimize based on access patterns

## Related Configurations

- [Azure WIF TempoStack](../azure-wif-tempostack/README.md) - Distributed Azure deployment
- [AWS STS Integration](../aws-sts-monolithic/README.md) - AWS equivalent
- [GCP WIF Integration](../gcp-wif-monolithic/README.md) - GCP equivalent

## Test Execution

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/azure-wif-monolithic
```

**Note**: This test demonstrates secure cloud-native authentication using Azure Workload Identity Federation, providing credential-free access to Azure Blob Storage for enterprise trace storage requirements.

**References**: 
- [`install-monolithic.yaml`](./install-monolithic.yaml)
- [`azure-wif-create.sh`](./azure-wif-create.sh)
- [`azure-wif-delete.sh`](./azure-wif-delete.sh)

