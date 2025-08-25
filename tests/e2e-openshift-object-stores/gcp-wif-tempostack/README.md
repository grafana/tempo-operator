# TempoStack with GCP Cloud Storage using Workload Identity Federation

This test validates TempoStack deployment using Google Cloud Storage (GCS) with Workload Identity Federation (WIF). It demonstrates secure, keyless access to GCP Cloud Storage using OpenShift's OIDC provider and GCP IAM service account impersonation for distributed Tempo deployments.

## Test Overview

### Purpose
- **GCP Workload Identity Federation**: Tests secure authentication to GCS without service account keys
- **Distributed Architecture**: Validates WIF with multi-component TempoStack deployment
- **OIDC Federation**: Tests OpenShift's OIDC provider integration with GCP IAM
- **Enterprise Security**: Demonstrates keyless authentication for production workloads

### Components
- **TempoStack**: Distributed Tempo deployment with GCS storage
- **GCP Cloud Storage**: Google Cloud object storage for trace data persistence
- **GCP Service Account**: Scoped permissions for TempoStack workloads
- **Workload Identity Pool**: GCP federation configuration for OpenShift OIDC
- **OIDC Provider**: OpenShift identity provider for GCP federation

## Authentication Flow

```
[TempoStack ServiceAccounts] 
        ↓ (OIDC Tokens)
[GCP Workload Identity Federation]
        ↓ (Service Account Impersonation)
[GCS Bucket Access]
        ↓
[Distributed Trace Storage Operations]
```

## Deployment Steps

### 1. Create GCP Infrastructure
```bash
./gcp-wif-create.sh
```

The [`gcp-wif-create.sh`](gcp-wif-create.sh) script performs:
- **Service Account Creation**: Creates GCP service account with `Storage Object Admin` role
- **Workload Identity Setup**: Configures service account impersonation for multiple TempoStack components
- **GCS Bucket Creation**: Creates storage bucket for trace data
- **Identity Pool Configuration**: Sets up workload identity federation
- **Secret Creation**: Creates Kubernetes secret with GCS configuration

### 2. Deploy TempoStack with WIF
```bash
kubectl apply -f install-tempostack.yaml
```

Key configuration from [`install-tempostack.yaml`](install-tempostack.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: gcpwifts
spec:
  storage:
    secret:
      name: gcs-secret
      type: gcs
  storageSize: 10Gi
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
          type: route
```

### 3. Generate and Verify Traces
```bash
kubectl apply -f generate-traces.yaml
kubectl apply -f verify-traces.yaml
```

## Key Features Tested

### GCP Workload Identity Federation
- ✅ Service account impersonation using OpenShift OIDC tokens
- ✅ Multi-component workload identity pool configuration
- ✅ Principal mapping from multiple Kubernetes service accounts to GCP
- ✅ Scoped permissions for GCS bucket access across components

### TempoStack Configuration
- ✅ Distributed deployment with GCS backend
- ✅ Jaeger Query UI with OpenShift Route
- ✅ 10Gi storage allocation for trace data
- ✅ Resource limits: 4Gi memory, 2000m CPU
- ✅ Multiple service accounts for different components

### Security Features
- ✅ No static GCP credentials stored in cluster
- ✅ Least-privilege service account permissions
- ✅ OIDC-based authentication with OpenShift identity
- ✅ Encrypted GCS bucket access via TLS
- ✅ Component-level security isolation

### Distributed Architecture Benefits
- ✅ Independent component scaling and management
- ✅ Fault tolerance through component separation
- ✅ Better resource utilization for high-volume workloads
- ✅ Production-ready deployment model

## GCP Service Account Configuration

The test creates IAM bindings for multiple TempoStack service accounts:
```bash
# Allow multiple Kubernetes SAs to impersonate GCP SA
for component in distributor ingester querier query-frontend compactor; do
  gcloud iam service-accounts add-iam-policy-binding $SERVICE_ACCOUNT_EMAIL \
    --role="roles/iam.workloadIdentityUser" \
    --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${NAMESPACE}:tempo-${NAME}-${component}"
done
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
- Sufficient cluster resources for distributed deployment
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

## Comparison with TempoMonolithic WIF

### TempoStack Advantages
- **Scalability**: Better for high-volume trace workloads
- **Component Separation**: Independent scaling of ingestion and query
- **Fault Tolerance**: Component isolation reduces single points of failure
- **Production Ready**: Better suited for enterprise deployments

### TempoMonolithic Advantages
- **Simpler Setup**: Single service account vs. multiple service accounts
- **Resource Efficiency**: Lower memory and CPU overhead
- **Unified Security**: Single WIF configuration per deployment
- **Development Friendly**: Easier to deploy and manage for testing

## Component Service Account Mapping

Each TempoStack component uses its own service account for GCS access:
- `tempo-{name}-distributor`: Handles trace ingestion
- `tempo-{name}-ingester`: Processes and stores traces
- `tempo-{name}-querier`: Retrieves traces for queries
- `tempo-{name}-query-frontend`: Handles query routing
- `tempo-{name}-compactor`: Manages trace data compaction

## Troubleshooting

### Common Issues

**Service Account Impersonation Failures**:
- Verify workload identity pool covers all TempoStack service accounts
- Check principal mappings include all component service accounts
- Ensure OpenShift OIDC issuer is accessible from GCP

**GCS Access Denied**:
- Confirm GCP service account has appropriate Storage permissions
- Verify bucket policy allows access from the service account
- Check that all TempoStack components can impersonate the service account

**Component Startup Failures**:
- Validate WIF configuration for each service account
- Check resource limits don't prevent component startup
- Ensure all required GCP APIs are enabled

**Network Connectivity**:
- Verify cluster can reach GCP APIs from all nodes
- Check firewall rules and network policies
- Ensure DNS resolution for GCP endpoints

This test demonstrates how TempoStack can securely integrate with Google Cloud Storage using modern authentication patterns, providing a production-ready solution for enterprise observability workloads without the security risks of long-lived credentials.

