# TempoMonolithic with Azure Blob Storage

This test validates TempoMonolithic deployment using Azure Blob Storage as the backend for trace data persistence. It demonstrates how to configure and deploy a single-component Tempo instance with Azure cloud storage integration.

## Test Overview

### Purpose
- **Azure Blob Storage Integration**: Tests Azure cloud storage backend with TempoMonolithic
- **Cloud Storage Configuration**: Validates secure access to Azure storage accounts
- **Monolithic Architecture**: Verifies single-component deployment with cloud storage
- **End-to-End Workflow**: Tests complete trace lifecycle from ingestion to querying

### Components
- **TempoMonolithic**: Single-component Tempo deployment with Azure storage
- **Azure Blob Storage**: Microsoft cloud object storage for trace data persistence
- **Azure Storage Account**: Container for blob storage with access credentials
- **Storage Container**: Logical grouping for trace data organization

## Deployment Steps

### 1. Create Azure Storage Infrastructure
```bash
./create-bucket.sh
```

The [`create-bucket.sh`](create-bucket.sh) script performs:
- **Storage Account Creation**: Creates Azure Storage Account for blob storage
- **Container Setup**: Creates storage container for trace data
- **Access Key Generation**: Generates storage access keys for authentication
- **Secret Creation**: Creates Kubernetes secret with Azure storage configuration

### 2. Deploy TempoMonolithic with Azure Storage
```bash
kubectl apply -f 01-install-tempo.yaml
```

Key configuration from [`01-install-tempo.yaml`](01-install-tempo.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  storage:
    traces:
      backend: azure
      azure:
        secret: azure-secret  # Contains Azure storage configuration
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f 03-generate-traces.yaml
kubectl apply -f 04-verify-traces.yaml
```

## Key Features Tested

### Azure Blob Storage Integration
- ✅ Azure Storage Account access and authentication
- ✅ Blob container creation and management
- ✅ Secure credential handling with Kubernetes secrets
- ✅ Cross-region storage access and reliability

### TempoMonolithic Configuration
- ✅ Single-component deployment with Azure backend
- ✅ Simplified resource management and operation
- ✅ Direct Azure Blob Storage integration
- ✅ Automatic trace data organization in containers

### Storage Operations
- ✅ Trace data persistence to Azure Blob Storage
- ✅ Efficient blob storage and retrieval
- ✅ Storage container lifecycle management
- ✅ Data durability and availability

### End-to-End Validation
- ✅ Trace ingestion through OTLP endpoint
- ✅ Successful storage to Azure Blob Storage
- ✅ Trace retrieval and querying capabilities
- ✅ Complete observability pipeline functionality

## Azure Storage Secret Configuration

The `azure-secret` contains Azure-specific configuration:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-secret
data:
  # Azure storage account credentials and configuration
  account_name: <base64-encoded-account-name>
  account_key: <base64-encoded-account-key>
  container_name: <base64-encoded-container-name>
```

## Environment Requirements

### Azure Prerequisites
- Valid Azure subscription with storage permissions
- Azure Storage Account creation capabilities
- Blob Storage service enabled
- Network connectivity from OpenShift cluster to Azure

### OpenShift Prerequisites
- Sufficient cluster resources for TempoMonolithic deployment
- Ability to create secrets and deploy custom resources
- Network policies allowing egress to Azure endpoints

## Cleanup Process

The test automatically cleans up Azure resources:
```bash
./delete-bucket.sh
```

This removes:
- Azure Storage Account and associated resources
- Storage containers and all blob data
- Kubernetes secrets and configurations

## Comparison with Other Cloud Providers

### Azure Advantages
- **Global Availability**: Extensive global datacenter presence
- **Enterprise Integration**: Strong integration with Microsoft ecosystem
- **Security Features**: Advanced security and compliance capabilities
- **Cost Management**: Flexible pricing tiers and lifecycle policies

### Storage Features
- **Hot/Cool/Archive Tiers**: Cost-optimized storage based on access patterns
- **Geo-redundancy**: Built-in data replication across regions
- **Access Control**: Fine-grained permissions and access policies
- **Monitoring**: Comprehensive metrics and logging capabilities

## Troubleshooting

### Common Issues

**Azure Authentication Failures**:
- Verify storage account credentials are correct
- Check that storage account allows access from cluster network
- Ensure storage account is in the correct Azure region

**Blob Storage Access Issues**:
- Confirm storage container exists and is accessible
- Verify network connectivity to Azure storage endpoints
- Check firewall rules and network security groups

**TempoMonolithic Startup Issues**:
- Validate Azure secret contains all required fields
- Check TempoMonolithic pod logs for authentication errors
- Ensure sufficient cluster resources for deployment

**Trace Storage Problems**:
- Verify blob storage permissions are correctly configured
- Check storage account capacity and quotas
- Monitor Azure storage metrics for performance issues

## Architecture Benefits

### Monolithic Deployment
- **Simplicity**: Single component reduces operational complexity
- **Resource Efficiency**: Lower overhead compared to distributed deployments
- **Quick Setup**: Faster deployment for development and testing
- **Cost Effective**: Reduced infrastructure costs for smaller workloads

### Azure Integration
- **Scalability**: Leverage Azure's storage scalability
- **Durability**: Built-in data protection and backup capabilities
- **Performance**: Optimized storage performance for trace workloads
- **Compliance**: Azure compliance certifications for regulated workloads

This test demonstrates how TempoMonolithic can effectively integrate with Microsoft Azure cloud storage services, providing a reliable and scalable solution for trace data persistence in cloud-native environments.
