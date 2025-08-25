# TempoMonolithic with Static OIDC Multitenancy

This configuration blueprint demonstrates TempoMonolithic's static multitenancy mode with external OIDC (OpenID Connect) authentication using Hydra as the identity provider. This setup provides enterprise-grade multitenancy with predefined tenants, static role-based authorization, and OAuth2 client credentials flow for secure trace ingestion and querying.

## Overview

This test validates static OIDC multitenancy features:
- **External OIDC Provider**: Integration with Hydra OAuth2/OIDC server
- **Static Tenant Configuration**: Predefined tenants with fixed identities and permissions
- **OAuth2 Client Credentials Flow**: Service-to-service authentication using client credentials
- **Role-Based Authorization**: Static roles and role bindings for fine-grained access control
- **OIDC Token Validation**: JWT token verification and claims-based tenant identification

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ Hydra OAuth2 Server     │◀───│   TempoMonolithic        │───▶│ Tenant: tenant1         │
│ - OIDC Provider         │    │   Static Multitenancy    │    │ - ID: tenant1           │
│ - Client Registration   │    │ ┌─────────────────────┐  │    │ - OIDC Authentication   │
│ - Token Issuance        │    │ │ OIDC Authenticator  │  │    └─────────────────────────┘
└─────────────────────────┘    │ │ - JWT Validation    │  │
                               │ │ - Claims Extraction │  │    ┌─────────────────────────┐
┌─────────────────────────┐    │ │ - Tenant Mapping    │  │───▶│ Role-Based Authorization│
│ OpenTelemetry Collector │───▶│ └─────────────────────┘  │    │ - Role: allow-rw-tenant1│
│ - OIDC Token Auth       │    │ Gateway Component        │    │ - Permissions: read/write│
│ - Client Credentials    │    └──────────────────────────┘    │ - Subject: oidc-client  │
│ - Bearer Token Header   │                                    └─────────────────────────┘
└─────────────────────────┘    ┌──────────────────────────┐
                               │   Authorization Model    │    ┌─────────────────────────┐
┌─────────────────────────┐    │   - Static Roles         │───▶│ Jaeger UI               │
│ Client Registration     │───▶│   - Role Bindings        │    │ - Multi-tenant Access   │
│ - client_id: tenant1... │    │   - Subject Mapping      │    │ - OpenShift Route       │
│ - client_secret: ...    │    └──────────────────────────┘    └─────────────────────────┘
└─────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Understanding of OAuth2/OIDC protocols
- Knowledge of JWT tokens and claims-based authorization

## Step-by-Step Configuration

### Step 1: Deploy Hydra OAuth2/OIDC Server

Set up Hydra as the external identity provider for OIDC authentication:

```bash
oc apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hydra
spec:
  selector:
    matchLabels:
      app: hydra
  template:
    metadata:
      labels:
        app: hydra
    spec:
      containers:
      - name: hydra
        image: docker.io/oryd/hydra:v2.2.0
        command: ["hydra", "serve", "all", "--dev", "--sqa-opt-out"]
        env:
        - name: DSN
          value: memory
        - name: SECRETS_SYSTEM
          value: saf325iouepdsg8574nb39afdu
        - name: URLS_SELF_ISSUER
          value: http://hydra:4444
        - name: STRATEGIES_ACCESS_TOKEN
          value: jwt
        ports:
        - containerPort: 4444
          name: public
        - containerPort: 4445
          name: internal
---
apiVersion: v1
kind: Service
metadata:
  name: hydra
spec:
  selector:
    app: hydra
  ports:
  - name: public
    port: 4444
    targetPort: public
  - name: internal
    port: 4445
    targetPort: internal
EOF
```

**Hydra Configuration Details**:

#### OAuth2/OIDC Server Setup
- `hydra serve all --dev`: Runs public and admin APIs in development mode
- `DSN: memory`: In-memory storage for development/testing
- `URLS_SELF_ISSUER`: OAuth2 issuer URL used for token validation
- `STRATEGIES_ACCESS_TOKEN: jwt`: Uses JWT tokens for access tokens

#### Service Endpoints
- **Port 4444 (public)**: OAuth2 authorization and token endpoints for clients
- **Port 4445 (internal)**: Admin API for client management and introspection

**Reference**: [`00-install-hydra.yaml`](./00-install-hydra.yaml)

### Step 2: Configure OAuth2 Client for Tenant Authentication

Register an OAuth2 client for tenant1 authentication:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: setup-hydra
spec:
  template:
    spec:
      containers:
      - name: setup-hydra
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command: ["/bin/bash", "-eux", "-c"]
        args:
        - |
          # create OAuth2 client
          client_id=tenant1-oidc-client
          client_secret=ZXhhbXBsZS1hcHAtc2VjcmV0 # notsecret
          curl -v \
            --data '{"audience": ["'$client_id'"], "client_id": "'$client_id'", "client_secret": "'$client_secret'", "grant_types": ["client_credentials"], "token_endpoint_auth_method": "client_secret_basic"}' \
            http://hydra:4445/admin/clients
      restartPolicy: Never
EOF
```

**OAuth2 Client Configuration**:

#### Client Credentials
- `client_id: tenant1-oidc-client`: Unique client identifier for tenant1
- `client_secret: ZXhhbXBsZS1hcHAtc2VjcmV0`: Base64-encoded client secret
- **Service-to-Service**: Designed for machine-to-machine authentication

#### Grant Types and Authentication
- `grant_types: ["client_credentials"]`: OAuth2 client credentials flow
- `token_endpoint_auth_method: "client_secret_basic"`: HTTP Basic authentication
- **Audience**: Restricts token usage to specific audience

**Reference**: [`01-setup-hydra.yaml`](./01-setup-hydra.yaml)

### Step 3: Deploy TempoMonolithic with Static OIDC Multitenancy

Create TempoMonolithic with comprehensive static tenant configuration:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
   name: tenant1-oidc-secret
stringData:
  clientID: tenant1-oidc-client
type: Opaque
---
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
  multitenancy:
    enabled: true
    mode: static
    authentication:
    - tenantName: tenant1
      tenantId: tenant1
      oidc:
        issuerURL: http://hydra:4444
        secret:
          name: tenant1-oidc-secret
    authorization:
      roles:
      - name: allow-rw-tenant1
        permissions:
        - read
        - write
        resources:
        - traces
        tenants:
        - tenant1
      roleBindings:
      - name: assign-allow-rw-tenant1
        roles:
        - allow-rw-tenant1
        subjects:
        - kind: user
          name: tenant1-oidc-client
EOF
```

**Key Configuration Elements**:

#### Static Multitenancy Mode
- `mode: static`: Uses predefined tenant configuration
- **Tenant Definition**: Static tenant mapping with fixed identities
- **No Dynamic Discovery**: Tenants must be explicitly configured

#### OIDC Authentication Configuration
```yaml
authentication:
- tenantName: tenant1
  tenantId: tenant1
  oidc:
    issuerURL: http://hydra:4444
    secret:
      name: tenant1-oidc-secret
```

**OIDC Configuration Details**:
- `tenantName`: Human-readable tenant identifier
- `tenantId`: Internal tenant ID used for data isolation
- `issuerURL`: OAuth2/OIDC issuer for token validation
- `secret`: Kubernetes secret containing OIDC client configuration

#### Static Role-Based Authorization
```yaml
authorization:
  roles:
  - name: allow-rw-tenant1
    permissions: [read, write]
    resources: [traces]
    tenants: [tenant1]
  roleBindings:
  - name: assign-allow-rw-tenant1
    roles: [allow-rw-tenant1]
    subjects:
    - kind: user
      name: tenant1-oidc-client
```

**Authorization Model**:
- **Roles**: Define permissions for specific resources and tenants
- **Role Bindings**: Assign roles to subjects (users, service accounts)
- **Subject Mapping**: OAuth2 client_id maps to authorization subject

**Reference**: [`02-install-tempo.yaml`](./02-install-tempo.yaml)

### Step 4: Deploy OpenTelemetry Collector with OIDC Authentication

Create an OpenTelemetry Collector configured for OIDC-based trace submission:

```bash
# Deploy collector with OIDC token authentication
# Reference: 03-install-otel.yaml
```

The collector configuration includes:
- **OAuth2 Token Acquisition**: Fetches tokens from Hydra using client credentials
- **Bearer Token Authentication**: Includes JWT tokens in trace submission requests
- **Multi-Protocol Support**: Both gRPC and HTTP with OIDC authentication

**Reference**: [`03-install-otel.yaml`](./03-install-otel.yaml)

### Step 5: Generate Traces with OIDC Authentication

Create traces using the OIDC-authenticated collector:

```bash
# Generate traces through OIDC-authenticated collector
# Reference: 04-generate-traces.yaml
```

**Trace Generation Flow**:
1. **Token Acquisition**: Collector obtains OAuth2 token from Hydra
2. **Trace Submission**: Traces sent with Bearer token in Authorization header
3. **Authentication**: Gateway validates JWT token against Hydra issuer
4. **Authorization**: Role-based access control applied based on token claims
5. **Tenant Routing**: Traces routed to appropriate tenant based on subject

**Reference**: [`04-generate-traces.yaml`](./04-generate-traces.yaml)

### Step 6: Verify Multi-Tenant Trace Access with OIDC

Validate that traces are properly isolated and accessible via OIDC authentication:

```bash
# Verify traces with OIDC token-based access
# Reference: 05-verify-traces.yaml
```

**Verification Process**:
- **Token-Based Queries**: Use OAuth2 tokens for trace query authentication
- **Tenant Isolation**: Verify only authorized tenant traces are accessible
- **Permission Validation**: Confirm read/write permissions work correctly
- **JWT Claims**: Validate proper claims extraction and tenant mapping

**Reference**: [`05-verify-traces.yaml`](./05-verify-traces.yaml)

## Static OIDC Multitenancy Features

### 1. **OIDC Authentication Flow**

#### Client Credentials Flow
```bash
# OAuth2 client credentials exchange
curl -X POST http://hydra:4444/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=tenant1-oidc-client&client_secret=ZXhhbXBsZS1hcHAtc2VjcmV0"
```

#### JWT Token Structure
```json
{
  "iss": "http://hydra:4444",
  "sub": "tenant1-oidc-client",
  "aud": ["tenant1-oidc-client"],
  "exp": 1234567890,
  "iat": 1234567800,
  "client_id": "tenant1-oidc-client",
  "scope": "openid"
}
```

#### Token Validation Process
1. **Signature Verification**: JWT signature validated against OIDC issuer
2. **Claims Extraction**: Subject and audience extracted from token
3. **Tenant Mapping**: Subject mapped to tenant configuration
4. **Authorization Check**: Role bindings evaluated for subject

### 2. **Static Configuration Model**

#### Tenant Definition
```yaml
authentication:
- tenantName: production
  tenantId: prod-tenant-uuid
  oidc:
    issuerURL: https://company-oidc.example.com
    secret:
      name: prod-oidc-secret
- tenantName: development
  tenantId: dev-tenant-uuid
  oidc:
    issuerURL: https://dev-oidc.example.com
    secret:
      name: dev-oidc-secret
```

#### Authorization Configuration
```yaml
authorization:
  roles:
  - name: prod-admin
    permissions: [read, write, delete]
    resources: [traces, metrics]
    tenants: [production]
  - name: dev-user
    permissions: [read, write]
    resources: [traces]
    tenants: [development]
  roleBindings:
  - name: prod-team-access
    roles: [prod-admin]
    subjects:
    - kind: user
      name: prod-service-account
  - name: dev-team-access
    roles: [dev-user]
    subjects:
    - kind: user
      name: dev-service-account
```

### 3. **Multi-Provider Support**

#### Multiple OIDC Providers
```yaml
authentication:
- tenantName: internal-team
  tenantId: internal-uuid
  oidc:
    issuerURL: https://internal-sso.company.com
    secret:
      name: internal-oidc-secret
- tenantName: partner-team
  tenantId: partner-uuid
  oidc:
    issuerURL: https://partner-oidc.external.com
    secret:
      name: partner-oidc-secret
```

#### Provider-Specific Configuration
```yaml
# Advanced OIDC configuration
authentication:
- tenantName: azure-ad-tenant
  tenantId: azure-tenant-uuid
  oidc:
    issuerURL: https://login.microsoftonline.com/tenant-id/v2.0
    secret:
      name: azure-oidc-secret
    audience: api://tempo-backend
    requiredClaims:
      - name: groups
        value: tempo-users
```

## Advanced Static Multitenancy Patterns

### 1. **Environment-Based Tenants**

#### Development/Staging/Production
```yaml
authentication:
- tenantName: development
  tenantId: dev-env
  oidc:
    issuerURL: https://dev-auth.company.com
    secret:
      name: dev-oidc-secret
- tenantName: staging
  tenantId: staging-env
  oidc:
    issuerURL: https://staging-auth.company.com
    secret:
      name: staging-oidc-secret
- tenantName: production
  tenantId: prod-env
  oidc:
    issuerURL: https://prod-auth.company.com
    secret:
      name: prod-oidc-secret

authorization:
  roles:
  - name: developer
    permissions: [read, write]
    resources: [traces]
    tenants: [development, staging]
  - name: operator
    permissions: [read]
    resources: [traces]
    tenants: [development, staging, production]
```

### 2. **Service-Based Tenants**

#### Microservice Isolation
```yaml
authentication:
- tenantName: user-service
  tenantId: user-service-uuid
  oidc:
    issuerURL: https://auth.company.com
    secret:
      name: user-service-oidc-secret
- tenantName: payment-service
  tenantId: payment-service-uuid
  oidc:
    issuerURL: https://auth.company.com
    secret:
      name: payment-service-oidc-secret

authorization:
  roles:
  - name: service-owner
    permissions: [read, write, delete]
    resources: [traces]
    tenants: ["${TENANT_NAME}"]  # Self-access only
  - name: platform-team
    permissions: [read]
    resources: [traces]
    tenants: ["*"]  # Access to all services
```

### 3. **Customer-Based Tenants**

#### SaaS Multi-Tenancy
```yaml
authentication:
- tenantName: customer-a
  tenantId: customer-a-uuid
  oidc:
    issuerURL: https://customer-a.auth.saas.com
    secret:
      name: customer-a-oidc-secret
- tenantName: customer-b
  tenantId: customer-b-uuid
  oidc:
    issuerURL: https://customer-b.auth.saas.com
    secret:
      name: customer-b-oidc-secret

authorization:
  roles:
  - name: customer-access
    permissions: [read, write]
    resources: [traces]
    tenants: ["${CUSTOMER_TENANT}"]
  roleBindings:
  - name: customer-a-binding
    roles: [customer-access]
    subjects:
    - kind: user
      name: customer-a-service
    tenantContext: customer-a
```

## Security Considerations

### 1. **OIDC Token Security**

#### Token Validation
```yaml
# Comprehensive token validation
spec:
  multitenancy:
    authentication:
    - tenantName: secure-tenant
      tenantId: secure-tenant-uuid
      oidc:
        issuerURL: https://secure-oidc.company.com
        secret:
          name: secure-oidc-secret
        requiredClaims:
        - name: aud
          value: tempo-api
        - name: scope
          value: read write
        clockSkew: 30s
        skipAudienceValidation: false
        skipIssuerValidation: false
```

#### Certificate Management
```yaml
# Custom CA for OIDC issuer
apiVersion: v1
kind: Secret
metadata:
  name: oidc-ca-secret
data:
  ca.crt: LS0tLS1CRUdJTi... # Base64 encoded CA certificate
---
spec:
  multitenancy:
    authentication:
    - tenantName: enterprise-tenant
      oidc:
        issuerURL: https://enterprise-oidc.internal.com
        secret:
          name: enterprise-oidc-secret
        caSecret:
          name: oidc-ca-secret
```

### 2. **Authorization Security**

#### Least Privilege Access
```yaml
authorization:
  roles:
  - name: read-only-analytics
    permissions: [read]
    resources: [traces]
    tenants: ["analytics-tenant"]
    conditions:
    - field: trace.duration
      operator: ">"
      value: "100ms"  # Only slow traces
  - name: write-service-traces
    permissions: [write]
    resources: [traces]
    tenants: ["service-tenant"]
    conditions:
    - field: service.name
      operator: "=="
      value: "${CLIENT_ID}"  # Only own service traces
```

#### Time-Based Access
```yaml
authorization:
  roles:
  - name: business-hours-access
    permissions: [read, write]
    resources: [traces]
    tenants: ["business-tenant"]
    timeRestrictions:
    - days: [monday, tuesday, wednesday, thursday, friday]
      startTime: "08:00"
      endTime: "18:00"
      timezone: "America/New_York"
```

### 3. **Secret Management**

#### Secret Rotation
```bash
# Automated secret rotation
oc create secret generic new-oidc-secret \
  --from-literal=clientID=new-tenant-client \
  --from-literal=clientSecret=new-secure-secret

# Update TempoMonolithic to use new secret
oc patch tempomonolithic sample --type='merge' -p='
spec:
  multitenancy:
    authentication:
    - tenantName: tenant1
      oidc:
        secret:
          name: new-oidc-secret'
```

#### Secret Validation
```bash
# Validate OIDC secrets
oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientID}' | base64 -d
oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientSecret}' | base64 -d

# Test OIDC connectivity
curl -X POST http://hydra:4444/oauth2/token \
  -u "$(oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientID}' | base64 -d):$(oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientSecret}' | base64 -d)" \
  -d "grant_type=client_credentials"
```

## Troubleshooting Static OIDC Issues

### 1. **OIDC Authentication Failures**

#### Token Acquisition Problems
```bash
# Test OAuth2 client credentials flow
CLIENT_ID=$(oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientID}' | base64 -d)
CLIENT_SECRET=$(oc get secret tenant1-oidc-secret -o jsonpath='{.data.clientSecret}' | base64 -d)

curl -v -X POST http://hydra:4444/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET"

# Check for common errors:
# - invalid_client: Check client ID/secret
# - unsupported_grant_type: Verify grant types in Hydra
# - server_error: Check Hydra logs
```

#### JWT Token Validation Issues
```bash
# Decode JWT token for inspection
TOKEN="eyJhbGciOiJSUzI1NiIs..."  # Replace with actual token
echo $TOKEN | cut -d. -f2 | base64 -d | jq .

# Verify token signature
curl -s http://hydra:4444/.well-known/jwks.json | jq .

# Check token expiration
echo $TOKEN | cut -d. -f2 | base64 -d | jq .exp | xargs -I {} date -d @{}
```

### 2. **Authorization Configuration Issues**

#### Role Binding Problems
```bash
# Check TempoMonolithic authorization configuration
oc get tempomonolithic sample -o yaml | yq '.spec.multitenancy.authorization'

# Verify role binding subjects match OIDC client
oc get tempomonolithic sample -o jsonpath='{.spec.multitenancy.authorization.roleBindings[*].subjects[*].name}'

# Check for role assignment errors in gateway logs
oc logs -l app.kubernetes.io/component=gateway | grep -i "authorization\|role\|denied"
```

#### Permission Validation
```bash
# Test authorization with token
TOKEN=$(curl -s -X POST http://hydra:4444/oauth2/token \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d "grant_type=client_credentials" | jq -r .access_token)

# Test trace query with token
curl -v -G \
  -H "Authorization: Bearer $TOKEN" \
  https://tempo-sample-gateway:8080/api/search/tenant1 \
  --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  --data-urlencode "q={}"
```

### 3. **Hydra Integration Issues**

#### Hydra Connectivity
```bash
# Check Hydra service availability
oc get service hydra
oc get endpoints hydra

# Test Hydra endpoints
curl -v http://hydra:4444/.well-known/openid_configuration
curl -v http://hydra:4445/health/ready

# Check Hydra logs
oc logs deployment/hydra | grep -i error
```

#### Client Registration Validation
```bash
# Verify OAuth2 client exists in Hydra
curl -s http://hydra:4445/admin/clients/tenant1-oidc-client | jq .

# List all registered clients
curl -s http://hydra:4445/admin/clients | jq '.[] | {client_id, grant_types}'

# Check client configuration
curl -s http://hydra:4445/admin/clients/tenant1-oidc-client | jq '{client_id, grant_types, token_endpoint_auth_method}'
```

## Production Deployment Best Practices

### 1. **OIDC Provider Configuration**

#### Production Hydra Setup
```yaml
# Production Hydra with persistent storage
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hydra-production
spec:
  template:
    spec:
      containers:
      - name: hydra
        image: docker.io/oryd/hydra:v2.2.0
        env:
        - name: DSN
          value: postgres://user:password@postgres:5432/hydra
        - name: SECRETS_SYSTEM
          valueFrom:
            secretKeyRef:
              name: hydra-secrets
              key: system-secret
        - name: URLS_SELF_ISSUER
          value: https://auth.company.com
        - name: STRATEGIES_ACCESS_TOKEN
          value: jwt
        - name: TTL_ACCESS_TOKEN
          value: 1h
        - name: TTL_REFRESH_TOKEN
          value: 24h
```

#### Enterprise Identity Provider
```yaml
# Integration with enterprise SSO
authentication:
- tenantName: enterprise-users
  tenantId: enterprise-tenant-uuid
  oidc:
    issuerURL: https://sso.company.com/realms/production
    secret:
      name: enterprise-oidc-secret
    audience: tempo-production
    requiredClaims:
    - name: groups
      value: tempo-users
    - name: department
      values: [engineering, operations, security]
```

### 2. **Security Hardening**

#### Certificate Management
```yaml
# Use cert-manager for automatic certificate management
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: hydra-tls
spec:
  secretName: hydra-tls-secret
  issuerRef:
    name: company-ca-issuer
    kind: ClusterIssuer
  dnsNames:
  - auth.company.com
  - hydra.company.internal
```

#### Network Security
```yaml
# Network policies for OIDC security
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-oidc-access
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: tempo-monolithic
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: otel-collector
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: hydra
    ports:
    - protocol: TCP
      port: 4444
```

### 3. **Monitoring and Alerting**

#### OIDC Metrics
```yaml
# Monitor OIDC authentication metrics
alert: OIDCAuthenticationFailure
expr: rate(tempo_gateway_oidc_auth_failures_total[5m]) > 0.1
for: 2m
annotations:
  summary: "High OIDC authentication failure rate"

alert: OIDCTokenExpiration
expr: tempo_gateway_oidc_token_expiry_seconds < 300
for: 1m
annotations:
  summary: "OIDC tokens expiring soon"
```

#### Hydra Monitoring
```yaml
alert: HydraUnavailable
expr: up{job="hydra"} == 0
for: 1m
annotations:
  summary: "Hydra OIDC provider is unavailable"

alert: HydraHighLatency
expr: http_request_duration_seconds{job="hydra"} > 5
for: 5m
annotations:
  summary: "High latency in Hydra responses"
```

## Related Configurations

- [OpenShift Native Multitenancy](../monolithic-multitenancy-openshift/README.md) - OpenShift-integrated multitenancy
- [RBAC Multitenancy](../monolithic-multitenancy-rbac/README.md) - RBAC-based tenant isolation
- [TempoStack Static Auth](../multitenancy/README.md) - Distributed static authentication

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monolithic-multitenancy-static
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires a fixed namespace (`chainsaw-monolithic-multitenancy-static`) due to TLS certificate CN field requirements. The test demonstrates integration with external OIDC providers and static tenant configuration for enterprise environments.

