# TempoMonolithic with AWS S3 Storage (Traditional Credentials)

This configuration blueprint demonstrates TempoMonolithic deployment with traditional AWS S3 storage using static access keys. This setup provides a straightforward approach to AWS S3 integration suitable for development environments, testing scenarios, and deployments where advanced credential management is not required.

## Overview

This test validates traditional AWS S3 integration featuring:
- **Static AWS Credentials**: Traditional access key and secret key authentication
- **S3 Bucket Management**: Automated bucket creation and cleanup
- **Simple Configuration**: Minimal setup with direct credential management
- **Development-Friendly**: Easy setup for testing and development environments
- **CI Integration**: Supports both OpenShift CI and local development workflows

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ TempoMonolithic         │───▶│   AWS S3 Service         │───▶│ S3 Bucket               │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │    │ ┌─────────────────────┐ │
│ │ S3 Backend          │ │    │ │ S3 API              │  │    │ │ example-user-monolithic   │ │
│ │ - Static Creds      │ │    │ │ - us-east-2         │  │    │ │ -s3                 │ │
│ │ - Direct Access     │ │    │ │ - Standard Endpoint │  │    │ │ - Trace Objects     │ │
│ └─────────────────────┘ │    │ └─────────────────────┘  │    │ └─────────────────────┘ │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘
          │                              ▲
          │ ┌────────────────────────────┴─────────────────────────┐
          │ │               Kubernetes Secret                       │
          └▶│ ┌─────────────────────┐ ┌─────────────────────┐     │
            │ │ Access Key ID       │ │ Secret Access Key   │     │
            │ │ - Static Credential │ │ - Static Credential │     │
            │ └─────────────────────┘ └─────────────────────┘     │
            │ ┌─────────────────────┐ ┌─────────────────────┐     │
            │ │ Bucket Name         │ │ S3 Endpoint         │     │
            │ │ - example-user-monolithic │ │ - Regional Endpoint │     │
            │ └─────────────────────┘ └─────────────────────┘     │
            └───────────────────────────────────────────────────────┘

Credential Sources:
┌─────────────────────────┐    ┌─────────────────────────┐
│ OpenShift CI            │    │ Local Development       │
│ - CLUSTER_PROFILE_DIR   │    │ - kube-system secret    │
│ - .awscred file         │    │ - aws-creds             │
│ - CI Environment        │    │ - Manual Setup          │
└─────────────────────────┘    └─────────────────────────┘
```

## Prerequisites

- OpenShift cluster
- AWS account with S3 access permissions
- Tempo Operator installed
- AWS credentials configured via one of:
  - OpenShift CI: `CLUSTER_PROFILE_DIR` with `.awscred` file
  - Local: `aws-creds` secret in `kube-system` namespace
- `oc` CLI access

## Step-by-Step Configuration

### Step 1: Create S3 Bucket and Credentials

Execute the bucket creation script that handles both credential sourcing and bucket management:

```bash
./create-bucket.sh
```

**Script Functionality from [`create-bucket.sh`](./create-bucket.sh)**:

#### Credential Detection and Sourcing
```bash
# OpenShift CI credential sourcing
if [ -n "${CLUSTER_PROFILE_DIR}" ]; then
    export AWS_ACCESS_KEY_ID=$(grep "aws_access_key_id=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
    export AWS_SECRET_ACCESS_KEY=$(grep "aws_secret_access_key=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
else
    # Local development credential sourcing
    export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_access_key_id' | base64 -d)
    export AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_secret_access_key' | base64 -d)
fi
```

**Credential Sourcing Strategy**:
- **CI Integration**: Automatically detects OpenShift CI environment via `CLUSTER_PROFILE_DIR`
- **Local Development**: Falls back to cluster-stored credentials
- **Error Handling**: Validates credential extraction and exits on failure
- **Flexibility**: Supports multiple deployment environments

#### S3 Bucket Management
```bash
BUCKET_NAME="example-user-monolithic-s3"
REGION="us-east-2"
AWS_BUCKET_ENDPOINT="https://s3.${REGION}.amazonaws.com"

# Check and remove existing bucket
if aws s3api head-bucket --bucket $BUCKET_NAME --region $REGION 2>/dev/null; then
    aws s3 rb s3://$BUCKET_NAME --region $REGION --force
    sleep 30  # Wait for eventual consistency
fi

# Create new bucket with regional configuration
aws s3api create-bucket --bucket $BUCKET_NAME --region $REGION \
  --create-bucket-configuration LocationConstraint=$REGION
```

**Bucket Configuration Details**:
- **Fixed Naming**: Uses predictable bucket name for testing consistency
- **Regional Deployment**: Creates bucket in us-east-2 for cost optimization
- **Cleanup Handling**: Removes existing bucket to ensure clean test state
- **Eventual Consistency**: Waits for AWS eventual consistency after deletion

#### Kubernetes Secret Creation
```bash
kubectl -n $NAMESPACE create secret generic s3-secret \
  --from-literal=bucket="$BUCKET_NAME" \
  --from-literal=endpoint="$AWS_BUCKET_ENDPOINT" \
  --from-literal=access_key_id="$AWS_ACCESS_KEY_ID" \
  --from-literal=access_key_secret="$AWS_SECRET_ACCESS_KEY"
```

**Secret Configuration**:
- **Complete S3 Info**: Includes bucket, endpoint, and credentials
- **Regional Endpoint**: Uses region-specific S3 endpoint for performance
- **Traditional Auth**: Static access key and secret key authentication

### Step 2: Deploy TempoMonolithic with S3 Storage

Apply TempoMonolithic with traditional S3 configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: s3-secret
EOF
```

**Key Configuration Elements**:

#### S3 Storage Backend
- `backend: s3`: Specifies AWS S3 as the trace storage backend
- `secret: s3-secret`: References the Kubernetes secret containing S3 credentials
- **Default Configuration**: Uses operator defaults for other S3 settings

#### Simplified Deployment
- **Minimal Configuration**: Only essential S3 parameters specified
- **Default Settings**: Relies on operator defaults for other configurations
- **No TLS/UI**: Basic deployment focused on S3 storage validation

**Storage Configuration Details**:
The operator automatically configures:
- **S3 Client**: AWS SDK client with provided credentials
- **Bucket Operations**: Read/write permissions to specified bucket
- **Regional Access**: Uses endpoint region for optimal performance
- **Object Lifecycle**: Automatic object management for trace blocks

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Verify TempoMonolithic Readiness

Wait for TempoMonolithic to initialize with S3 storage:

```bash
kubectl get --namespace $NAMESPACE tempomonolithics simplest \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

#### S3 Integration Validation
```bash
# Check TempoMonolithic status
oc describe tempomonolithic simplest

# Verify S3 secret mounting
oc get pods -l app.kubernetes.io/name=tempo-monolithic
oc describe pod tempo-simplest-0

# Check S3 connectivity in logs
oc logs tempo-simplest-0 | grep -i s3
```

### Step 4: Generate Traces for S3 Storage

Create sample traces to validate S3 storage functionality:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
        args:
        - traces
        - --otlp-endpoint=tempo-simplest:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Generation Flow**:
1. **Trace Creation**: telemetrygen generates 10 sample traces
2. **OTLP Ingestion**: Traces sent to TempoMonolithic via OTLP gRPC
3. **S3 Storage**: TempoMonolithic stores traces as objects in S3 bucket
4. **AWS Authentication**: Uses static credentials for S3 API calls

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 5: Verify S3-Stored Traces

Validate that traces are properly stored and retrievable from S3:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
spec:
  template:
    spec:
      containers:
      - name: verify-traces
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          echo "✓ Successfully verified \$num_traces traces stored in S3"
      restartPolicy: Never
EOF
```

**Verification Process**:
- **API Query**: Uses Tempo's search API to retrieve traces
- **S3 Retrieval**: TempoMonolithic queries S3 objects for trace data
- **Count Validation**: Confirms all generated traces are accessible
- **Storage Validation**: Indirectly validates S3 storage and retrieval

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## Traditional AWS S3 Features

### 1. **Static Credential Management**

#### Access Key Configuration
```yaml
# Static credentials in Kubernetes secret
apiVersion: v1
kind: Secret
metadata:
  name: s3-secret
stringData:
  bucket: "tempo-traces-bucket"
  endpoint: "https://s3.us-east-2.amazonaws.com"
  access_key_id: "AKIA..."
  access_key_secret: "secret..."
```

#### Regional Endpoint Configuration
```bash
# Regional endpoints for performance optimization
US_EAST_1="https://s3.us-east-1.amazonaws.com"
US_EAST_2="https://s3.us-east-2.amazonaws.com"
US_WEST_2="https://s3.us-west-2.amazonaws.com"
EU_WEST_1="https://s3.eu-west-1.amazonaws.com"
```

### 2. **S3 Bucket Operations**

#### Bucket Lifecycle Management
```bash
# Automated bucket creation with regional configuration
aws s3api create-bucket \
  --bucket tempo-traces \
  --region us-east-2 \
  --create-bucket-configuration LocationConstraint=us-east-2

# Bucket policy for application access
aws s3api put-bucket-policy \
  --bucket tempo-traces \
  --policy file://bucket-policy.json
```

#### Object Storage Patterns
```bash
# Tempo stores traces as S3 objects with hierarchical structure
# bucket/
# ├── index/
# │   ├── tenant-id/
# │   │   └── yyyy-mm-dd/
# │   │       └── block-id.json
# └── blocks/
#     ├── tenant-id/
#     │   └── yyyy-mm-dd/
#     │       └── block-id/
#     │           ├── data.parquet
#     │           ├── meta.json
#     │           └── bloom_filter
```

### 3. **Performance Optimization**

#### Regional Deployment Best Practices
```yaml
# Match TempoMonolithic region with S3 bucket region
spec:
  storage:
    traces:
      s3:
        region: us-east-2  # Same as bucket region
        endpoint: https://s3.us-east-2.amazonaws.com
```

#### Connection Pooling and Timeouts
```yaml
spec:
  extraConfig:
    tempo:
      storage:
        s3:
          # Connection optimization
          max_idle_conns: 100
          max_idle_conns_per_host: 100
          idle_conn_timeout: 90s
          # Request timeouts
          timeout: 30s
          retry_count: 3
```

## Advanced S3 Configuration

### 1. **Security Hardening**

#### Bucket Encryption
```bash
# Enable S3 bucket encryption
aws s3api put-bucket-encryption \
  --bucket tempo-traces \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "AES256"
      }
    }]
  }'
```

#### IAM Policy Optimization
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ],
    "Resource": [
      "arn:aws:s3:::tempo-traces/*",
      "arn:aws:s3:::tempo-traces"
    ]
  }]
}
```

### 2. **Cost Optimization**

#### S3 Storage Classes
```yaml
spec:
  extraConfig:
    tempo:
      storage:
        s3:
          storage_class: STANDARD_IA  # Infrequent Access for cost savings
```

#### Lifecycle Policies
```bash
# Automated lifecycle management
aws s3api put-bucket-lifecycle-configuration \
  --bucket tempo-traces \
  --lifecycle-configuration '{
    "Rules": [{
      "Status": "Enabled",
      "Filter": {"Prefix": "blocks/"},
      "Transitions": [{
        "Days": 30,
        "StorageClass": "GLACIER"
      }]
    }]
  }'
```

### 3. **High Availability**

#### Cross-Region Replication
```bash
# Enable versioning (required for replication)
aws s3api put-bucket-versioning \
  --bucket tempo-traces \
  --versioning-configuration Status=Enabled

# Configure cross-region replication
aws s3api put-bucket-replication \
  --bucket tempo-traces \
  --replication-configuration file://replication-config.json
```

#### Multi-Region Access Points
```yaml
# Configure for multi-region deployment
spec:
  storage:
    traces:
      s3:
        endpoint: https://tempo-traces.s3-global.amazonaws.com
        # Uses AWS Global Accelerator for optimal routing
```

## Production Deployment Considerations

### 1. **Credential Security**

#### Credential Rotation
```bash
# Regular credential rotation
aws iam create-access-key --user-name tempo-service-user
# Update Kubernetes secret with new credentials
# Delete old access key after validation
aws iam delete-access-key --access-key-id AKIA... --user-name tempo-service-user
```

#### Secret Management
```yaml
# Use external secret management
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secret-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-2
```

### 2. **Monitoring and Observability**

#### S3 Performance Metrics
```bash
# Monitor S3 operation latency
oc port-forward deployment/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep s3_request_duration

# Check S3 error rates
curl http://localhost:3200/metrics | grep s3_request_errors_total
```

#### Cost Monitoring
```bash
# Monitor S3 storage costs
aws ce get-cost-and-usage \
  --time-period Start=2024-01-01,End=2024-01-31 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --group-by Type=DIMENSION,Key=SERVICE
```

### 3. **Backup and Disaster Recovery**

#### Backup Strategy
```yaml
# Configure backup bucket for disaster recovery
spec:
  storage:
    traces:
      s3:
        bucket: tempo-traces-primary
        backup_bucket: tempo-traces-backup
```

#### Point-in-Time Recovery
```bash
# Use S3 versioning for point-in-time recovery
aws s3api list-object-versions --bucket tempo-traces --prefix blocks/
aws s3api get-object --bucket tempo-traces --key blocks/trace.json --version-id version-id
```

## Troubleshooting S3 Integration

### 1. **Credential Issues**

#### Access Key Validation
```bash
# Test AWS credentials
aws sts get-caller-identity

# Verify S3 access
aws s3 ls s3://example-user-monolithic-s3

# Check bucket permissions
aws s3api get-bucket-acl --bucket example-user-monolithic-s3
```

#### Secret Configuration
```bash
# Check secret contents
oc get secret s3-secret -o yaml

# Verify secret mounting in pod
oc describe pod tempo-simplest-0 | grep -A10 "Mounts:"

# Test S3 access from pod
oc exec tempo-simplest-0 -- aws s3 ls s3://example-user-monolithic-s3
```

### 2. **S3 Connectivity Problems**

#### Network Connectivity
```bash
# Test S3 endpoint connectivity
oc exec tempo-simplest-0 -- curl -I https://s3.us-east-2.amazonaws.com

# Check DNS resolution
oc exec tempo-simplest-0 -- nslookup s3.us-east-2.amazonaws.com

# Verify TLS connectivity
oc exec tempo-simplest-0 -- openssl s_client -connect s3.us-east-2.amazonaws.com:443
```

#### Bucket Operations
```bash
# Monitor S3 operations in Tempo logs
oc logs tempo-simplest-0 | grep -i "s3\|bucket\|aws"

# Check for S3 errors
oc logs tempo-simplest-0 | grep -i "error\|fail\|denied"

# Verify bucket region configuration
aws s3api get-bucket-location --bucket example-user-monolithic-s3
```

### 3. **Performance Issues**

#### Latency Analysis
```bash
# Monitor S3 request latency
oc port-forward deployment/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep s3_request_duration_seconds

# Check for throttling
oc logs tempo-simplest-0 | grep -i "throttl\|rate.limit\|503"
```

#### Throughput Optimization
```yaml
# Optimize S3 configuration for high throughput
spec:
  extraConfig:
    tempo:
      storage:
        s3:
          max_retries: 3
          request_timeout: 30s
          part_size: 16MB
          max_concurrent_requests: 10
```

## Related Configurations

- [AWS STS Integration](../aws-sts-tempostack/README.md) - Modern STS-based authentication
- [AWS CCO Integration](../aws-sts-cco-monolithic/README.md) - OpenShift CCO management
- [TempoStack AWS](../tempostack-aws/README.md) - Distributed AWS S3 deployment

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/monolithic-aws
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test provides a baseline AWS S3 integration using traditional access keys, suitable for development and testing environments. For production deployments, consider using AWS STS or CCO-based authentication for enhanced security.

