# TempoStack with Google Cloud Storage (GCS)

This test validates TempoStack deployment using Google Cloud Storage (GCS) as the backend for trace data persistence. It demonstrates how to configure and deploy a distributed Tempo instance with traditional GCP service account key authentication for production-scale workloads.

## Test Overview

### Purpose
- **GCS Integration**: Tests Google Cloud Storage backend with distributed TempoStack
- **Service Account Authentication**: Validates traditional GCP authentication using service account keys
- **Distributed Architecture**: Verifies multi-component deployment with cloud storage
- **Production Scalability**: Demonstrates enterprise-ready deployment model

### Components
- **TempoStack**: Distributed Tempo deployment with GCS storage
- **Google Cloud Storage**: Google cloud object storage for trace data persistence
- **GCP Service Account**: Authentication credentials for GCS access
- **Storage Bucket**: Container for trace data organization in GCS
- **Multiple Service Accounts**: Component-specific authentication for distributed architecture

## Deployment Steps

### 1. Create GCS Storage Infrastructure
```bash
./create-bucket.sh
```

The [`create-bucket.sh`](create-bucket.sh) script performs:
- **Credential Extraction**: Extracts GCP service account credentials from cluster secret
- **Bucket Cleanup**: Removes existing bucket if present to ensure clean state
- **Bucket Creation**: Creates new GCS bucket for trace storage
- **Secret Creation**: Creates Kubernetes secret with GCS configuration and credentials

### 2. Deploy TempoStack with GCS Storage
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
      name: gcs-secret
      type: gcs
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

### Google Cloud Storage Integration
- ✅ GCS bucket access and authentication using service account keys
- ✅ Bucket lifecycle management (creation, cleanup, recreation)
- ✅ Secure credential handling with Kubernetes secrets
- ✅ Cross-region storage access and reliability

### TempoStack Configuration
- ✅ Distributed deployment with GCS backend
- ✅ Jaeger Query UI with Kubernetes Ingress
- ✅ 200M storage allocation for trace data
- ✅ Resource limits: 2Gi memory, 2000m CPU
- ✅ Multiple service accounts for different components

### Distributed Architecture Benefits
- ✅ Independent component scaling and management
- ✅ Fault tolerance through component separation
- ✅ Better resource utilization for high-volume workloads
- ✅ Production-ready deployment model

### Authentication Method
- ✅ Service account key-based authentication
- ✅ Credential extraction from cluster secrets
- ✅ JSON key file handling and mounting
- ✅ Secure storage of authentication materials

## GCS Storage Secret Configuration

The `gcs-secret` contains GCS-specific configuration:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcs-secret
data:
  bucketname: <base64-encoded-bucket-name>
  key.json: <base64-encoded-service-account-key>
```

## TempoStack Component Architecture

The distributed deployment creates multiple components:
- **Distributor**: Handles trace ingestion and load balancing
- **Ingester**: Processes and stores traces to GCS
- **Querier**: Retrieves traces for query processing
- **Query Frontend**: Handles query routing and caching
- **Compactor**: Manages trace data compaction and retention

```
[Trace Ingestion] → [Distributor] → [Ingester] → [Google Cloud Storage]
                                       ↑              ↓
[Query UI] ← [Query Frontend] ← [Querier] ← [Compactor]
```

## Environment Requirements

### GCP Prerequisites
- Valid GCP project with Storage API enabled
- Service account with Storage Object Admin permissions
- GCS bucket creation permissions
- Service account key available in cluster (extracted from `kube-system/gcp-credentials`)

### OpenShift Prerequisites
- Sufficient cluster resources for distributed TempoStack deployment
- Access to extract secrets from `kube-system` namespace
- Network connectivity to Google Cloud Storage endpoints

## Comparison with TempoMonolithic GCS

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

## Authentication vs. Workload Identity Federation

### Service Account Key Approach (This Test)
- **Traditional Method**: Uses long-lived service account keys
- **Direct Authentication**: JSON key file provides direct GCS access
- **Simpler Setup**: Straightforward credential configuration
- **Security Considerations**: Requires careful key management and rotation

### Workload Identity Federation Approach (Alternative)
- **Modern Security**: Uses temporary credentials and OIDC federation
- **No Long-lived Keys**: Eliminates key management overhead
- **Enhanced Security**: Reduces credential exposure and improves audit trail
- **Complex Setup**: Requires additional GCP and OpenShift configuration

## Cleanup Process

The test includes bucket cleanup in the setup phase:
```bash
# Remove existing bucket if present
gcloud alpha storage rm --recursive gs://$BUCKET_NAME

# Create fresh bucket for testing
gsutil mb gs://$BUCKET_NAME
```

## GCS Integration Benefits

### Storage Features
- **Multi-Regional Storage**: Automatic data replication across regions
- **Lifecycle Management**: Automated data archiving and deletion policies
- **Access Control**: Fine-grained IAM permissions and bucket policies
- **Monitoring**: Comprehensive Cloud Monitoring integration

### Performance Optimization
- **Global Infrastructure**: Extensive global network and edge locations
- **Integration**: Deep integration with Google Cloud ecosystem
- **Analytics**: Built-in integration with BigQuery and other analytics tools
- **Cost Management**: Flexible storage classes and pricing options

## Troubleshooting

### Common Issues

**GCP Authentication Failures**:
- Verify service account key is valid and not expired
- Check that service account has Storage Object Admin permissions
- Ensure GCP project and billing are properly configured

**Bucket Access Issues**:
- Confirm bucket exists and is in the correct GCP project
- Verify network connectivity to GCS endpoints
- Check bucket policies and IAM permissions

**Credential Extraction Problems**:
- Ensure `gcp-credentials` secret exists in `kube-system` namespace
- Verify service account JSON format is correct
- Check that extracted credentials have proper permissions

**TempoStack Component Issues**:
- Validate GCS secret contains all required fields (`bucketname`, `key.json`)
- Check individual component pod logs for authentication errors
- Ensure sufficient cluster resources for all components

**Distributed Deployment Problems**:
- Verify all TempoStack components can access GCS
- Check component inter-communication and service discovery
- Monitor resource usage across all components

## Performance Considerations

### GCS Optimization
- **Regional Placement**: Deploy storage in same region as cluster for lower latency
- **Storage Classes**: Use appropriate storage classes based on access patterns
- **Bandwidth**: Ensure sufficient network bandwidth for high-volume trace workloads
- **Monitoring**: Use Cloud Monitoring to track storage performance metrics

### TempoStack Scaling
- **Component Resources**: Allocate appropriate CPU and memory for each component
- **Ingestion Rate**: Scale distributors and ingesters based on trace volume
- **Query Performance**: Scale queriers based on query load patterns
- **Storage Growth**: Monitor and plan for trace data growth over time

This test demonstrates how TempoStack can effectively integrate with Google Cloud Storage using traditional service account authentication, providing a reliable and scalable solution for trace data persistence in distributed Google Cloud environments.
