# TempoStack with AWS S3 using STS and Cloud Credential Operator (CCO)

This test validates TempoStack deployment using AWS S3 storage with Security Token Service (STS) authentication managed by OpenShift's Cloud Credential Operator (CCO). It demonstrates the integration between OpenShift CCO and AWS IAM for secure, temporary credential management.

## Test Overview

### Purpose
- **AWS STS Integration**: Tests AWS Security Token Service authentication with OpenShift workloads
- **Cloud Credential Operator**: Validates CCO's ability to manage AWS credentials for TempoStack
- **Temporary Credentials**: Ensures secure access to S3 using temporary, scoped credentials
- **OIDC Federation**: Tests OpenShift's OIDC provider integration with AWS IAM

### Components
- **TempoStack**: Distributed Tempo deployment configured for AWS S3 storage
- **AWS S3 Bucket**: Cloud object storage for trace data persistence
- **AWS IAM Role**: Scoped permissions for TempoStack service accounts
- **OpenShift CCO**: Manages AWS credential lifecycle and injection
- **OIDC Provider**: OpenShift identity provider for AWS federation

## Authentication Flow

```
[TempoStack ServiceAccount] 
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
./aws-sts-s3-create.sh tmcco chainsaw-awscco-tempo
```

The [`aws-sts-s3-create.sh`](aws-sts-s3-create.sh) script performs:
- **S3 Bucket Creation**: Creates `tracing-{namespace}-{build-id}` bucket in `us-east-2`
- **IAM Role Setup**: Creates role with trust relationship for OpenShift OIDC provider
- **Trust Policy Configuration**: Allows specific TempoStack service accounts to assume role
- **Policy Attachment**: Grants `AmazonS3FullAccess` permissions to the role
- **Secret Creation**: Creates Kubernetes secret with bucket, region, and role ARN
- **Operator Configuration**: Patches Tempo Operator subscription with role ARN

### 2. Deploy TempoStack with STS
```bash
kubectl apply -f install-tempostack.yaml
```

Key configuration from [`install-tempostack.yaml`](install-tempostack.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: tmcco
spec:
  storage:
    secret:
      name: aws-sts
      type: s3
      credentialMode: token-cco  # CCO manages credentials
  storageSize: 10Gi
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: route
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

### Cloud Credential Operator Integration
- ✅ CCO credential injection into TempoStack pods
- ✅ Automatic credential refresh and lifecycle management
- ✅ Secure credential storage without long-lived keys
- ✅ Operator subscription patching with role ARN

### TempoStack Configuration
- ✅ Distributed deployment with S3 backend
- ✅ Jaeger Query UI with OpenShift Route
- ✅ 10Gi storage allocation for trace data
- ✅ Resource limits: 4Gi memory, 2000m CPU

### Security Features
- ✅ No static AWS credentials stored in cluster
- ✅ Least-privilege IAM role permissions
- ✅ OIDC-based authentication with OpenShift identity
- ✅ Encrypted S3 bucket access via TLS

## IAM Trust Policy

The test creates a trust relationship allowing specific service accounts:
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
            "system:serviceaccount:{namespace}:tempo-{name}",
            "system:serviceaccount:{namespace}:tempo-{name}-query-frontend"
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
- Cloud Credential Operator enabled and functional

## Cleanup Process

The test automatically cleans up AWS resources:
```bash
./aws-sts-s3-delete.sh tmcco chainsaw-awscco-tempo
```

This removes:
- S3 bucket and all contained objects
- IAM role and attached policies
- Kubernetes secrets and configurations

## Troubleshooting

### Common Issues

**IAM Role Assumption Failures**:
- Verify OIDC provider is correctly configured in AWS
- Check trust policy includes correct service account names
- Ensure OpenShift cluster identity matches OIDC provider

**S3 Access Denied**:
- Confirm IAM role has appropriate S3 permissions
- Verify bucket policy allows access from the role
- Check regional restrictions and bucket location

**CCO Integration Problems**:
- Ensure Cloud Credential Operator is running and healthy
- Verify operator subscription patch was successful
- Check CredentialsRequest objects are created properly

This test demonstrates enterprise-grade security integration between OpenShift and AWS, ensuring TempoStack can securely access cloud storage without exposing long-lived credentials within the cluster.

