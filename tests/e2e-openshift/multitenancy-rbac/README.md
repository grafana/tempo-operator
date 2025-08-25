# Multi-Tenancy with OpenShift RBAC Integration

This test demonstrates how to configure TempoStack with multi-tenancy support using OpenShift's Role-Based Access Control (RBAC) system. This enables secure isolation of traces between different teams, projects, or environments while leveraging OpenShift's native authentication and authorization mechanisms.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Namespace 1   │    │   Gateway with   │    │   Tenant: dev   │
│  (rbac-sa-1)    │───▶│   RBAC Enabled   │───▶│   ID: 1610...fa │
│                 │    │                  │    │                 │
└─────────────────┘    │                  │    └─────────────────┘
                       │                  │
┌─────────────────┐    │   TempoStack     │    ┌─────────────────┐
│   Namespace 2   │───▶│   Gateway        │───▶│   Tenant: prod  │
│  (rbac-sa-2)    │    │                  │    │   ID: 1610...fb │
└─────────────────┘    │                  │    └─────────────────┘
                       │                  │
┌─────────────────┐    │                  │    ┌─────────────────┐
│  Cluster Admin  │───▶│                  │───▶│   All Tenants   │
│  (kubeadmin)    │    └──────────────────┘    │   Access        │
└─────────────────┘                            └─────────────────┘
```

## Multi-Tenancy Features

### OpenShift RBAC Integration
- **Namespace Isolation**: Traces automatically isolated by OpenShift namespace
- **Service Account Mapping**: Each namespace has dedicated service accounts
- **Role-Based Access**: Uses OpenShift roles for trace access control
- **Authentication**: Leverages OpenShift's built-in authentication system

### Tenant Configuration
- **Dev Tenant**: `tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"`
- **Prod Tenant**: `tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"`
- **Gateway Mode**: OpenShift mode with RBAC enforcement
- **Authentication Method**: OpenShift native authentication

## Test Components

### TempoStack Multi-Tenant Configuration
- **File**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)
- **Tenants Mode**: `openshift` - integrates with OpenShift RBAC
- **Gateway**: Enabled with RBAC enforcement
- **Authentication**: Defines dev and prod tenants with unique IDs
- **RBAC**: Custom ClusterRole and ClusterRoleBinding for trace access

### Service Account and Namespace Setup
- **File**: [`create-SAs-with-namespace-access.yaml`](./create-SAs-with-namespace-access.yaml)
- **Namespaces**: Creates `chainsaw-test-rbac-1` and `chainsaw-test-rbac-2`
- **Service Accounts**: Dedicated accounts for each namespace
- **Role Bindings**: Namespace-admin access for service accounts
- **Cluster Admin**: Separate cluster-admin service account for comparison

### OpenTelemetry Collector Configuration
- **File**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)
- **Multi-Tenant**: Configured to work with TempoStack gateway
- **Authentication**: Uses service account tokens for tenant identification

## Quick Start

### Prerequisites
- OpenShift cluster with Tempo Operator
- OpenTelemetry Operator installed
- MinIO or S3-compatible storage
- Cluster administrator access

### Step-by-Step Deployment

1. **Install Storage Backend**
   ```bash
   # Deploy MinIO for multi-tenant trace storage
   kubectl apply -f 00-install-storage.yaml
   kubectl wait --for=condition=ready pod -l app=minio -n chainsaw-rbac --timeout=300s
   ```

2. **Deploy Multi-Tenant TempoStack**
   ```bash
   # Create TempoStack with OpenShift RBAC integration
   kubectl apply -f 01-install-tempo.yaml
   kubectl wait --for=condition=ready tempostack simplst -n chainsaw-rbac --timeout=300s
   ```

3. **Deploy OpenTelemetry Collector**
   ```bash
   # Create collector for multi-tenant trace ingestion
   kubectl apply -f 02-install-otelcol.yaml
   kubectl wait --for=condition=ready opentelemetrycollector opentelemetry -n chainsaw-rbac --timeout=300s
   ```

4. **Create Service Accounts and Namespaces**
   ```bash
   # Set up isolated namespaces with dedicated service accounts
   kubectl apply -f create-SAs-with-namespace-access.yaml
   kubectl wait --for=condition=ready project chainsaw-test-rbac-1 --timeout=60s
   kubectl wait --for=condition=ready project chainsaw-test-rbac-2 --timeout=60s
   ```

5. **Generate Tenant-Specific Traces**
   ```bash
   # Generate traces from namespace 1
   kubectl apply -f tempo-rbac-sa-1-traces-gen.yaml
   
   # Generate traces from namespace 2
   kubectl apply -f tempo-rbac-sa-2-traces-gen.yaml
   ```

6. **Verify RBAC Isolation**
   ```bash
   # Verify service account can only access its own traces
   kubectl apply -f tempo-rbac-sa-1-traces-verify.yaml
   
   # Verify cluster admin can access all traces
   kubectl apply -f kubeadmin-traces-verify.yaml
   ```

## RBAC Configuration Details

### Tenant Authentication
```yaml
tenants:
  mode: openshift
  authentication:
    - tenantName: dev
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
    - tenantName: prod
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
```

### Gateway with RBAC
```yaml
template:
  gateway:
    enabled: true
    rbac:
      enabled: true  # Enforces OpenShift RBAC for trace access
```

### Custom ClusterRole for Trace Access
```yaml
rules:
  - apiGroups: ['tempo.grafana.com']
    resources: [dev]  # Tenant-specific resource access
    resourceNames: [traces]
    verbs: ['get']
```

## Testing Procedure

The complete test is defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml) and follows these steps:

1. **Storage Setup**: Deploy MinIO object storage
2. **Multi-Tenant TempoStack**: Create TempoStack with OpenShift RBAC mode
3. **Collector Deployment**: Deploy OpenTelemetry Collector
4. **Namespace Creation**: Create isolated namespaces with service accounts
5. **Trace Generation**: Generate traces from different namespaces
6. **RBAC Verification**: Verify that users can only access their own traces
7. **Admin Verification**: Confirm cluster admins can access all traces

## Multi-Tenancy Validation

### Service Account Permissions
```bash
# Check service account tokens and permissions
kubectl get serviceaccount tempo-rbac-sa-1 -n chainsaw-test-rbac-1 -o yaml
kubectl get rolebinding -n chainsaw-test-rbac-1

# Verify cluster admin permissions
kubectl get clusterrolebinding tempo-rbac-cluster-admin-binding -o yaml
```

### Tenant Isolation Testing
```bash
# Test trace access with different service accounts
kubectl auth can-i get traces --as=system:serviceaccount:chainsaw-test-rbac-1:tempo-rbac-sa-1
kubectl auth can-i get traces --as=system:serviceaccount:chainsaw-test-rbac-2:tempo-rbac-sa-2

# Verify gateway RBAC enforcement
kubectl get tempostack simplst -n chainsaw-rbac -o jsonpath='{.spec.template.gateway.rbac}'
```

### Trace Query Access
```bash
# Query traces as specific service account (should only see own traces)
kubectl get route -n chainsaw-rbac
curl -H "Authorization: Bearer $(kubectl get secret -n chainsaw-test-rbac-1 -o jsonpath='{.items[0].data.token}' | base64 -d)" \
     https://tempo-gateway-route/api/traces/v1/dev/api/search
```

## Production Considerations

### Security Best Practices
- **Principle of Least Privilege**: Service accounts only have access to their tenant's data
- **Network Policies**: Implement network isolation between tenants
- **Resource Quotas**: Set limits on trace ingestion per tenant
- **Audit Logging**: Enable OpenShift audit logging for trace access

### Scalability Planning
- **Tenant Limits**: Plan for maximum number of tenants
- **Resource Allocation**: Size TempoStack components for multi-tenant load
- **Storage Partitioning**: Consider tenant-specific storage partitions
- **Gateway Scaling**: Scale gateway replicas based on tenant count

### Monitoring and Alerting
- **Per-Tenant Metrics**: Monitor trace volume per tenant
- **RBAC Violations**: Alert on unauthorized access attempts
- **Gateway Performance**: Monitor authentication and authorization latency
- **Storage Usage**: Track storage consumption per tenant

## Troubleshooting

### Common RBAC Issues

1. **Access Denied Errors**
   ```bash
   # Check user/service account permissions
   kubectl auth can-i get traces --as=system:serviceaccount:chainsaw-test-rbac-1:tempo-rbac-sa-1
   
   # Verify ClusterRole and bindings
   kubectl get clusterrole tempostack-traces-reader-rbac -o yaml
   kubectl get clusterrolebinding tempostack-traces-reader-rbac -o yaml
   ```

2. **Cross-Tenant Access Issues**
   ```bash
   # Verify tenant configuration
   kubectl get tempostack simplst -n chainsaw-rbac -o jsonpath='{.spec.tenants}'
   
   # Check gateway logs for RBAC denials
   kubectl logs -n chainsaw-rbac -l app.kubernetes.io/component=gateway
   ```

3. **Service Account Token Problems**
   ```bash
   # Check service account secrets
   kubectl get serviceaccount tempo-rbac-sa-1 -n chainsaw-test-rbac-1 -o yaml
   
   # Verify token mounting
   kubectl describe pod -n chainsaw-test-rbac-1 -l app=trace-generator
   ```

### Debug Commands
```bash
# Check TempoStack multi-tenancy status
kubectl get tempostack simplst -n chainsaw-rbac -o jsonpath='{.status.conditions}'

# Verify gateway configuration
kubectl get service -n chainsaw-rbac -l app.kubernetes.io/component=gateway

# Check tenant authentication
kubectl get configmap -n chainsaw-rbac -l app.kubernetes.io/component=gateway

# Test trace ingestion per tenant
kubectl port-forward -n chainsaw-rbac svc/tempo-simplst-gateway 8080:8080
curl -H "X-Scope-OrgID: 1610b0c3-c509-4592-a256-a1871353dbfa" \
     http://localhost:8080/api/traces/v1/dev/api/search
```

## Comparison with Static Multi-Tenancy

| Feature | OpenShift RBAC | Static Multi-Tenancy |
|---------|----------------|---------------------|
| Authentication | OpenShift native | External OIDC provider |
| Authorization | OpenShift RBAC | Static tenant mapping |
| User Management | OpenShift Users/Groups | External identity provider |
| Namespace Integration | Native integration | Manual configuration |
| Complexity | Lower (built-in) | Higher (external deps) |
| Flexibility | OpenShift-specific | Provider-agnostic |

## Related Resources

- [Static Multi-Tenancy Configuration](../monolithic-multitenancy-static/README.md)
- [Single Tenant Authentication](../tempo-single-tenant-auth/README.md)
- [OpenShift RBAC Documentation](https://docs.openshift.com/container-platform/latest/authentication/using-rbac.html)
- [TempoStack Multi-Tenancy Guide](../../../docs/multitenancy-configuration.md)
- [OpenTelemetry Multi-Tenant Setup](https://opentelemetry.io/docs/concepts/multitenancy/)