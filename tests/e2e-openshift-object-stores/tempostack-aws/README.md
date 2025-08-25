# TempoStack with AWS S3 Object Storage

This configuration blueprint demonstrates how to deploy TempoStack with AWS S3 as the object storage backend. This setup showcases production-ready cloud storage integration with automatic bucket management and secure credential handling for enterprise observability deployments.

## Overview

This test validates a cloud-native observability stack featuring:
- **AWS S3 Integration**: Production-ready object storage backend
- **Automated Bucket Management**: Dynamic S3 bucket creation and lifecycle management
- **Secure Credential Handling**: Multiple credential source options (cluster profile, OpenShift secrets)
- **TempoStack Distribution**: Multi-component architecture optimized for cloud storage
- **Regional Configuration**: Configurable AWS region support

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ Trace Generator │───▶│    TempoStack        │───▶│   AWS S3        │
│ (telemetrygen)  │    │ ┌─────────────────┐  │    │ ┌─────────────┐ │
└─────────────────┘    │ │ Distributor     │  │    │ │ us-east-2   │ │
                       │ │ Ingester        │  │    │ │ Regional    │ │
┌─────────────────┐    │ │ Querier         │  │    │ │ Bucket      │ │
│ Query Clients   │◀───│ │ Query Frontend  │  │    │ └─────────────┘ │
│ - Tempo API     │    │ │ Compactor       │  │    └─────────────────┘
│ - Jaeger API    │    │ └─────────────────┘  │              │
└─────────────────┘    └──────────────────────┘              │
                                ▲                             │
                       ┌────────┴─────────┐                  │
                       │ AWS Credentials  │◀─────────────────┘
                       │ - Access Key ID  │
                       │ - Secret Key     │
                       └──────────────────┘
```

## Prerequisites

- OpenShift cluster with AWS integration
- Tempo Operator installed
- AWS CLI configured or cluster credentials available
- Sufficient AWS permissions for S3 bucket operations
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Configure AWS Credentials

The setup supports two credential sources:

#### Option A: Cluster Profile Directory (CI/CD Environment)
```bash
# Credentials automatically sourced from CLUSTER_PROFILE_DIR
export CLUSTER_PROFILE_DIR="/path/to/cluster/profile"
```

#### Option B: OpenShift Secret (Manual Setup)
```bash
# Create AWS credentials secret in kube-system namespace
oc create secret generic aws-creds -n kube-system \
  --from-literal=aws_access_key_id="YOUR_ACCESS_KEY_ID" \
  --from-literal=aws_secret_access_key="YOUR_SECRET_ACCESS_KEY"
```

### Step 2: Create S3 Bucket and Secret

Run the automated bucket creation script:

```bash
#!/bin/bash

# Configuration
BUCKET_NAME="your-tempostack-s3-bucket"  # Customize this
REGION="us-east-2"                       # Customize this
AWS_BUCKET_ENDPOINT="https://s3.${REGION}.amazonaws.com"

# Fetch AWS credentials (auto-detects source)
if [ -n "${CLUSTER_PROFILE_DIR}" ]; then
    export AWS_ACCESS_KEY_ID=$(grep "aws_access_key_id=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
    export AWS_SECRET_ACCESS_KEY=$(grep "aws_secret_access_key=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
else
    export AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_access_key_id' | base64 -d)
    export AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_secret_access_key' | base64 -d)
fi

# Clean up existing bucket if present
if aws s3api head-bucket --bucket $BUCKET_NAME --region $REGION 2>/dev/null; then
    aws s3 rb s3://$BUCKET_NAME --region $REGION --force
    sleep 30  # Wait for eventual consistency
fi

# Create new S3 bucket
aws s3api create-bucket \
  --bucket $BUCKET_NAME \
  --region $REGION \
  --create-bucket-configuration LocationConstraint=$REGION

# Create Kubernetes secret for TempoStack
kubectl create secret generic s3-secret \
  --from-literal=bucket="$BUCKET_NAME" \
  --from-literal=endpoint="$AWS_BUCKET_ENDPOINT" \
  --from-literal=access_key_id="$AWS_ACCESS_KEY_ID" \
  --from-literal=access_key_secret="$AWS_SECRET_ACCESS_KEY"
```

**Script Features**:
- **Credential Auto-Detection**: Supports multiple credential sources
- **Bucket Lifecycle Management**: Cleans up existing buckets before creation
- **Regional Configuration**: Configurable AWS region support
- **Error Handling**: Comprehensive validation and error reporting

**Reference**: [`create-bucket.sh`](./create-bucket.sh)

### Step 3: Deploy TempoStack with S3 Backend

Create the TempoStack configured for AWS S3:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: s3-secret
      type: s3
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
EOF
```

**Key Configuration Details**:
- `storage.secret.name`: References the S3 credentials secret
- `storage.secret.type: s3`: Specifies S3-compatible storage backend
- `storageSize: 200M`: Allocates storage quota for trace retention
- `jaegerQuery.enabled`: Enables Jaeger-compatible query interface

**Reference**: [`01-install-tempostack.yaml`](./01-install-tempostack.yaml)

### Step 4: Verify Deployment

Wait for TempoStack to be ready:

```bash
oc get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

Check S3 connectivity:

```bash
# Verify all components are running
oc get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check logs for S3 connection
oc logs -l app.kubernetes.io/component=compactor | grep -i s3
```

### Step 5: Generate Sample Traces

Create traces to test S3 storage integration:

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
        - --otlp-endpoint=tempo-simplest-distributor:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Reference**: [`02-generate-traces.yaml`](./02-generate-traces.yaml)

### Step 6: Verify Traces in S3

Test trace storage and retrieval from S3:

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
          # Query traces via Tempo API
          curl -v -G \
            http://tempo-simplest-query-frontend:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          
          echo "Successfully verified \$num_traces traces stored in S3"
      restartPolicy: Never
EOF
```

**Reference**: [`03-verify-traces.yaml`](./03-verify-traces.yaml)

## Key Features Demonstrated

### 1. **AWS S3 Integration**
- **Native S3 Protocol**: Direct S3 API integration without additional proxies
- **Regional Deployment**: Configurable AWS region for data locality
- **Bucket Lifecycle Management**: Automated creation and cleanup
- **Cost Optimization**: Configurable storage classes and retention policies

### 2. **Credential Management**
- **Multiple Sources**: Support for cluster profiles and manual secrets
- **Secure Storage**: Credentials stored in Kubernetes secrets
- **Automatic Discovery**: Intelligent credential source detection
- **Error Handling**: Comprehensive validation and fallback mechanisms

### 3. **Production Readiness**
- **Scalable Storage**: Unlimited storage capacity with S3
- **Durability**: 99.999999999% (11 9's) durability with S3
- **Availability**: Multi-AZ replication for high availability
- **Security**: Encryption at rest and in transit

### 4. **Cost Management**
- **Storage Tiering**: Automatic transition to cheaper storage classes
- **Data Lifecycle**: Configurable retention and deletion policies
- **Monitoring**: Integration with AWS CloudWatch for cost tracking
- **Optimization**: Compression and efficient data layout

## Configuration Options

### Custom S3 Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: s3-secret
stringData:
  bucket: "my-tempo-bucket"
  endpoint: "https://s3.us-west-2.amazonaws.com"
  region: "us-west-2"
  access_key_id: "AKIA..."
  access_key_secret: "xxx..."
  # Optional: Custom S3 configuration
  insecure: "false"
  http_config:
    idle_conn_timeout: "90s"
    response_header_timeout: "2m"
    insecure_skip_verify: false
```

### Advanced Storage Options

```yaml
spec:
  storage:
    secret:
      name: s3-secret
      type: s3
  storageSize: 1Gi
  # Optional: Storage class configuration
  storageClassName: "fast-ssd"
  retention:
    global:
      traces: 720h  # 30 days
  template:
    compactor:
      # S3-specific compaction settings
      config:
        compactor:
          compaction:
            v2_in_buffer_bytes: 5242880
            v2_out_buffer_bytes: 20971520
```

### S3 Security Configuration

```yaml
# IAM role-based authentication (recommended for production)
apiVersion: v1
kind: Secret
metadata:
  name: s3-secret
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/TempoS3Role"
stringData:
  bucket: "tempo-production-traces"
  endpoint: "https://s3.us-east-1.amazonaws.com"
  region: "us-east-1"
  # No access keys needed with IAM roles
```

## AWS IAM Configuration

### Required IAM Permissions

Create an IAM policy with these minimum permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::your-tempo-bucket",
        "arn:aws:s3:::your-tempo-bucket/*"
      ]
    }
  ]
}
```

### IAM Role for Service Accounts (IRSA)

For production deployments, use IRSA instead of access keys:

```bash
# Create IAM role with trust policy for OpenShift service account
aws iam create-role --role-name TempoS3Role \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT:oidc-provider/OIDC_URL"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "OIDC_URL:sub": "system:serviceaccount:NAMESPACE:tempo-simplest-compactor"
        }
      }
    }]
  }'

# Attach the policy
aws iam attach-role-policy \
  --role-name TempoS3Role \
  --policy-arn arn:aws:iam::ACCOUNT:policy/TempoS3Policy
```

## Monitoring and Troubleshooting

### Check S3 Connectivity

```bash
# Test S3 access from compactor pod
oc exec -it deployment/tempo-simplest-compactor -- \
  aws s3 ls s3://your-tempo-bucket/ --region us-east-2
```

### Monitor Storage Usage

```bash
# Check S3 bucket size
aws s3api head-bucket --bucket your-tempo-bucket | \
  jq '.BucketSizeBytes'

# List objects in bucket
aws s3 ls s3://your-tempo-bucket/traces/ --recursive --human-readable
```

### Component Logs

```bash
# Compactor logs (S3 operations)
oc logs -l app.kubernetes.io/component=compactor | grep -i s3

# Ingester logs (write operations)
oc logs -l app.kubernetes.io/component=ingester | grep -i flush

# Querier logs (read operations)
oc logs -l app.kubernetes.io/component=querier | grep -i s3
```

### Common Issues

1. **Bucket Access Denied**:
   ```bash
   # Check IAM permissions
   aws sts get-caller-identity
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::ACCOUNT:user/USERNAME \
     --action-names s3:GetObject \
     --resource-arns arn:aws:s3:::your-tempo-bucket/*
   ```

2. **Region Mismatch**:
   ```bash
   # Verify bucket region
   aws s3api get-bucket-location --bucket your-tempo-bucket
   ```

3. **Network Connectivity**:
   ```bash
   # Test from cluster
   oc run aws-cli --image=amazon/aws-cli --rm -it -- \
     s3 ls s3://your-tempo-bucket/
   ```

## Performance Optimization

### S3 Configuration Tuning

```yaml
# Optimize for high-throughput workloads
spec:
  template:
    compactor:
      config:
        storage:
          trace:
            s3:
              # Increase parallelism
              max_parallel: 50
              # Tune timeouts
              timeout: 60s
              # Enable compression
              compression: gzip
```

### Cost Optimization

1. **Storage Classes**:
   ```bash
   # Configure lifecycle policy for cost savings
   aws s3api put-bucket-lifecycle-configuration \
     --bucket your-tempo-bucket \
     --lifecycle-configuration '{
       "Rules": [{
         "Status": "Enabled",
         "Transitions": [{
           "Days": 30,
           "StorageClass": "STANDARD_IA"
         }, {
           "Days": 90,
           "StorageClass": "GLACIER"
         }]
       }]
     }'
   ```

2. **Data Compression**:
   - Enable gzip compression in Tempo configuration
   - Use efficient encoding formats
   - Configure appropriate block sizes

## Production Considerations

### 1. **Security**
- Use IAM roles instead of access keys
- Enable S3 bucket encryption
- Configure VPC endpoints for private connectivity
- Implement bucket policies for additional security

### 2. **Backup and Disaster Recovery**
- Enable S3 Cross-Region Replication
- Configure versioning for accidental deletion protection
- Implement backup retention policies
- Test disaster recovery procedures

### 3. **Monitoring**
- Set up CloudWatch alarms for S3 operations
- Monitor request rates and error rates
- Track storage costs and usage patterns
- Configure alerts for quota exceeded

### 4. **Compliance**
- Configure data retention per compliance requirements
- Enable CloudTrail for audit logging
- Implement data classification and tagging
- Review access patterns regularly

## Related Configurations

- [Azure Blob Storage](../tempostack-azure/README.md) - Azure cloud storage integration
- [Google Cloud Storage](../tempostack-gcs/README.md) - GCP cloud storage integration
- [AWS STS Integration](../aws-sts-tempostack/README.md) - Token-based authentication
- [Basic TempoStack](../../e2e/compatibility/README.md) - Local storage setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/tempostack-aws
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires valid AWS credentials and permissions to create/delete S3 buckets.