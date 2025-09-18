# TempoStack with AWS STS (Security Token Service) Authentication

This configuration blueprint demonstrates how to deploy TempoStack with AWS STS authentication using OpenShift's Workload Identity Federation. This setup showcases enterprise-grade cloud security with IAM roles, OIDC federation, and temporary credential management for secure access to AWS S3 without storing long-term credentials.

## Overview

This test validates a cloud-native security stack featuring:
- **AWS STS Integration**: Short-lived token authentication via OpenShift OIDC provider
- **Workload Identity Federation**: OpenShift service accounts federated with AWS IAM roles
- **Temporary Credentials**: No long-term AWS credentials stored in cluster
- **Automated IAM Management**: Dynamic role and policy creation for TempoStack components
- **Enterprise Security**: Least-privilege access with automatic credential rotation

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│   TempoStack    │───▶│   AWS STS Service    │───▶│    AWS S3       │
│ ┌─────────────┐ │    │ ┌─────────────────┐  │    │ ┌─────────────┐ │
│ │ Service     │ │    │ │ Assume Role     │  │    │ │ S3 Bucket   │ │
│ │ Accounts    │ │    │ │ With Web        │  │    │ │ Access      │ │
│ │ + Tokens    │ │    │ │ Identity        │  │    │ └─────────────┘ │
│ └─────────────┘ │    │ └─────────────────┘  │    └─────────────────┘
└─────────────────┘    └──────────────────────┘
          │                        ▲
          │ ┌──────────────────────┴─────────────────────┐
          │ │         OpenShift OIDC Provider            │
          └▶│ ┌─────────────────┐ ┌─────────────────┐    │
            │ │ Service Account │ │ OIDC Issuer     │    │
            │ │ JWT Tokens      │ │ Federation      │    │
            │ └─────────────────┘ └─────────────────┘    │
            └────────────────────────────────────────────┘
                                   ▲
                          ┌──────────────────────┐
                          │    AWS IAM Role      │
                          │ - Trust Relationship │
                          │ - S3 Policy          │
                          │ - Federated Identity │
                          └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+) with OIDC federation capabilities
- AWS account with IAM permissions for role and policy creation
- Tempo Operator installed
- AWS CLI configured (for CI environments)
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Create AWS Infrastructure and IAM Resources

Run the automated setup script to create S3 bucket, IAM role, and trust relationship:

```bash
./aws-sts-s3-create.sh tmstack chainsaw-awssts-tempo
```

**Script Functionality from [`aws-sts-s3-create.sh`](./aws-sts-s3-create.sh)**:

#### AWS S3 Bucket Creation
- Creates region-specific S3 bucket with unique naming
- Configures bucket for TempoStack trace storage
- Uses naming pattern: `tracing-{namespace}-{build-namespace}`

#### OIDC Provider Integration
```bash
oidc_provider=$(oc get authentication cluster -o json | jq -r '.spec.serviceAccountIssuer' | sed 's~http[s]*://~~g')
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
```

#### IAM Trust Relationship
Creates trust policy allowing OpenShift service accounts to assume AWS role:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:aws:iam::{account}:oidc-provider/{oidc-provider}"
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
  }]
}
```

#### Secret Creation
Creates Kubernetes secret with STS configuration:
- `bucket`: S3 bucket name for trace storage
- `region`: AWS region for S3 operations  
- `role_arn`: IAM role ARN for STS assume role operations

### Step 2: Deploy TempoStack with STS Authentication

Apply TempoStack configuration for STS-based S3 access:

```bash
oc apply -f install-tempostack.yaml
```

**Key STS Configuration from [`install-tempostack.yaml`](./install-tempostack.yaml)**:
- **Storage Secret**: References `aws-sts` secret containing role ARN and bucket info
- **S3 Storage Type**: Configured for S3-compatible storage with STS authentication
- **Route Integration**: OpenShift route for external Jaeger UI access
- **Resource Allocation**: 4Gi memory and 2000m CPU for production workloads

### Step 3: Verify TempoStack Readiness

Wait for TempoStack to initialize with STS authentication:

```bash
oc get --namespace chainsaw-awssts-tempo tempo tmstack -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

### Step 4: Generate Sample Traces

Create traces to validate STS-authenticated S3 storage:

```bash
oc apply -f generate-traces.yaml
```

**Reference**: [`generate-traces.yaml`](./generate-traces.yaml)

### Step 5: Verify Traces with STS Storage

Validate that traces are properly stored in S3 via STS authentication:

```bash
oc apply -f verify-traces.yaml
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

### Step 6: Cleanup AWS Resources

After testing, clean up AWS infrastructure:

```bash
./aws-sts-s3-delete.sh tmstack chainsaw-awssts-tempo
```

**Reference**: [`aws-sts-s3-delete.sh`](./aws-sts-s3-delete.sh)

## Key Features Demonstrated

### 1. **AWS STS Authentication**
- **Temporary Credentials**: Short-lived tokens instead of permanent access keys
- **Automatic Rotation**: Credentials automatically refreshed by AWS STS
- **Zero Credential Storage**: No long-term secrets stored in Kubernetes
- **Audit Trail**: Complete AWS CloudTrail logging of all STS operations

### 2. **OpenShift OIDC Federation**
- **Service Account Integration**: OpenShift service accounts mapped to AWS IAM roles
- **JWT Token Exchange**: Service account tokens exchanged for AWS STS tokens
- **Federated Identity**: OpenShift cluster acts as trusted OIDC identity provider
- **Seamless Authentication**: No manual credential management required

### 3. **Enterprise Security Model**
- **Least Privilege Access**: IAM roles with minimal required permissions
- **Trust Boundaries**: Specific service accounts authorized for specific roles
- **Conditional Access**: OIDC subject claims enforce access restrictions
- **Infrastructure as Code**: Automated IAM resource creation and cleanup

### 4. **Production Cloud Integration**
- **Multi-Region Support**: Configurable AWS regions for data locality
- **Scalable Architecture**: Supports multiple TempoStack deployments
- **Cost Optimization**: No data egress charges for same-region operations
- **Compliance Ready**: Meets enterprise cloud security requirements

## STS Authentication Flow

### Token Exchange Process

1. **Service Account Token**: TempoStack pod receives OpenShift service account JWT
2. **STS Assume Role**: Pod exchanges JWT for temporary AWS credentials via STS
3. **Credential Validation**: AWS validates OIDC token against trust relationship
4. **Temporary Access**: AWS returns temporary credentials with limited scope
5. **S3 Operations**: Pod uses temporary credentials for S3 bucket operations
6. **Automatic Refresh**: Credentials automatically renewed before expiration

### Security Benefits

- **No Credential Leakage**: No long-term credentials can be compromised
- **Automatic Rotation**: Credentials expire and refresh automatically
- **Audit Compliance**: All access logged and traceable
- **Principle of Least Privilege**: Access limited to specific resources and actions

## AWS IAM Configuration Details

### Trust Relationship Components

```json
{
  "Principal": {
    "Federated": "arn:aws:iam::{account}:oidc-provider/{openshift-oidc-issuer}"
  },
  "Condition": {
    "StringEquals": {
      "{oidc-issuer}:sub": [
        "system:serviceaccount:{namespace}:tempo-{name}",
        "system:serviceaccount:{namespace}:tempo-{name}-query-frontend"
      ]
    }
  }
}
```

### Required IAM Permissions

- **S3 Bucket Access**: Read/write permissions for trace storage
- **STS Operations**: Assume role permissions for credential exchange
- **KMS Operations**: Encryption key access if S3 encryption enabled

### Service Account Mapping

Each TempoStack component service account maps to specific IAM role:
- `tempo-{name}`: Primary service account for compactor and ingester
- `tempo-{name}-query-frontend`: Query frontend service account

## Monitoring and Troubleshooting

### Verify STS Configuration

```bash
# Check secret configuration
oc get secret aws-sts -o yaml

# Verify service account annotations
oc get serviceaccount tempo-tmstack -o yaml

# Check IAM role ARN
oc get secret aws-sts -o jsonpath='{.data.role_arn}' | base64 -d
```

### Validate AWS Resources

```bash
# List IAM roles
aws iam list-roles --query 'Roles[?starts_with(RoleName, `tracing-`)]'

# Check S3 bucket
aws s3 ls tracing-chainsaw-awssts-tempo-cioptmstack

# Verify OIDC provider
aws iam list-open-id-connect-providers
```

### Debug STS Authentication

```bash
# Check component logs for STS operations
oc logs -l app.kubernetes.io/component=compactor | grep -i sts

# Monitor S3 access patterns
oc logs -l app.kubernetes.io/component=ingester | grep -i s3

# Verify credential refresh
oc logs -l app.kubernetes.io/component=querier | grep -i "credential\|token"
```

### Common STS Issues

1. **Trust Relationship Mismatch**:
   ```bash
   # Verify OIDC issuer URL
   oc get authentication cluster -o jsonpath='{.spec.serviceAccountIssuer}'
   
   # Check service account subject format
   oc get serviceaccount tempo-tmstack -o jsonpath='{.metadata.name}'
   ```

2. **IAM Permission Errors**:
   ```bash
   # Check attached policies
   aws iam list-attached-role-policies --role-name tracing-chainsaw-awssts-tempo-cioptmstack
   
   # Verify S3 permissions
   aws s3api get-bucket-policy --bucket tracing-chainsaw-awssts-tempo-cioptmstack
   ```

3. **Token Expiration Issues**:
   ```bash
   # Monitor token refresh cycles
   oc logs -l app.kubernetes.io/component=distributor | grep -i "token refresh"
   ```

## Production Considerations

### 1. **Security Hardening**
- Implement least-privilege IAM policies
- Use S3 bucket policies for additional access control
- Enable CloudTrail for comprehensive audit logging
- Regular rotation of OIDC provider trust relationships

### 2. **High Availability**
- Deploy across multiple AWS availability zones
- Configure S3 Cross-Region Replication for disaster recovery
- Monitor STS service availability and quota limits
- Implement circuit breakers for STS failures

### 3. **Cost Optimization**
- Use S3 lifecycle policies for cost-effective storage
- Monitor STS API call volume and costs
- Optimize credential refresh intervals
- Consider S3 Intelligent Tiering for variable access patterns

### 4. **Operational Excellence**
- Automate IAM resource lifecycle management
- Monitor STS token usage and expiration patterns
- Implement automated testing of STS authentication flow
- Document troubleshooting procedures for STS issues

## Related Configurations

- [Basic AWS S3](../tempostack-aws/README.md) - AWS S3 with traditional credentials
- [Azure Workload Identity](../azure-wif-tempostack/README.md) - Azure equivalent of STS
- [GCP Workload Identity](../gcp-wif-tempostack/README.md) - GCP workload identity federation
- [Basic TempoStack](../../e2e/compatibility/README.md) - Local storage baseline

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/aws-sts-tempostack
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test is designed for OpenShift CI environments with automatic AWS credential management and requires proper OIDC federation setup.