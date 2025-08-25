# TempoStack Gateway with OIDC Authentication and Jaeger Query Control

This configuration blueprint demonstrates how to deploy TempoStack with an authentication gateway featuring OIDC integration, static multi-tenancy, and dynamic Jaeger query component control. This setup showcases enterprise-grade authentication and flexible query interface management for production observability deployments.

## Overview

This test validates a gateway-enabled observability stack featuring:
- **Authentication Gateway**: OIDC-based authentication with tenant routing
- **Static Multi-Tenancy**: Pre-configured tenant authentication and authorization
- **Dynamic Jaeger Query Control**: Runtime enabling/disabling of Jaeger query interface
- **Role-Based Authorization**: Granular permissions for different user roles
- **External Identity Provider**: Integration with Dex OIDC provider

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ External Users  │───▶│  Authentication      │───▶│   TempoStack    │
│ - OIDC Login    │    │     Gateway          │    │ ┌─────────────┐ │
│ - JWT Tokens    │    │ ┌─────────────────┐  │    │ │ Multi-Tenant│ │
└─────────────────┘    │ │ OIDC Provider   │  │    │ │ Components  │ │
                       │ │ (Dex)           │  │    │ └─────────────┘ │
┌─────────────────┐    │ │ Tenant Routing  │  │    └─────────────────┘
│ Query Interfaces│◀───│ │ RBAC Policies   │  │              │
│ - Gateway API   │    │ └─────────────────┘  │              │
│ - Jaeger UI     │    └──────────────────────┘              │
│   (Optional)    │                                          │
└─────────────────┘    ┌──────────────────────┐              │
                       │ MinIO Object Storage │◀─────────────┘
                       │ (S3 Compatible)      │
                       └──────────────────────┘
```

## Prerequisites

- Kubernetes cluster with sufficient resources
- Tempo Operator installed
- OIDC provider (e.g., Dex) available for authentication
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Deploy OIDC Provider (Dex)

First, deploy Dex as the OIDC identity provider:

```bash
# This step assumes you have Dex deployed
# Example Dex configuration would include:
# - issuerURL: http://dex.svc:30556/dex
# - Client configuration for Tempo integration
# - User directory (LDAP, static users, etc.)
```

### Step 2: Create Authentication Secrets

Create secrets for object storage and OIDC configuration:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
   name: minio-test
stringData:
  endpoint: http://minio.minio.svc:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
   name: oidc-test
stringData:
  clientID: test
  clientSecret: ZXhhbXBsZS1hcHAtc2VjcmV0
type: Opaque
EOF
```

**Secret Configuration Details**:
- `minio-test`: Object storage credentials for trace persistence
- `oidc-test`: OIDC client credentials for authentication flow

**Reference**: [`01-install.yaml`](./01-install.yaml)

### Step 3: Deploy TempoStack with Gateway

Create TempoStack with authentication gateway and multi-tenancy:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: foo
spec:
  template:
    gateway:
      enabled: true
    queryFrontend:
      jaegerQuery:
        enabled: false
  storage:
    secret:
      type: s3
      name: minio-test
  storageSize: 200M
  tenants:
    mode: static
    authentication:
      - tenantName: test-oidc
        tenantId: test-oidc
        oidc:
          issuerURL: http://dex.svc:30556/dex
          redirectURL: http://tempo-foo-gateway.svc:8080/oidc/test-oidc/callback
          usernameClaim: email
          secret:
            name: oidc-test
    authorization:
      roleBindings:
      - name: "test"
        roles:
        - read-write
        subjects:
        - kind: user
          name: "admin@example.com"
      roles:
      - name: read-write
        permissions:
        - read
        - write
        resources:
        - logs
        - metrics
        - traces
        tenants:
        - test-oidc
EOF
```

**Key Gateway Configuration Details**:

#### Gateway Settings
- `gateway.enabled: true`: Enables authentication gateway
- `jaegerQuery.enabled: false`: Initially disables Jaeger query interface

#### Static Multi-Tenancy
- `tenants.mode: static`: Uses pre-configured tenant definitions
- Tenant authentication via OIDC provider integration

#### OIDC Configuration
- `issuerURL`: External OIDC provider endpoint
- `redirectURL`: Callback URL for authentication flow
- `usernameClaim`: JWT claim used for user identification

#### Authorization Model
- **Roles**: Define permissions (read, write) for resources
- **RoleBindings**: Map users to roles for specific tenants
- **Resources**: Control access to logs, metrics, and traces

**Reference**: [`03-install-disable-jaeger-query.yaml`](./03-install-disable-jaeger-query.yaml)

### Step 4: Verify Gateway Deployment

Check that the gateway and authentication components are running:

```bash
# Verify TempoStack status
kubectl get tempostack foo -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Check gateway pod
kubectl get pods -l app.kubernetes.io/component=gateway

# Verify gateway service
kubectl get svc tempo-foo-gateway
```

### Step 5: Test OIDC Authentication Flow

Test the authentication workflow:

```bash
# Port-forward to gateway
kubectl port-forward svc/tempo-foo-gateway 8080:8080

# Access gateway (will redirect to OIDC provider)
curl -v http://localhost:8080/api/search?q={}
# Should return 302 redirect to Dex login page
```

### Step 6: Enable Jaeger Query Interface

Dynamically enable the Jaeger query interface:

```bash
kubectl patch tempostack foo --type='merge' -p='{"spec":{"template":{"queryFrontend":{"jaegerQuery":{"enabled":true}}}}}'
```

Verify Jaeger query container is added:

```bash
# Wait for deployment update
kubectl rollout status deployment/tempo-foo-query-frontend

# Check for tempo-query container
kubectl get deployment tempo-foo-query-frontend -o jsonpath='{.spec.template.spec.containers[*].name}'
# Should include: tempo-query-frontend, tempo-query
```

### Step 7: Disable Jaeger Query Interface

Test dynamic disabling of Jaeger query:

```bash
kubectl patch tempostack foo --type='merge' -p='{"spec":{"template":{"queryFrontend":{"jaegerQuery":{"enabled":false}}}}}'
```

Verify the tempo-query container is removed:

```bash
# Wait for container removal
while kubectl get deployment tempo-foo-query-frontend -o jsonpath='{.spec.template.spec.containers[*].name}' | grep -q tempo-query; do
  echo "tempo-query container still exists. Waiting..."
  sleep 5
done
echo "tempo-query container successfully removed"
```

## Key Features Demonstrated

### 1. **Authentication Gateway**
- **OIDC Integration**: Enterprise identity provider support
- **Token Validation**: JWT token verification and claims extraction
- **Tenant Routing**: Automatic tenant identification from authentication
- **Session Management**: Secure session handling and logout

### 2. **Static Multi-Tenancy**
- **Pre-configured Tenants**: Admin-defined tenant configurations
- **OIDC per Tenant**: Different identity providers per tenant
- **Isolation**: Complete data isolation between tenants
- **Flexible Authentication**: Multiple authentication methods supported

### 3. **Role-Based Authorization**
- **Granular Permissions**: Read/write access control
- **Resource-based Security**: Separate permissions for logs, metrics, traces
- **User-to-Role Mapping**: Flexible role binding system
- **Tenant-scoped Roles**: Roles applied per tenant

### 4. **Dynamic Query Interface Control**
- **Runtime Configuration**: Enable/disable Jaeger query without downtime
- **Container Management**: Automatic container addition/removal
- **Service Discovery**: Automatic service endpoint updates
- **Zero-downtime Updates**: Seamless configuration changes

## Authentication Flow

### OIDC Authentication Process

1. **Initial Request**: User accesses gateway endpoint
2. **Authentication Check**: Gateway validates existing session/token
3. **OIDC Redirect**: Redirect to configured OIDC provider
4. **User Login**: User authenticates with identity provider
5. **Token Exchange**: OIDC provider returns authorization code
6. **Token Validation**: Gateway exchanges code for access token
7. **Claims Extraction**: Extract user identity and tenant information
8. **Authorization**: Apply RBAC policies based on user/tenant
9. **Request Proxying**: Forward authorized requests to Tempo components

### Static Tenant Configuration

```yaml
# Example tenant configuration
tenants:
  mode: static
  authentication:
    - tenantName: production
      tenantId: prod-001
      oidc:
        issuerURL: https://auth.company.com
        redirectURL: https://tempo-gateway.company.com/oidc/production/callback
        usernameClaim: preferred_username
        groupsClaim: groups
        secret:
          name: prod-oidc-secret
    - tenantName: development
      tenantId: dev-001
      oidc:
        issuerURL: https://dev-auth.company.com
        redirectURL: https://tempo-gateway.company.com/oidc/development/callback
        usernameClaim: email
        secret:
          name: dev-oidc-secret
```

## Authorization Configuration

### Role Definitions

```yaml
authorization:
  roles:
  - name: admin
    permissions: [read, write]
    resources: [logs, metrics, traces]
    tenants: ["*"]  # All tenants
  
  - name: developer
    permissions: [read, write]
    resources: [traces]
    tenants: [development, staging]
  
  - name: viewer
    permissions: [read]
    resources: [traces, metrics]
    tenants: [production]
```

### Role Bindings

```yaml
authorization:
  roleBindings:
  - name: admin-binding
    roles: [admin]
    subjects:
    - kind: user
      name: "admin@company.com"
    - kind: group
      name: "platform-team"
  
  - name: dev-team-binding
    roles: [developer]
    subjects:
    - kind: group
      name: "development-team"
```

## Gateway API Usage

### Authenticated Requests

```bash
# Get authentication token
TOKEN=$(curl -s -X POST https://auth.company.com/oauth2/token \
  -d "grant_type=client_credentials" \
  -d "client_id=tempo-client" \
  -d "client_secret=secret" | jq -r .access_token)

# Query traces with authentication
curl -H "Authorization: Bearer $TOKEN" \
  -H "X-Scope-OrgID: test-oidc" \
  "http://tempo-foo-gateway:8080/api/search?q={}"

# Search specific service traces
curl -H "Authorization: Bearer $TOKEN" \
  -H "X-Scope-OrgID: test-oidc" \
  "http://tempo-foo-gateway:8080/api/search?service=my-service"
```

### Tenant-specific Queries

```bash
# Production tenant query
curl -H "Authorization: Bearer $PROD_TOKEN" \
  -H "X-Scope-OrgID: production" \
  "http://tempo-foo-gateway:8080/api/search"

# Development tenant query  
curl -H "Authorization: Bearer $DEV_TOKEN" \
  -H "X-Scope-OrgID: development" \
  "http://tempo-foo-gateway:8080/api/search"
```

## Troubleshooting

### Authentication Issues

```bash
# Check gateway logs
kubectl logs -l app.kubernetes.io/component=gateway

# Verify OIDC configuration
kubectl describe tempostack foo

# Test OIDC provider connectivity
kubectl exec deployment/tempo-foo-gateway -- \
  curl -v http://dex.svc:30556/dex/.well-known/openid_configuration
```

### Authorization Problems

```bash
# Check user permissions
kubectl logs -l app.kubernetes.io/component=gateway | grep -i "authorization\|rbac"

# Verify role bindings
kubectl get tempostack foo -o yaml | grep -A 20 authorization

# Test token validation
kubectl exec deployment/tempo-foo-gateway -- \
  curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/search
```

### Jaeger Query Control Issues

```bash
# Check query frontend deployment
kubectl describe deployment tempo-foo-query-frontend

# Verify container configuration
kubectl get deployment tempo-foo-query-frontend -o yaml | grep -A 10 containers

# Monitor configuration changes
kubectl get events --sort-by='.lastTimestamp' | grep tempo-foo
```

### Common Issues

1. **OIDC Redirect Loop**:
   ```bash
   # Check redirect URL configuration
   kubectl get secret oidc-test -o yaml
   # Ensure redirectURL matches gateway service endpoint
   ```

2. **Permission Denied**:
   ```bash
   # Verify user in role bindings
   kubectl get tempostack foo -o jsonpath='{.spec.tenants.authorization.roleBindings[*].subjects[*].name}'
   ```

3. **Gateway Not Responding**:
   ```bash
   # Check service endpoints
   kubectl get endpoints tempo-foo-gateway
   # Verify pod readiness
   kubectl get pods -l app.kubernetes.io/component=gateway
   ```

## Production Considerations

### 1. **Security Hardening**
- Use HTTPS for all OIDC communications
- Implement proper JWT token validation
- Configure secure session storage
- Regular rotation of client secrets

### 2. **High Availability**
- Deploy multiple gateway replicas
- Use external session storage (Redis)
- Implement health checks and monitoring
- Configure proper load balancing

### 3. **Performance Optimization**
- Cache OIDC provider responses
- Optimize JWT token validation
- Use connection pooling
- Monitor authentication latency

### 4. **Monitoring and Alerting**
- Track authentication success/failure rates
- Monitor OIDC provider availability
- Alert on permission denied events
- Log security events for audit

## Related Configurations

- [Multi-tenant RBAC](../../e2e-openshift/multitenancy-rbac/README.md) - Advanced RBAC patterns
- [Single Tenant Auth](../../e2e-openshift/tempo-single-tenant-auth/README.md) - Simplified authentication
- [TLS Security](../../e2e-openshift/tls-singletenant/README.md) - Encrypted communications
- [Basic TempoStack](../compatibility/README.md) - Non-authenticated baseline

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/gateway
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test demonstrates dynamic Jaeger query component control and requires a properly configured OIDC provider for full functionality.