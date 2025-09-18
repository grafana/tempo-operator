# TempoMonolithic with IBM Cloud Object Storage

This test validates TempoMonolithic deployment using IBM Cloud Object Storage (COS) as the S3-compatible backend for trace data persistence. It demonstrates how to configure and deploy a single-component Tempo instance with IBM Cloud storage integration using HMAC authentication.

## Test Overview

### Purpose
- **IBM Cloud Object Storage Integration**: Tests IBM COS as S3-compatible backend with TempoMonolithic
- **HMAC Authentication**: Validates IBM Cloud storage access using HMAC keys
- **Enterprise Cloud Integration**: Demonstrates integration with IBM Cloud services
- **End-to-End Workflow**: Tests complete trace lifecycle from ingestion to querying

### Components
- **TempoMonolithic**: Single-component Tempo deployment with IBM COS storage
- **IBM Cloud Object Storage**: IBM's S3-compatible object storage service
- **IBM Cloud Resource Group**: Logical container for IBM Cloud resources
- **Service Instance**: IBM Cloud Object Storage service instance
- **HMAC Credentials**: Access keys for S3-compatible authentication

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

### 2. Deploy TempoMonolithic with IBM COS
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
      backend: s3  # IBM COS uses S3-compatible API
      s3:
        secret: ibm-cos-secret  # Contains IBM COS configuration
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f 03-generate-traces.yaml
kubectl apply -f 04-verify-traces.yaml
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

### TempoMonolithic Configuration
- ✅ Single-component deployment with IBM COS backend
- ✅ S3-compatible storage configuration
- ✅ Direct IBM Cloud Object Storage integration
- ✅ Automatic trace data organization in COS buckets

### IBM Cloud Services Integration
- ✅ IBM Cloud CLI-based resource provisioning
- ✅ Resource group and service instance management
- ✅ Service key creation with HMAC enabled
- ✅ Secure credential extraction and storage

### End-to-End Validation
- ✅ Trace ingestion through OTLP endpoint
- ✅ Successful storage to IBM Cloud Object Storage
- ✅ Trace retrieval and querying capabilities
- ✅ Complete observability pipeline functionality

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

## IBM Cloud Resource Architecture

```
[IBM Cloud Account]
    ↓
[Resource Group: example-user-tracing-mono]
    ↓
[Cloud Object Storage Service Instance]
    ↓
[COS Bucket: example-user-tempo-bucket-mono]
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

## IBM Cloud CLI Operations

The test performs these IBM Cloud operations:
```bash
# Authentication and setup
ibmcloud login -r us-east
ibmcloud plugin install cloud-object-storage
ibmcloud target -g example-user-tracing-mono

# Resource creation
ibmcloud resource group-create example-user-tracing-mono
ibmcloud resource service-instance-create example-user-tempo-bucket-mono cloud-object-storage standard global

# Bucket and authentication
ibmcloud cos bucket-create --bucket example-user-tempo-bucket-mono
ibmcloud resource service-key-create example-user-tempo-bucket-mono Writer --parameters '{"HMAC":true}'
```

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

## Comparison with Other Cloud Providers

### IBM Cloud Advantages
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

**TempoMonolithic Startup Issues**:
- Validate IBM COS secret contains all required S3 fields
- Check TempoMonolithic pod logs for S3 authentication errors
- Ensure network connectivity to IBM Cloud Object Storage endpoints

## Architecture Benefits

### Monolithic Deployment
- **Simplicity**: Single component reduces operational complexity
- **Resource Efficiency**: Lower overhead compared to distributed deployments
- **Quick Setup**: Faster deployment for development and testing
- **Cost Effective**: Reduced infrastructure costs for smaller workloads

### IBM Cloud Integration
- **Enterprise Grade**: Built for enterprise workloads and compliance
- **Hybrid Capabilities**: Seamless integration with on-premises systems
- **Global Scale**: Worldwide infrastructure with local data residency
- **Security**: Advanced encryption and security capabilities

This test demonstrates how TempoMonolithic can effectively integrate with IBM Cloud Object Storage using S3-compatible APIs, providing a reliable and enterprise-grade solution for trace data persistence in IBM Cloud environments.
