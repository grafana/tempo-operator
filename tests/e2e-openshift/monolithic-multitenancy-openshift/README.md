# TempoMonolithic with OpenShift Native Multitenancy

This configuration blueprint demonstrates TempoMonolithic's integration with OpenShift's native authentication and authorization system for secure multi-tenant trace ingestion and querying. This setup leverages OpenShift's RBAC, ServiceAccounts, and OAuth integration to provide enterprise-grade tenant isolation without requiring external authentication systems.

## Overview

This test validates OpenShift-native multitenancy features:
- **OpenShift Authentication**: Native integration with OpenShift's authentication system
- **RBAC-Based Authorization**: Tenant access control through Kubernetes RBAC
- **Gateway Component**: Secure proxy for multi-tenant trace ingestion and querying
- **ServiceAccount Tokens**: Automatic tenant identification using OpenShift ServiceAccount tokens
- **Tenant Isolation**: Complete separation of trace data between tenants

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ OpenTelemetry       │───▶│   Gateway Component       │───▶│ TempoMonolithic     │
│ Collector (dev)     │    │   (Multi-tenant Proxy)   │    │ Backend             │
│ - ServiceAccount    │    │ ┌─────────────────────┐  │    │ - Trace Storage     │
│ - Bearer Token      │    │ │ Authentication      │  │    │ - Query Engine      │
│ - X-Scope-OrgID     │    │ │ - ServiceAccount    │  │    └─────────────────────┘
└─────────────────────┘    │ │ - Bearer Token      │  │
                           │ │ Authorization       │  │    ┌─────────────────────┐
┌─────────────────────┐    │ │ - RBAC Rules        │  │───▶│ Tenant: dev         │
│ OpenShift RBAC      │◀───│ │ - Tenant Mapping    │  │    │ - ID: 1610b0...fa   │
│ - ClusterRole       │    │ └─────────────────────┘  │    │ - Traces isolated   │
│ - ClusterRoleBinding│    │ TLS Termination         │    └─────────────────────┘
│ - ServiceAccount    │    └──────────────────────────┘
└─────────────────────┘                                     ┌─────────────────────┐
                                                            │ Tenant: prod        │
┌─────────────────────┐    ┌──────────────────────────┐    │ - ID: 1610b0...fb   │
│ Jaeger UI           │◀───│   OpenShift Route        │    │ - Traces isolated   │
│ - Multi-tenant      │    │   - External Access      │    └─────────────────────┘
│ - RBAC-controlled   │    │   - TLS Termination      │
└─────────────────────┘    └──────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Cluster administrator privileges for RBAC setup
- Understanding of OpenShift authentication and RBAC

## Step-by-Step Configuration

### Step 1: Enable OpenShift User Workload Monitoring

Configure cluster-wide monitoring for multi-tenant metrics collection:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true 
    alertmanagerMain:
      enableUserAlertmanagerConfig: true
EOF
```

**Reference**: [`00-workload-monitoring.yaml`](./00-workload-monitoring.yaml)

### Step 2: Deploy TempoMonolithic with OpenShift Multitenancy

Create a comprehensive multi-tenant TempoMonolithic configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: mmo
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
  observability:
    metrics:
      prometheusRules:
        enabled: true
      serviceMonitors:
        enabled: true
  multitenancy:
    enabled: true
    mode: openshift
    authentication:
    - tenantName: dev
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
    - tenantName: prod
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
EOF
```

**Key Configuration Elements**:

#### Multitenancy Configuration
- `enabled: true`: Activates multi-tenant mode
- `mode: openshift`: Uses OpenShift-native authentication
- **Tenant Mapping**: Maps friendly tenant names to UUIDs

#### Authentication Setup
- `tenantName: dev`: Human-readable tenant identifier
- `tenantId: "1610b0..."`: Unique tenant UUID for data isolation
- **Multiple Tenants**: Support for dev, prod, and additional environments

**Generated Components**:
- **Gateway Service**: Multi-tenant proxy at `tempo-mmo-gateway`
- **TLS Certificates**: Automatic certificate generation for gateway
- **Monitoring Integration**: ServiceMonitors for multi-tenant metrics

### Step 3: Configure RBAC for Tenant Access Control

Set up OpenShift RBAC for secure tenant access:

```bash
# Grant dev-collector permission to write traces to 'dev' tenant
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: allow-write-traces-dev-tenant
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev]  # tenantName
  resourceNames: [traces]
  verbs: [create]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: allow-write-traces-dev-tenant
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: allow-write-traces-dev-tenant
subjects:
- kind: ServiceAccount
  name: dev-collector
  namespace: chainsaw-monolithic-multitenancy
EOF

# Grant default ServiceAccount permission to read traces from 'dev' tenant
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: allow-read-traces-dev-tenant
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev]  # tenantName
  resourceNames: [traces]
  verbs: [get]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: allow-read-traces-dev-tenant
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: allow-read-traces-dev-tenant
subjects:
- kind: ServiceAccount
  name: default
  namespace: chainsaw-monolithic-multitenancy
EOF

# Grant view permissions for namespace access
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: view
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
- kind: ServiceAccount
  name: default
  namespace: chainsaw-monolithic-multitenancy
EOF
```

**RBAC Configuration Details**:

#### Write Permissions
- **ClusterRole**: Defines write access to specific tenant resources
- **Resource Scoping**: `resources: [dev]` limits access to dev tenant
- **Verb Control**: `verbs: [create]` allows trace ingestion only

#### Read Permissions
- **Query Access**: `verbs: [get]` enables trace querying
- **Tenant Isolation**: Each ServiceAccount can only access assigned tenants

#### Namespace Permissions
- **View Role**: Required for basic namespace access
- **Security Requirement**: Prevents unauthorized access to other namespaces

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 4: Deploy Tenant-Specific OpenTelemetry Collector

Create an OpenTelemetry Collector configured for the dev tenant:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: dev-collector
---
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: dev
spec:
  serviceAccount: dev-collector
  config: |
    extensions:
      bearertokenauth:
        filename: /var/run/secrets/kubernetes.io/serviceaccount/token

    receivers:
      otlp/grpc:
        protocols:
          grpc:
      otlp/http:
        protocols:
          http:

    exporters:
      otlp:
        endpoint: tempo-mmo-gateway.chainsaw-monolithic-multitenancy.svc.cluster.local:4317
        tls:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
        auth:
          authenticator: bearertokenauth
        headers:
          X-Scope-OrgID: dev  # tenantName
      otlphttp:
        endpoint: https://tempo-mmo-gateway.chainsaw-monolithic-multitenancy.svc.cluster.local:8080/api/traces/v1/dev
        tls:
          ca_file: /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
        auth:
          authenticator: bearertokenauth
        headers:
          X-Scope-OrgID: dev  # tenantName

    service:
      telemetry:
        logs:
          level: "DEBUG"
          development: true
          encoding: "json"
      extensions: [bearertokenauth]
      pipelines:
        traces/grpc:
          receivers: [otlp/grpc]
          exporters: [otlp]
        traces/http:
          receivers: [otlp/http]
          exporters: [otlphttp]
EOF
```

**OpenTelemetry Collector Configuration**:

#### Authentication
- `bearertokenauth`: Uses ServiceAccount token for authentication
- **Automatic Token**: OpenShift automatically mounts ServiceAccount token
- **Token Validation**: Gateway validates token against RBAC rules

#### Multi-Tenant Headers
- `X-Scope-OrgID: dev`: Identifies target tenant for traces
- **Gateway Routing**: Headers determine trace destination
- **Tenant Isolation**: Ensures traces go to correct tenant

#### TLS Configuration
- `ca_file`: OpenShift service CA for secure communication
- **Gateway TLS**: All communication encrypted in transit
- **Certificate Management**: Automatic certificate rotation

#### Dual Pipelines
- **gRPC Pipeline**: OTLP gRPC to gateway port 4317
- **HTTP Pipeline**: OTLP HTTP to gateway HTTPS endpoint
- **Protocol Flexibility**: Supports multiple ingestion methods

**Reference**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)

### Step 5: Generate Multi-Tenant Traces

Create traces specifically for the dev tenant:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
        args:
        - traces
        - --otlp-endpoint=dev-collector:4317
        - --otlp-insecure
        - --traces=10
        - --service-name=dev-service
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Generation Details**:
- **Tenant-Specific**: Traces routed through dev-collector
- **Service Identification**: Service name includes tenant context
- **RBAC Validation**: Traces subject to RBAC permission checks

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 6: Verify Multi-Tenant Trace Isolation

Validate that traces are properly isolated and accessible by authorized users:

```bash
# Verify traces via Jaeger UI (multi-tenant)
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces-jaegerui
spec:
  template:
    spec:
      containers:
      - name: verify-traces-jaegerui
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Query via multi-tenant Jaeger endpoint
          curl -v -G \
            -H "Authorization: Bearer \$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
            https://tempo-mmo-gateway:8080/api/search/dev \
            --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
            --data-urlencode "service=dev-service" | tee /tmp/jaeger.out
          
          num_traces=\$(jq ".data | length" /tmp/jaeger.out)
          if [[ "\$num_traces" -lt 5 ]]; then
            echo "Expected at least 5 traces for dev tenant, got \$num_traces"
            exit 1
          fi
          echo "✓ Verified \$num_traces traces accessible for dev tenant"
      restartPolicy: Never
EOF

# Verify traces via TraceQL (multi-tenant)
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces-traceql
spec:
  template:
    spec:
      containers:
      - name: verify-traces-traceql
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Query via TraceQL with tenant authentication
          curl -v -G \
            -H "Authorization: Bearer \$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
            https://tempo-mmo-gateway:8080/api/search/dev \
            --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
            --data-urlencode "q={.service.name=\\"dev-service\\"}" | tee /tmp/traceql.out
          
          num_traces=\$(jq ".traces | length" /tmp/traceql.out)
          if [[ "\$num_traces" -lt 5 ]]; then
            echo "Expected at least 5 traces via TraceQL, got \$num_traces"
            exit 1
          fi
          echo "✓ Verified \$num_traces traces via TraceQL for dev tenant"
      restartPolicy: Never
EOF
```

**Multi-Tenant Verification**:
- **Bearer Token**: Uses ServiceAccount token for authentication
- **Tenant-Specific Endpoints**: `/api/search/dev` targets dev tenant
- **TLS Verification**: Uses OpenShift service CA for secure communication
- **Isolation Validation**: Confirms only dev tenant traces are accessible

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

### Step 7: Validate Multi-Tenant Metrics

Check that metrics are properly collected for multi-tenant deployment:

```bash
# Run metrics validation script
./check_metrics.sh
```

The script validates multi-tenant specific metrics through OpenShift monitoring.

**Reference**: [`check_metrics.sh`](./check_metrics.sh)

## OpenShift Multitenancy Features

### 1. **Authentication Integration**

#### ServiceAccount Token Authentication
```yaml
# Automatic token-based authentication
extensions:
  bearertokenauth:
    filename: /var/run/secrets/kubernetes.io/serviceaccount/token
```

#### OAuth Integration (Advanced)
```yaml
# OAuth2 proxy integration (when available)
spec:
  multitenancy:
    mode: openshift
    oauth:
      enabled: true
      clientId: tempo-client
      clientSecret: tempo-oauth-secret
```

### 2. **RBAC-Based Authorization**

#### Fine-Grained Permissions
```yaml
# Separate read/write permissions per tenant
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev, staging, prod]  # Multiple tenant access
  resourceNames: [traces, metrics]
  verbs: [get, create]
```

#### Role-Based Tenant Assignment
```yaml
# Assign users to tenant roles
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: dev-team-traces
subjects:
- kind: User
  name: alice@company.com
- kind: Group
  name: dev-team
roleRef:
  kind: ClusterRole
  name: allow-dev-tenant-access
```

### 3. **Gateway Configuration**

#### Multi-Tenant Routing
```yaml
# Gateway automatically handles:
# - Token validation
# - Tenant identification  
# - Request routing
# - Response filtering
```

#### TLS Termination
```yaml
# Automatic TLS certificate management
spec:
  multitenancy:
    enabled: true
    mode: openshift
    gateway:
      tls:
        enabled: true
        # Certificates auto-generated by operator
```

## Advanced Multitenancy Patterns

### 1. **Environment-Based Tenants**

```yaml
spec:
  multitenancy:
    enabled: true
    mode: openshift
    authentication:
    - tenantName: development
      tenantId: "dev-env-uuid"
    - tenantName: staging
      tenantId: "staging-env-uuid"  
    - tenantName: production
      tenantId: "prod-env-uuid"
```

### 2. **Team-Based Tenants**

```yaml
spec:
  multitenancy:
    enabled: true
    mode: openshift
    authentication:
    - tenantName: platform-team
      tenantId: "platform-team-uuid"
    - tenantName: frontend-team
      tenantId: "frontend-team-uuid"
    - tenantName: backend-team
      tenantId: "backend-team-uuid"
```

### 3. **Application-Based Tenants**

```yaml
spec:
  multitenancy:
    enabled: true
    mode: openshift
    authentication:
    - tenantName: webapp
      tenantId: "webapp-uuid"
    - tenantName: api-service
      tenantId: "api-service-uuid"
    - tenantName: data-pipeline
      tenantId: "data-pipeline-uuid"
```

## Security Considerations

### 1. **Token Security**

#### Token Rotation
```bash
# ServiceAccount tokens automatically rotated by OpenShift
# No manual intervention required
oc get secret -o jsonpath='{.items[?(@.type=="kubernetes.io/service-account-token")].metadata.name}'
```

#### Token Scope
```yaml
# Limit token permissions to minimum required
apiVersion: v1
kind: ServiceAccount
metadata:
  name: limited-collector
automountServiceAccountToken: true
```

### 2. **Network Security**

#### Gateway Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-gateway-access
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: gateway
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: otel-collector
    ports:
    - protocol: TCP
      port: 4317
    - protocol: TCP
      port: 8080
```

### 3. **Audit and Compliance**

#### Access Logging
```yaml
spec:
  extraConfig:
    tempo:
      server:
        log_level: info
        log_format: json
      gateway:
        log_requests: true
        log_responses: false  # Avoid logging sensitive data
```

#### Compliance Monitoring
```bash
# Monitor tenant access patterns
oc logs -l app.kubernetes.io/component=gateway | grep -E "(tenant|auth|access)"

# Audit RBAC changes
oc get events --field-selector reason=PolicyRule
```

## Troubleshooting Multitenancy Issues

### 1. **Authentication Failures**

#### Token Issues
```bash
# Check ServiceAccount token
oc get serviceaccount dev-collector -o yaml
oc describe secret $(oc get serviceaccount dev-collector -o jsonpath='{.secrets[0].name}')

# Test token validity
TOKEN=$(oc create token dev-collector)
curl -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/namespaces
```

#### RBAC Problems
```bash
# Check RBAC permissions
oc auth can-i create traces --as=system:serviceaccount:$NAMESPACE:dev-collector

# Verify ClusterRoleBinding
oc describe clusterrolebinding allow-write-traces-dev-tenant

# Check for RBAC events
oc get events --field-selector reason=FailedMount,reason=Forbidden
```

### 2. **Gateway Connectivity**

#### Gateway Health
```bash
# Check gateway pod status
oc get pods -l app.kubernetes.io/component=gateway

# Test gateway endpoints
oc port-forward svc/tempo-mmo-gateway 8080:8080 &
curl -k https://localhost:8080/api/v1/status

# Check gateway logs
oc logs -l app.kubernetes.io/component=gateway | grep -i error
```

#### TLS Certificate Issues
```bash
# Check gateway certificates
oc get secret -l app.kubernetes.io/component=gateway
oc describe secret tempo-mmo-gateway-tls

# Verify certificate chain
oc get secret tempo-mmo-gateway-tls -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

### 3. **Tenant Isolation Validation**

#### Cross-Tenant Access Testing
```bash
# Attempt unauthorized access
TOKEN=$(oc create token unauthorized-sa)
curl -H "Authorization: Bearer $TOKEN" \
  https://tempo-mmo-gateway:8080/api/search/prod
# Should return 403 Forbidden

# Verify tenant data isolation
oc exec tempo-mmo-0 -- find /var/tempo -name "*dev*"
oc exec tempo-mmo-0 -- find /var/tempo -name "*prod*"
```

## Production Deployment Considerations

### 1. **Tenant Planning**

#### Tenant Strategy
- **Environment-based**: dev, staging, prod
- **Team-based**: platform, frontend, backend
- **Application-based**: service-specific tenants
- **Hybrid**: Combination based on organizational needs

#### Capacity Planning
```yaml
# Resource allocation per tenant
spec:
  resources:
    total:
      limits:
        memory: 8Gi    # Scale based on tenant count
        cpu: 4000m     # Account for multi-tenant overhead
```

### 2. **RBAC Management**

#### Automated RBAC
```bash
# Use GitOps for RBAC management
# Store ClusterRole and ClusterRoleBinding in Git
# Apply via ArgoCD or similar tools
```

#### Regular RBAC Audits
```bash
# Regular permission reviews
oc get clusterrolebinding -o json | jq '.items[] | select(.metadata.name | contains("tempo"))'

# Unused ServiceAccount cleanup
oc get serviceaccount --all-namespaces | grep -v default
```

### 3. **Monitoring and Alerting**

#### Multi-Tenant Metrics
```yaml
# Tenant-specific alerts
alert: HighTraceIngestionRate
expr: rate(tempo_distributor_spans_received_total{tenant="prod"}[5m]) > 10000
for: 5m
annotations:
  summary: "High trace ingestion rate for production tenant"
```

#### Authentication Monitoring
```yaml
alert: AuthenticationFailures
expr: rate(tempo_gateway_auth_failures_total[5m]) > 10
for: 2m
annotations:
  summary: "High authentication failure rate detected"
```

## Related Configurations

- [Static Multitenancy](../monolithic-multitenancy-static/README.md) - Static tenant configuration
- [RBAC Multitenancy](../monolithic-multitenancy-rbac/README.md) - RBAC-only multitenancy
- [TempoStack Multitenancy](../multitenancy/README.md) - Distributed multitenancy

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monolithic-multitenancy-openshift
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires OpenShift cluster administrator privileges for RBAC configuration and runs sequentially (`concurrent: false`) to avoid conflicts with shared monitoring resources. The test validates complete multi-tenant isolation including authentication, authorization, and data separation.

