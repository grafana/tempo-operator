# TempoStack with Azure Blob Storage

This test validates TempoStack deployment using Azure Blob Storage as the backend for trace data persistence. It demonstrates how to configure and deploy a distributed Tempo instance with Azure cloud storage integration for production-scale workloads.

## Test Overview

### Purpose
- **Azure Blob Storage Integration**: Tests Azure cloud storage backend with distributed TempoStack
- **Distributed Architecture**: Validates multi-component deployment with cloud storage
- **Production Scalability**: Demonstrates enterprise-ready deployment model
- **End-to-End Workflow**: Tests complete trace lifecycle from ingestion to querying

### Components
- **TempoStack**: Distributed Tempo deployment with Azure storage
- **Azure Blob Storage**: Microsoft cloud object storage for trace data persistence
- **Azure Storage Account**: Container for blob storage with access credentials
- **Storage Container**: Logical grouping for trace data organization
- **Multiple Service Accounts**: Component-specific authentication for distributed architecture

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

### 2. Deploy TempoStack with Azure Storage
```bash
kubectl apply -f 01-install-tempostack.yaml
```

Key configuration from [`01-install-tempostack.yaml`](01-install-tempostack.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: azure-secret
      type: azure
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: ingress
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f 02-generate-traces.yaml
kubectl apply -f 03-verify-traces.yaml
```

## Key Features Tested

### Azure Blob Storage Integration
- ✅ Azure Storage Account access and authentication
- ✅ Blob container creation and management
- ✅ Secure credential handling with Kubernetes secrets
- ✅ Cross-region storage access and reliability

### TempoStack Configuration
- ✅ Distributed deployment with Azure backend
- ✅ Jaeger Query UI with Kubernetes Ingress
- ✅ 200M storage allocation for trace data
- ✅ Resource limits: 2Gi memory, 2000m CPU
- ✅ Multiple service accounts for different components

### Distributed Architecture Benefits
- ✅ Independent component scaling and management
- ✅ Fault tolerance through component separation
- ✅ Better resource utilization for high-volume workloads
- ✅ Production-ready deployment model

### End-to-End Validation
- ✅ Trace ingestion through distributed OTLP endpoints
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

## TempoStack Component Architecture

The distributed deployment creates multiple components:
- **Distributor**: Handles trace ingestion and load balancing
- **Ingester**: Processes and stores traces to Azure Blob Storage
- **Querier**: Retrieves traces for query processing
- **Query Frontend**: Handles query routing and caching
- **Compactor**: Manages trace data compaction and retention

```
[Trace Ingestion] → [Distributor] → [Ingester] → [Azure Blob Storage]
                                       ↑              ↓
[Query UI] ← [Query Frontend] ← [Querier] ← [Compactor]
```

## Environment Requirements

### Azure Prerequisites
- Valid Azure subscription with storage permissions
- Azure Storage Account creation capabilities
- Blob Storage service enabled
- Network connectivity from OpenShift cluster to Azure

### OpenShift Prerequisites
- Sufficient cluster resources for distributed TempoStack deployment
- Ability to create secrets and deploy custom resources
- Network policies allowing egress to Azure endpoints

## Comparison with TempoMonolithic

### TempoStack Advantages
- **Scalability**: Better for high-volume trace workloads
- **Component Separation**: Independent scaling of ingestion and query
- **Fault Tolerance**: Component isolation reduces single points of failure
- **Production Ready**: Better suited for enterprise deployments

### TempoMonolithic Advantages
- **Simpler Setup**: Single component vs. multiple service accounts
- **Resource Efficiency**: Lower memory and CPU overhead
- **Unified Component**: All functionality in single pod
- **Development Friendly**: Easier to deploy and manage for testing

## Cleanup Process

The test automatically cleans up Azure resources:
```bash
./delete-bucket.sh
```

This removes:
- Azure Storage Account and associated resources
- Storage containers and all blob data
- Kubernetes secrets and configurations

## Azure Integration Benefits

### Enterprise Features
- **Global Availability**: Extensive global datacenter presence
- **Enterprise Integration**: Strong integration with Microsoft ecosystem
- **Security Features**: Advanced security and compliance capabilities
- **Cost Management**: Flexible pricing tiers and lifecycle policies

### Storage Capabilities
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

**TempoStack Component Issues**:
- Validate Azure secret contains all required fields
- Check individual component pod logs for authentication errors
- Ensure sufficient cluster resources for all components

**Distributed Deployment Problems**:
- Verify all TempoStack components can access Azure storage
- Check component inter-communication and service discovery
- Monitor resource usage across all components

## Performance Considerations

### Azure Storage Optimization
- **Access Patterns**: Configure appropriate storage tiers based on trace access patterns
- **Regional Placement**: Deploy storage in same region as cluster for lower latency
- **Bandwidth**: Ensure sufficient network bandwidth for high-volume trace workloads
- **Monitoring**: Use Azure Monitor to track storage performance metrics

### TempoStack Scaling
- **Component Resources**: Allocate appropriate CPU and memory for each component
- **Ingestion Rate**: Scale distributors and ingesters based on trace volume
- **Query Performance**: Scale queriers based on query load patterns
- **Storage Growth**: Monitor and plan for trace data growth over time

This test demonstrates how TempoStack can effectively integrate with Microsoft Azure cloud storage services, providing a reliable, scalable, and enterprise-ready solution for trace data persistence in distributed observability environments.
