# TempoMonolithic with GCP Cloud Storage using Workload Identity Federation

This test validates TempoMonolithic deployment using Google Cloud Storage (GCS) with Workload Identity Federation (WIF). It demonstrates secure, keyless access to GCP Cloud Storage using OpenShift's OIDC provider and GCP IAM service account impersonation for single-component Tempo deployments.

## Test Overview

### Purpose
- **GCP Workload Identity Federation**: Tests secure authentication to GCS without service account keys
- **Keyless Authentication**: Validates secure GCS access without static credentials
- **OIDC Federation**: Tests OpenShift's OIDC provider integration with GCP IAM
- **Monolithic Architecture**: Verifies WIF authentication with single-component Tempo deployment

### Components
- **TempoMonolithic**: Single-component Tempo deployment with GCS storage
- **GCP Cloud Storage**: Google Cloud object storage for trace data persistence
- **GCP Service Account**: Scoped permissions for TempoMonolithic workloads
- **Workload Identity Pool**: GCP federation configuration for OpenShift OIDC
- **OIDC Provider**: OpenShift identity provider for GCP federation

## Authentication Flow

```
[TempoMonolithic ServiceAccount] 
        ↓ (OIDC Token)
[GCP Workload Identity Federation]
        ↓ (Service Account Impersonation)
[GCS Bucket Access]
        ↓
[Trace Storage Operations]
```

## Deployment Steps

### 1. Create GCP Infrastructure
```bash
./gcp-wif-create.sh
```

The [`gcp-wif-create.sh`](gcp-wif-create.sh) script performs:
- **Service Account Creation**: Creates GCP service account with `Storage Object Admin` role
- **Workload Identity Setup**: Configures service account impersonation for OpenShift
- **GCS Bucket Creation**: Creates storage bucket for trace data
- **Identity Pool Configuration**: Sets up workload identity federation
- **Secret Creation**: Creates Kubernetes secret with GCS configuration

### 2. Deploy TempoMonolithic with WIF
```bash
kubectl apply -f install-monolithic.yaml
```

Key configuration from [`install-monolithic.yaml`](install-monolithic.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: gcpwifmn
spec:
  storage:
    traces:
      backend: gcs
      gcs:
        secret: gcs-secret  # Contains WIF configuration
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

### GCP Workload Identity Federation
- ✅ Service account impersonation using OpenShift OIDC tokens
- ✅ Workload identity pool configuration and binding
- ✅ Principal mapping from Kubernetes service accounts to GCP
- ✅ Scoped permissions for GCS bucket access

### TempoMonolithic Configuration
- ✅ Single-component deployment with GCS backend
- ✅ Jaeger UI with OpenShift Route access
- ✅ Secure GCS integration without service account keys
- ✅ WIF credential injection and management

### Security Features
- ✅ No static GCP credentials stored in cluster
- ✅ Least-privilege service account permissions
- ✅ OIDC-based authentication with OpenShift identity
- ✅ Encrypted GCS bucket access via TLS

### Monolithic Architecture Benefits
- ✅ Simplified deployment model with single component
- ✅ Reduced resource overhead compared to distributed deployment
- ✅ Direct GCS integration without multiple service accounts
- ✅ Suitable for development and smaller production workloads

## GCP Service Account Configuration

The test creates IAM bindings for workload identity:
```bash
# Allow Kubernetes SA to impersonate GCP SA
gcloud iam service-accounts add-iam-policy-binding $SERVICE_ACCOUNT_EMAIL \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${NAMESPACE}:tempo-${NAME}"
```

## Environment Requirements

### GCP Prerequisites
- Valid GCP project with billing enabled
- Google Cloud Storage API enabled
- Identity and Access Management (IAM) API enabled
- Workload Identity Federation enabled
- Appropriate IAM permissions for service account creation

### OpenShift Prerequisites
- OpenShift cluster with OIDC provider configured
- Valid OIDC issuer URL accessible from GCP
- Proper network connectivity to GCP services

## Cleanup Process

The test automatically cleans up GCP resources:
```bash
./gcp-wif-delete.sh
```

This removes:
- GCS bucket and all contained objects
- GCP service account and IAM bindings
- Workload identity pool configurations
- Kubernetes secrets and configurations

## Comparison with AWS STS

### GCP WIF Advantages
- **Native Integration**: Deep integration with GCP services
- **Fine-grained Control**: Granular permission scoping
- **Service Account Impersonation**: Flexible identity delegation
- **Audit Trail**: Comprehensive GCP audit logging

### Workload Identity vs. Service Account Keys
- **Security**: No long-lived credentials in cluster
- **Rotation**: Automatic credential lifecycle management
- **Compliance**: Meets enterprise security requirements
- **Monitoring**: Better visibility into access patterns

## Troubleshooting

### Common Issues

**Service Account Impersonation Failures**:
- Verify workload identity pool is correctly configured
- Check principal mapping includes correct Kubernetes service account
- Ensure OpenShift OIDC issuer is accessible from GCP

**GCS Access Denied**:
- Confirm GCP service account has appropriate Storage permissions
- Verify bucket policy allows access from the service account
- Check regional restrictions and bucket location

**OIDC Token Issues**:
- Validate OpenShift OIDC provider configuration
- Ensure token audience matches GCP expectations
- Check token expiration and renewal mechanisms

**Network Connectivity**:
- Verify cluster can reach GCP APIs
- Check firewall rules and network policies
- Ensure DNS resolution for GCP endpoints

This test demonstrates how TempoMonolithic can securely integrate with Google Cloud Storage using modern authentication patterns, eliminating the need for service account keys while maintaining simplicity of deployment.

