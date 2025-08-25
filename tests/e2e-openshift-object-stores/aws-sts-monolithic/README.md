# TempoMonolithic with AWS S3 using STS Authentication

This test validates TempoMonolithic deployment using AWS S3 storage with Security Token Service (STS) authentication. It demonstrates secure, keyless access to AWS S3 using OpenShift's OIDC provider and AWS IAM role federation for single-component Tempo deployments.

## Test Overview

### Purpose
- **AWS STS Integration**: Tests AWS Security Token Service authentication for TempoMonolithic
- **Keyless Authentication**: Validates secure S3 access without static AWS credentials
- **OIDC Federation**: Tests OpenShift's OIDC provider integration with AWS IAM
- **Monolithic Architecture**: Verifies STS authentication with single-component Tempo deployment

### Components
- **TempoMonolithic**: Single-component Tempo deployment with S3 storage
- **AWS S3 Bucket**: Cloud object storage for trace data persistence
- **AWS IAM Role**: Scoped permissions for TempoMonolithic service account
- **OIDC Provider**: OpenShift identity provider for AWS federation

## Authentication Flow

```
[TempoMonolithic ServiceAccount] 
        ↓ (OIDC Token)
[AWS STS AssumeRoleWithWebIdentity]
        ↓ (Temporary Credentials)
[S3 Bucket Access]
        ↓
[Trace Storage Operations]
```

## Deployment Steps

### 1. Create AWS Infrastructure
```bash
./aws-sts-s3-create.sh tm chainsaw-aws-mono
```

The [`aws-sts-s3-create.sh`](aws-sts-s3-create.sh) script performs:
- **S3 Bucket Creation**: Creates `tracing-{namespace}-{build-id}` bucket in `us-east-2`
- **IAM Role Setup**: Creates role with trust relationship for OpenShift OIDC provider
- **Trust Policy Configuration**: Allows specific TempoMonolithic service account to assume role
- **Policy Attachment**: Grants `AmazonS3FullAccess` permissions to the role
- **Secret Creation**: Creates Kubernetes secret with bucket, region, and role ARN

### 2. Deploy TempoMonolithic with STS
```bash
kubectl apply -f install-monolithic.yaml
```

Key configuration from [`install-monolithic.yaml`](install-monolithic.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: tm
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: aws-sts
        roleArn: ${ROLE_ARN}  # Injected by test setup
  jaegerui:
    enabled: true
    route:
      enabled: true
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f generate-traces.yaml
kubectl apply -f verify-traces.yaml
```

## Key Features Tested

### AWS STS Authentication
- ✅ AssumeRoleWithWebIdentity using OpenShift OIDC tokens
- ✅ Temporary credential generation and rotation
- ✅ Service account to IAM role mapping
- ✅ Scoped permissions for S3 bucket access

### TempoMonolithic Configuration
- ✅ Single-component deployment with S3 backend
- ✅ Jaeger UI with OpenShift Route access
- ✅ Direct role ARN configuration in spec
- ✅ STS credential injection and management

### Security Features
- ✅ No static AWS credentials stored in cluster
- ✅ Least-privilege IAM role permissions
- ✅ OIDC-based authentication with OpenShift identity
- ✅ Encrypted S3 bucket access via TLS

### Monolithic Architecture Benefits
- ✅ Simplified deployment model with single component
- ✅ Reduced resource overhead compared to distributed deployment
- ✅ Direct S3 integration without multiple service accounts
- ✅ Suitable for development and smaller production workloads

## IAM Trust Policy

The test creates a trust relationship allowing the TempoMonolithic service account:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::{account}:oidc-provider/{openshift-oidc}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "{oidc-provider}:sub": [
            "system:serviceaccount:{namespace}:tempo-{name}"
          ]
        }
      }
    }
  ]
}
```

## Environment Requirements

### OpenShift CI Integration
- **OPENSHIFT_BUILD_NAMESPACE**: Unique identifier for test isolation
- **CLUSTER_PROFILE_DIR**: Path to AWS credentials for setup
- **AWS_SHARED_CREDENTIALS_FILE**: AWS credentials for infrastructure setup

### AWS Prerequisites
- Valid AWS account with S3 and IAM permissions
- OpenShift cluster with OIDC provider configured
- IAM permissions to create roles and attach policies

## Cleanup Process

The test automatically cleans up AWS resources:
```bash
./aws-sts-s3-delete.sh tm chainsaw-aws-mono
```

This removes:
- S3 bucket and all contained objects
- IAM role and attached policies
- Kubernetes secrets and configurations

## Comparison with TempoStack STS

### TempoMonolithic Advantages
- **Simpler Setup**: Single service account vs. multiple service accounts
- **Resource Efficiency**: Lower memory and CPU overhead
- **Direct Configuration**: Role ARN specified directly in CR spec
- **Unified Component**: All functionality in single pod

### TempoStack Advantages
- **Scalability**: Better for high-volume trace workloads
- **Component Separation**: Independent scaling of ingestion and query
- **Fault Tolerance**: Component isolation reduces single points of failure
- **Production Ready**: Better suited for enterprise deployments

## Troubleshooting

### Common Issues

**IAM Role Assumption Failures**:
- Verify OIDC provider is correctly configured in AWS
- Check trust policy includes correct service account name
- Ensure OpenShift cluster identity matches OIDC provider

**S3 Access Denied**:
- Confirm IAM role has appropriate S3 permissions
- Verify bucket policy allows access from the role
- Check regional restrictions and bucket location

**Trace Ingestion Problems**:
- Verify TempoMonolithic pod has assumed the role successfully
- Check pod logs for authentication errors
- Ensure S3 endpoint is accessible from cluster network

This test demonstrates how TempoMonolithic can securely integrate with AWS S3 using modern authentication patterns, eliminating the need for long-lived credentials while maintaining simplicity of deployment.

