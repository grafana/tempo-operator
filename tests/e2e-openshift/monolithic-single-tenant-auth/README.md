# TempoMonolithic Single Tenant Authentication

This test demonstrates how to configure TempoMonolithic with OpenShift authentication integration for single-tenant environments. This provides a simplified deployment model with built-in authentication using OpenShift's OAuth system while maintaining the simplicity of a monolithic architecture.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   OpenShift     │    │  TempoMonolithic │    │   Local Storage │
│   Users/Groups  │───▶│   with Jaeger UI │───▶│   (Embedded)    │
│                 │    │   (Authenticated)│    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         │                        │                       │
         ▼                        ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  OAuth Token    │    │  SAR (Subject    │    │  Trace Data     │
│  Authentication │    │  Access Review)  │    │  (In-Memory)    │
│                 │    │  Authorization   │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Deployment Model Comparison

### TempoMonolithic vs TempoStack
- **Simplicity**: Single binary vs multi-component architecture
- **Resource Usage**: Lower resource footprint for small-scale deployments
- **Scalability**: Vertical scaling vs horizontal scaling
- **Storage**: Local/in-memory storage vs object storage
- **Use Case**: Development, testing, small-scale production vs enterprise-scale

## Test Components

### TempoMonolithic Authentication Configuration
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Type**: `TempoMonolithic` (single binary deployment)
- **Authentication**: OpenShift OAuth integration with SAR validation
- **UI Access**: Jaeger UI with route-based external access
- **Storage**: Embedded storage (no external object store required)

### Authentication Features
- **OAuth Integration**: Uses OpenShift's built-in OAuth server
- **SAR Check**: Subject Access Review for permission validation
- **Route Access**: OpenShift route for external Jaeger UI access
- **Resource Limits**: Dedicated resources for authentication proxy

## Quick Start

### Prerequisites
- OpenShift cluster with Tempo Operator
- Valid OpenShift user accounts
- No external storage requirements (uses embedded storage)

### Step-by-Step Deployment

1. **Deploy Authenticated TempoMonolithic**
   ```bash
   # Create TempoMonolithic with OpenShift authentication
   kubectl apply -f install-tempo.yaml
   kubectl wait --for=condition=ready tempomonolithic monolithic-st -n chainsaw-mst --timeout=300s
   ```

2. **Generate Test Traces**
   ```bash
   # Create sample traces for authentication testing
   kubectl apply -f generate-traces.yaml
   ```

3. **Verify Authentication Integration**
   ```bash
   # Test Jaeger UI access with authentication
   kubectl apply -f verify-traces-jaeger.yaml
   
   # Test TraceQL API access
   kubectl apply -f verify-traces-traceql.yaml
   ```

4. **Access Jaeger UI**
   ```bash
   # Get the authenticated route URL
   oc get route -n chainsaw-mst
   
   # Access via browser (will redirect to OpenShift OAuth)
   # Users will authenticate with OpenShift credentials
   ```

## Authentication Configuration Details

### TempoMonolithic with Authentication
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: monolithic-st
spec:
  jaegerui:
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
    route:
      enabled: true  # Create OpenShift route for external access
```

### Subject Access Review (SAR)
- **Namespace**: `chainsaw-mst` - target namespace for permission validation
- **Resource**: `pods` - Kubernetes resource type to check
- **Verb**: `get` - required permission level
- **Purpose**: Users must have "get pods" permission in specified namespace

## Testing Procedure

The complete test is defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml) and executes:

1. **Monolithic Deployment**: Create TempoMonolithic with OAuth authentication
2. **Trace Generation**: Generate test traces using telemetrygen
3. **Jaeger UI Test**: Verify authenticated access to Jaeger UI
4. **TraceQL Test**: Verify authenticated API access via TraceQL

## Authentication Flow

### User Access Process
1. **Route Access**: User accesses Jaeger UI via OpenShift route
2. **OAuth Redirect**: System redirects to OpenShift OAuth server
3. **User Login**: User authenticates with OpenShift credentials
4. **Token Validation**: OAuth server issues access token
5. **SAR Authorization**: System validates user permissions via SAR
6. **UI Access**: User gains access to Jaeger UI and trace data

### Permission Validation
```bash
# Check user permissions for SAR validation
oc auth can-i get pods -n chainsaw-mst --as=username

# Test service account permissions
oc auth can-i get pods -n chainsaw-mst --as=system:serviceaccount:namespace:serviceaccount
```

## Monolithic Architecture Benefits

### Simplified Deployment
- **Single Binary**: One container with all Tempo components
- **No External Dependencies**: No object storage or complex networking
- **Quick Setup**: Minimal configuration required
- **Resource Efficient**: Lower overhead for small-scale deployments

### Embedded Storage
- **In-Memory Storage**: Fast access for recent traces
- **Local Persistence**: Optional local disk storage
- **No Network Dependencies**: Self-contained trace storage
- **Development Friendly**: Easy setup for testing and development

## Production Considerations

### Scalability Limitations
- **Vertical Scaling Only**: Cannot scale components independently
- **Memory Constraints**: Limited by single node memory capacity
- **Storage Limits**: Local storage constraints
- **High Availability**: Single point of failure

### Appropriate Use Cases
- **Development Environments**: Testing and debugging
- **Small-Scale Production**: Low-volume trace collection
- **Edge Deployments**: Resource-constrained environments
- **Proof of Concept**: Rapid prototyping and evaluation

### Migration Path
- **Start Simple**: Begin with TempoMonolithic for simplicity
- **Scale Up**: Migrate to TempoStack as requirements grow
- **Hybrid Approach**: Use both models for different environments
- **Gradual Transition**: Incremental migration strategy

## Security Features

### Authentication Integration
- **OpenShift OAuth**: Native integration with OpenShift authentication
- **Token-Based Access**: Secure token validation
- **Permission-Based**: RBAC integration via SAR
- **Route Security**: TLS-secured external access

### Access Control
```bash
# Create user with appropriate permissions
oc adm policy add-role-to-user view username -n chainsaw-mst

# Create group-based access
oc adm groups new tempo-users
oc adm groups add-users tempo-users user1 user2
oc adm policy add-role-to-group view tempo-users -n chainsaw-mst
```

## Troubleshooting

### Common Authentication Issues

1. **Access Denied Errors**
   ```bash
   # Check user permissions
   oc auth can-i get pods -n chainsaw-mst --as=username
   
   # Verify SAR configuration
   oc get tempomonolithic monolithic-st -n chainsaw-mst -o jsonpath='{.spec.jaegerui.authentication.sar}'
   ```

2. **Route Access Problems**
   ```bash
   # Check route configuration
   oc get route -n chainsaw-mst -o yaml
   
   # Verify route accessibility
   curl -I $(oc get route -n chainsaw-mst -o jsonpath='{.items[0].spec.host}')
   ```

3. **Authentication Proxy Issues**
   ```bash
   # Check auth proxy logs
   oc logs -n chainsaw-mst -l app.kubernetes.io/component=jaegerui
   
   # Verify proxy resources
   oc describe pod -n chainsaw-mst -l app.kubernetes.io/component=jaegerui
   ```

### Debug Commands
```bash
# Check TempoMonolithic status
oc get tempomonolithic monolithic-st -n chainsaw-mst -o jsonpath='{.status.conditions}'

# Test authentication endpoint
curl -I $(oc get route -n chainsaw-mst -o jsonpath='{.items[0].spec.host}')/oauth/start

# Check OAuth integration
oc get oauthclient | grep tempo

# Monitor authentication metrics
oc exec -n chainsaw-mst deployment/monolithic-st -- curl localhost:3200/metrics | grep auth
```

## Performance Characteristics

### Resource Usage
- **CPU**: Lower CPU overhead due to single binary
- **Memory**: Efficient memory usage for small trace volumes
- **Disk I/O**: Minimal disk I/O with in-memory storage
- **Network**: Reduced internal network traffic

### Scaling Considerations
```yaml
spec:
  resources:
    limits:
      cpu: 2000m      # Adjust based on trace volume
      memory: 4Gi     # Allocate sufficient memory for trace storage
    requests:
      cpu: 500m
      memory: 1Gi
```

## Migration to TempoStack

### When to Migrate
- **High Trace Volume**: Exceeding monolithic capacity
- **Horizontal Scaling**: Need for component-level scaling
- **Object Storage**: Requirement for durable trace storage
- **Multi-Tenancy**: Complex access control requirements

### Migration Strategy
```bash
# 1. Deploy TempoStack alongside TempoMonolithic
oc apply -f tempostack-configuration.yaml

# 2. Route new traces to TempoStack
# Update collector configuration

# 3. Verify TempoStack functionality
# Test trace ingestion and queries

# 4. Decommission TempoMonolithic
oc delete tempomonolithic monolithic-st
```

## Related Resources

- [TempoStack Single Tenant Auth](../tempo-single-tenant-auth/README.md)
- [TempoStack Multi-Tenancy](../multitenancy-rbac/README.md)
- [OpenShift OAuth Documentation](https://docs.openshift.com/container-platform/latest/authentication/configuring-oauth-clients.html)
- [Tempo Monolithic vs Stack Comparison](../../../docs/deployment-models.md)
- [Migration Guide](../../../docs/monolithic-to-stack-migration.md)