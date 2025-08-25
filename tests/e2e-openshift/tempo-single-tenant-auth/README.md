# Single Tenant Authentication with OpenShift Integration

This test demonstrates how to configure TempoStack in single-tenant mode with OpenShift authentication integration. This setup provides a secure, authenticated access to Jaeger UI while maintaining simplicity for single-tenant environments.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   OpenShift     │    │   Jaeger Query   │    │   TempoStack    │
│   Users/Groups  │───▶│   Frontend       │───▶│   Backend       │
│                 │    │   (Authenticated)│    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         │                        │                       ▼
         ▼                        ▼              ┌─────────────────┐
┌─────────────────┐    ┌──────────────────┐    │   Trace Storage │
│  OAuth Token    │    │  SAR (SubjectAccess│    │   (MinIO S3)    │
│  Authentication │    │  Review) Check   │    └─────────────────┘
└─────────────────┘    └──────────────────┘
```

## Authentication Features

### OpenShift Integration
- **OAuth Authentication**: Uses OpenShift's built-in OAuth server
- **Subject Access Review**: Validates user permissions via Kubernetes SAR
- **Token-Based**: Leverages OpenShift user tokens for authentication
- **Route Integration**: Secured access via OpenShift route

### Access Control
- **SAR Configuration**: `{"namespace": "chainsaw-mst", "resource": "pods", "verb": "get"}`
- **Resource Validation**: Users must have specific Kubernetes permissions
- **Single Tenant**: Simplified authentication for single-tenant environments
- **Jaeger UI**: Authenticated access to trace visualization

## Test Components

### TempoStack Authentication Configuration
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Authentication**: Enabled with OpenShift OAuth integration
- **SAR Check**: Subject Access Review for permission validation
- **Jaeger UI**: Enabled with route-based access
- **Resources**: Dedicated CPU/memory allocation for auth component

### Storage Configuration
- **File**: [`install-storage.yaml`](./install-storage.yaml)
- **Backend**: MinIO S3-compatible object storage
- **Size**: 200M storage allocation for traces
- **Integration**: Seamless with authenticated TempoStack

### Trace Generation and Verification
- **Generation**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Jaeger Verification**: [`verify-traces-jaeger.yaml`](./verify-traces-jaeger.yaml)
- **TraceQL Verification**: [`verify-traces-traceql.yaml`](./verify-traces-traceql.yaml)

## Quick Start

### Prerequisites
- OpenShift cluster with Tempo Operator
- Cluster administrator access for SAR configuration
- MinIO or S3-compatible storage
- Valid OpenShift user accounts

### Step-by-Step Deployment

1. **Install Storage Backend**
   ```bash
   # Deploy MinIO for authenticated trace storage
   kubectl apply -f install-storage.yaml
   kubectl wait --for=condition=ready pod -l app=minio -n chainsaw-tst --timeout=300s
   ```

2. **Deploy Authenticated TempoStack**
   ```bash
   # Create TempoStack with OpenShift authentication
   kubectl apply -f install-tempo.yaml
   kubectl wait --for=condition=ready tempostack tempo-st -n chainsaw-tst --timeout=300s
   ```

3. **Generate Test Traces**
   ```bash
   # Create traces for authentication testing
   kubectl apply -f generate-traces.yaml
   ```

4. **Verify Authentication Integration**
   ```bash
   # Test Jaeger UI access with authentication
   kubectl apply -f verify-traces-jaeger.yaml
   
   # Test TraceQL API access
   kubectl apply -f verify-traces-traceql.yaml
   ```

5. **Access Jaeger UI**
   ```bash
   # Get the authenticated route URL
   oc get route -n chainsaw-tst
   
   # Access via browser (will redirect to OpenShift OAuth)
   # Users will be prompted to log in with OpenShift credentials
   ```

## Authentication Configuration Details

### Jaeger Query Authentication
```yaml
queryFrontend:
  jaegerQuery:
    enabled: true
    authentication:
      enabled: true  # Enable OpenShift OAuth integration
      sar: "{\"namespace\": \"chainsaw-mst\", \"resource\": \"pods\", \"verb\": \"get\"}"
      resources:  # Dedicated resources for auth proxy
        limits:
          cpu: 200m
          memory: 512Gi
        requests:
          cpu: 100m
          memory: 256Mi
    ingress:
      type: route  # OpenShift route for external access
```

### Subject Access Review (SAR)
- **Purpose**: Validates user permissions before granting trace access
- **Namespace**: `chainsaw-mst` - target namespace for permission check
- **Resource**: `pods` - Kubernetes resource type to check
- **Verb**: `get` - required permission level
- **Effect**: Users must have "get pods" permission in the specified namespace

## Testing Procedure

The complete test is defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml) and executes:

1. **Storage Setup**: Deploy MinIO object storage
2. **Authentication Setup**: Create TempoStack with OAuth authentication
3. **Trace Generation**: Generate test traces
4. **Jaeger UI Test**: Verify authenticated access to Jaeger UI
5. **TraceQL Test**: Verify authenticated API access via TraceQL

## Authentication Flow

### User Login Process
1. **Route Access**: User accesses Jaeger UI via OpenShift route
2. **OAuth Redirect**: System redirects to OpenShift OAuth server
3. **User Authentication**: User logs in with OpenShift credentials
4. **Token Generation**: OAuth server generates access token
5. **SAR Validation**: System validates user permissions via SAR
6. **Access Granted**: User gains access to Jaeger UI and traces

### Permission Validation
```bash
# Check if user has required permissions
oc auth can-i get pods -n chainsaw-mst --as=username

# Test SAR directly
oc auth can-i get pods -n chainsaw-mst --as=system:serviceaccount:namespace:serviceaccount
```

## Production Considerations

### Security Best Practices
- **Least Privilege**: Configure SAR with minimal required permissions
- **Token Rotation**: Leverage OpenShift's automatic token rotation
- **Network Policies**: Restrict network access to authenticated endpoints
- **Audit Logging**: Enable OAuth and API access logging

### Performance Optimization
- **Auth Component Sizing**: Allocate appropriate resources for auth proxy
- **Token Caching**: OAuth tokens are cached to reduce authentication overhead
- **Route Optimization**: Configure route with appropriate timeout settings
- **Session Management**: Configure session timeouts for security

### Scaling Considerations
- **High Availability**: Deploy multiple Jaeger Query replicas for availability
- **Load Balancing**: OpenShift routes provide automatic load balancing
- **Resource Limits**: Set appropriate CPU/memory limits for auth components
- **Monitoring**: Monitor authentication success/failure rates

## Troubleshooting

### Common Authentication Issues

1. **Access Denied Errors**
   ```bash
   # Check user permissions
   oc auth can-i get pods -n chainsaw-mst --as=username
   
   # Verify SAR configuration
   oc get tempostack tempo-st -n chainsaw-tst -o jsonpath='{.spec.template.queryFrontend.jaegerQuery.authentication.sar}'
   ```

2. **OAuth Redirect Problems**
   ```bash
   # Check route configuration
   oc get route -n chainsaw-tst -o yaml
   
   # Verify OAuth client registration
   oc get oauthclient -o yaml
   ```

3. **Authentication Proxy Issues**
   ```bash
   # Check auth proxy logs
   oc logs -n chainsaw-tst -l app.kubernetes.io/component=query-frontend
   
   # Verify auth proxy resources
   oc describe pod -n chainsaw-tst -l app.kubernetes.io/component=query-frontend
   ```

### Debug Commands
```bash
# Check TempoStack authentication status
oc get tempostack tempo-st -n chainsaw-tst -o jsonpath='{.status.conditions}'

# Test route accessibility
curl -I $(oc get route -n chainsaw-tst -o jsonpath='{.items[0].spec.host}')

# Verify OAuth integration
oc get oauthclient | grep jaeger

# Check authentication metrics
oc exec -n chainsaw-tst deployment/tempo-st-query-frontend -- curl localhost:8080/metrics | grep auth
```

## User Management

### Creating Users for Testing
```bash
# Create test user (cluster admin required)
oc create user testuser
oc create identity htpasswd_provider:testuser
oc create useridentitymapping htpasswd_provider:testuser testuser

# Grant required permissions
oc adm policy add-role-to-user view testuser -n chainsaw-mst
```

### Group-Based Access
```bash
# Create group with trace access
oc adm groups new tempo-users
oc adm groups add-users tempo-users user1 user2
oc adm policy add-role-to-group view tempo-users -n chainsaw-mst
```

## Comparison with Multi-Tenant Authentication

| Feature | Single Tenant Auth | Multi-Tenant Auth |
|---------|-------------------|-------------------|
| Complexity | Lower | Higher |
| User Isolation | None (single tenant) | Full tenant isolation |
| Authentication | OpenShift OAuth only | OAuth + tenant mapping |
| Permission Model | Single SAR check | Per-tenant RBAC |
| Use Case | Simple environments | Enterprise multi-team |
| Setup Effort | Minimal | Extensive |

## Related Resources

- [Multi-Tenancy with RBAC](../multitenancy-rbac/README.md)
- [OpenShift OAuth Configuration](https://docs.openshift.com/container-platform/latest/authentication/configuring-oauth-clients.html)
- [Kubernetes Subject Access Review](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#checking-api-access)
- [Jaeger Authentication Guide](https://www.jaegertracing.io/docs/latest/security/)
- [TempoStack Authentication Options](../../../docs/authentication-configuration.md)