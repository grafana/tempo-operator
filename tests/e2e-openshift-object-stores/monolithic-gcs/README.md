# TempoMonolithic with Google Cloud Storage (GCS)

This test validates TempoMonolithic deployment using Google Cloud Storage (GCS) as the backend for trace data persistence. It demonstrates how to configure and deploy a single-component Tempo instance with traditional GCP service account key authentication.

## Test Overview

### Purpose
- **GCS Integration**: Tests Google Cloud Storage backend with TempoMonolithic
- **Service Account Authentication**: Validates traditional GCP authentication using service account keys
- **Monolithic Architecture**: Verifies single-component deployment with cloud storage
- **End-to-End Workflow**: Tests complete trace lifecycle from ingestion to querying

### Components
- **TempoMonolithic**: Single-component Tempo deployment with GCS storage
- **Google Cloud Storage**: Google cloud object storage for trace data persistence
- **GCP Service Account**: Authentication credentials for GCS access
- **Storage Bucket**: Container for trace data organization in GCS

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

### 2. Deploy TempoMonolithic with GCS Storage
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
      backend: gcs
      gcs:
        secret: gcs-secret  # Contains GCS bucket and service account key
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f 03-generate-traces.yaml
kubectl apply -f 04-verify-traces.yaml
```

## Key Features Tested

### Google Cloud Storage Integration
- ✅ GCS bucket access and authentication using service account keys
- ✅ Bucket lifecycle management (creation, cleanup, recreation)
- ✅ Secure credential handling with Kubernetes secrets
- ✅ Cross-region storage access and reliability

### TempoMonolithic Configuration
- ✅ Single-component deployment with GCS backend
- ✅ Simplified resource management and operation
- ✅ Direct GCS integration with service account authentication
- ✅ Automatic trace data organization in GCS buckets

### Authentication Method
- ✅ Service account key-based authentication
- ✅ Credential extraction from cluster secrets
- ✅ JSON key file handling and mounting
- ✅ Secure storage of authentication materials

### End-to-End Validation
- ✅ Trace ingestion through OTLP endpoint
- ✅ Successful storage to Google Cloud Storage
- ✅ Trace retrieval and querying capabilities
- ✅ Complete observability pipeline functionality

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

## Environment Requirements

### GCP Prerequisites
- Valid GCP project with Storage API enabled
- Service account with Storage Object Admin permissions
- GCS bucket creation permissions
- Service account key available in cluster (extracted from `kube-system/gcp-credentials`)

### OpenShift Prerequisites
- Sufficient cluster resources for TempoMonolithic deployment
- Access to extract secrets from `kube-system` namespace
- Network connectivity to Google Cloud Storage endpoints

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

## Comparison with Other Cloud Providers

### GCS Advantages
- **Global Infrastructure**: Extensive global network and edge locations
- **Integration**: Deep integration with Google Cloud ecosystem
- **Performance**: Optimized for high-throughput and low-latency access
- **Analytics**: Built-in integration with BigQuery and other analytics tools

### Storage Features
- **Multi-Regional Storage**: Automatic data replication across regions
- **Lifecycle Management**: Automated data archiving and deletion policies
- **Access Control**: Fine-grained IAM permissions and bucket policies
- **Monitoring**: Comprehensive Cloud Monitoring integration

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

**TempoMonolithic Startup Issues**:
- Validate GCS secret contains all required fields (`bucketname`, `key.json`)
- Check TempoMonolithic pod logs for authentication errors
- Ensure sufficient cluster resources for deployment

## Architecture Benefits

### Monolithic Deployment
- **Simplicity**: Single component reduces operational complexity
- **Resource Efficiency**: Lower overhead compared to distributed deployments
- **Quick Setup**: Faster deployment for development and testing
- **Cost Effective**: Reduced infrastructure costs for smaller workloads

### GCS Integration
- **Scalability**: Leverage GCS's automatic scaling capabilities
- **Durability**: Built-in data protection and redundancy
- **Performance**: Optimized storage performance for trace workloads
- **Global Access**: Worldwide data availability and access

This test demonstrates how TempoMonolithic can effectively integrate with Google Cloud Storage using traditional service account authentication, providing a reliable and scalable solution for trace data persistence in Google Cloud environments.
