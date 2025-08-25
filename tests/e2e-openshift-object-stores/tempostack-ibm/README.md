# TempoStack with IBM Cloud Object Storage

This test validates TempoStack deployment using IBM Cloud Object Storage (COS) as the S3-compatible backend for trace data persistence. It demonstrates how to configure and deploy a distributed Tempo instance with IBM Cloud storage integration using HMAC authentication for production-scale workloads.

## Test Overview

### Purpose
- **IBM Cloud Object Storage Integration**: Tests IBM COS as S3-compatible backend with distributed TempoStack
- **HMAC Authentication**: Validates IBM Cloud storage access using HMAC keys
- **Distributed Architecture**: Verifies multi-component deployment with cloud storage
- **Enterprise Cloud Integration**: Demonstrates integration with IBM Cloud services for production workloads

### Components
- **TempoStack**: Distributed Tempo deployment with IBM COS storage
- **IBM Cloud Object Storage**: IBM's S3-compatible object storage service
- **IBM Cloud Resource Group**: Logical container for IBM Cloud resources
- **Service Instance**: IBM Cloud Object Storage service instance
- **HMAC Credentials**: Access keys for S3-compatible authentication
- **Multiple Service Accounts**: Component-specific authentication for distributed architecture

## Deployment Steps

### 1. Create IBM Cloud Storage Infrastructure
```bash
./create-bucket.sh
```

The [`create-bucket.sh`](create-bucket.sh) script performs:
- **IBM Cloud Authentication**: Uses API key from cluster secrets for authentication
- **Resource Group Creation**: Creates dedicated resource group for test resources
- **Service Instance Setup**: Creates IBM Cloud Object Storage service instance
- **Bucket Creation**: Creates COS bucket for trace data storage
- **HMAC Key Generation**: Creates service key with HMAC authentication enabled
- **Secret Creation**: Creates Kubernetes secret with S3-compatible configuration

### 2. Deploy TempoStack with IBM COS
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
      name: ibm-cos-secret
      type: s3  # IBM COS uses S3-compatible API
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 4Gi
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

### 4. Cleanup IBM Cloud Resources
```bash
./delete-bucket.sh
```

## Key Features Tested

### IBM Cloud Object Storage Integration
- ✅ S3-compatible API access to IBM Cloud Object Storage
- ✅ HMAC authentication with access and secret keys
- ✅ IBM Cloud resource lifecycle management
- ✅ Cross-region storage access and reliability

### TempoStack Configuration
- ✅ Distributed deployment with IBM COS backend
- ✅ Jaeger Query UI with Kubernetes Ingress
- ✅ 200M storage allocation for trace data
- ✅ Resource limits: 4Gi memory, 2000m CPU
- ✅ Multiple service accounts for different components

### Distributed Architecture Benefits
- ✅ Independent component scaling and management
- ✅ Fault tolerance through component separation
- ✅ Better resource utilization for high-volume workloads
- ✅ Production-ready deployment model

### IBM Cloud Services Integration
- ✅ IBM Cloud CLI-based resource provisioning
- ✅ Resource group and service instance management
- ✅ Service key creation with HMAC enabled
- ✅ Secure credential extraction and storage

## IBM COS Secret Configuration

The `ibm-cos-secret` contains S3-compatible configuration for IBM COS:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ibm-cos-secret
data:
  endpoint: <base64-encoded-cos-endpoint>
  bucket: <base64-encoded-bucket-name>
  access_key_id: <base64-encoded-hmac-access-key>
  access_key_secret: <base64-encoded-hmac-secret-key>
```

## TempoStack Component Architecture

The distributed deployment creates multiple components:
- **Distributor**: Handles trace ingestion and load balancing
- **Ingester**: Processes and stores traces to IBM COS
- **Querier**: Retrieves traces for query processing
- **Query Frontend**: Handles query routing and caching
- **Compactor**: Manages trace data compaction and retention

```
[Trace Ingestion] → [Distributor] → [Ingester] → [IBM Cloud Object Storage]
                                       ↑              ↓
[Query UI] ← [Query Frontend] ← [Querier] ← [Compactor]
```

## IBM Cloud Resource Architecture

```
[IBM Cloud Account]
    ↓
[Resource Group: example-user-tracing-tempo]
    ↓
[Cloud Object Storage Service Instance]
    ↓
[COS Bucket: example-user-tempo-bucket-tempo]
    ↓
[Service Key with HMAC Authentication]
```

## Environment Requirements

### IBM Cloud Prerequisites
- Valid IBM Cloud account with billing enabled
- IBM Cloud Object Storage service access
- API key with appropriate IAM permissions
- Resource group creation permissions

### OpenShift Prerequisites
- IBM Cloud credentials available in cluster (`qe-ibmcloud-creds` secret in `kube-system`)
- IBM Cloud CLI tools available in test environment
- Network connectivity to IBM Cloud endpoints
- Sufficient cluster resources for distributed TempoStack deployment

## IBM Cloud CLI Operations

The test performs these IBM Cloud operations:
```bash
# Authentication and setup
ibmcloud login -r us-east
ibmcloud plugin install cloud-object-storage
ibmcloud target -g example-user-tracing-tempo

# Resource creation
ibmcloud resource group-create example-user-tracing-tempo
ibmcloud resource service-instance-create example-user-tempo-bucket-tempo cloud-object-storage standard global

# Bucket and authentication
ibmcloud cos bucket-create --bucket example-user-tempo-bucket-tempo
ibmcloud resource service-key-create example-user-tempo-bucket-tempo Writer --parameters '{"HMAC":true}'
```

## Comparison with TempoMonolithic IBM

### TempoStack Advantages
- **Scalability**: Better for high-volume trace workloads
- **Component Separation**: Independent scaling of ingestion and query
- **Fault Tolerance**: Component isolation reduces single points of failure
- **Production Ready**: Better suited for enterprise deployments

### TempoMonolithic Advantages
- **Simpler Setup**: Single service account vs. multiple service accounts
- **Resource Efficiency**: Lower memory and CPU overhead
- **Unified Component**: All functionality in single pod
- **Development Friendly**: Easier to deploy and manage for testing

## Cleanup Process

The test automatically cleans up IBM Cloud resources:
```bash
./delete-bucket.sh
```

This removes:
- IBM Cloud Object Storage bucket and contents
- Service keys and authentication credentials
- Service instance and associated resources
- Resource group (if empty)

## IBM Cloud Integration Benefits

### Enterprise Features
- **Enterprise Focus**: Strong enterprise features and compliance
- **Hybrid Cloud**: Excellent integration with on-premises infrastructure
- **Security**: Advanced security features and encryption options
- **Global Presence**: Worldwide data centers and edge locations

### S3 Compatibility
- **Standard API**: Uses familiar S3-compatible API and tools
- **Easy Migration**: Simplified migration from other S3-compatible storage
- **Tool Compatibility**: Works with existing S3 tools and libraries
- **Cost Optimization**: Flexible pricing and storage class options

## Troubleshooting

### Common Issues

**IBM Cloud Authentication Failures**:
- Verify API key is valid and has proper IAM permissions
- Check that IBM Cloud CLI is properly installed and configured
- Ensure network connectivity to IBM Cloud APIs

**Resource Creation Issues**:
- Confirm sufficient quota for resource group and service instances
- Verify billing account is active and has available credits
- Check region availability for Cloud Object Storage service

**HMAC Authentication Problems**:
- Ensure service key was created with HMAC enabled
- Verify access keys are correctly extracted and base64 encoded
- Check that COS endpoint and bucket configuration are correct

**TempoStack Component Issues**:
- Validate IBM COS secret contains all required S3 fields
- Check individual component pod logs for S3 authentication errors
- Ensure network connectivity to IBM Cloud Object Storage endpoints

**Distributed Deployment Problems**:
- Verify all TempoStack components can access IBM COS
- Check component inter-communication and service discovery
- Monitor resource usage across all components

## Performance Considerations

### IBM COS Optimization
- **Regional Placement**: Deploy storage in same region as cluster for lower latency
- **Storage Classes**: Use appropriate storage classes based on access patterns
- **Bandwidth**: Ensure sufficient network bandwidth for high-volume trace workloads
- **Monitoring**: Use IBM Cloud Monitoring to track storage performance metrics

### TempoStack Scaling
- **Component Resources**: Allocate appropriate CPU and memory for each component
- **Ingestion Rate**: Scale distributors and ingesters based on trace volume
- **Query Performance**: Scale queriers based on query load patterns
- **Storage Growth**: Monitor and plan for trace data growth over time

## Architecture Benefits

### Distributed Deployment
- **High Availability**: Component redundancy and fault tolerance
- **Horizontal Scaling**: Independent scaling of each component
- **Performance**: Optimized for high-throughput trace processing
- **Maintainability**: Easier to update and maintain individual components

### IBM Cloud Integration
- **Enterprise Grade**: Built for enterprise workloads and compliance
- **Hybrid Capabilities**: Seamless integration with on-premises systems
- **Global Scale**: Worldwide infrastructure with local data residency
- **Security**: Advanced encryption and security capabilities

This test demonstrates how TempoStack can effectively integrate with IBM Cloud Object Storage using S3-compatible APIs, providing a reliable and enterprise-grade solution for trace data persistence in distributed IBM Cloud environments.
