# OpenShift Multi-Tenant TempoStack with RBAC and Monitoring

This configuration blueprint demonstrates how to deploy a production-ready, multi-tenant Tempo observability stack on OpenShift with comprehensive RBAC controls, workload monitoring, and tenant isolation. This setup showcases enterprise-grade observability with secure tenant separation and integrated monitoring.

## Overview

This test validates a complete multi-tenant observability stack featuring:
- **Multi-Tenant TempoStack**: Secure tenant isolation with per-tenant retention and limits
- **OpenShift RBAC Integration**: Fine-grained access control using OpenShift users and groups
- **Gateway-based Authentication**: Token-based authentication with tenant routing
- **OpenTelemetry Collector**: Tenant-aware trace collection and forwarding
- **User Workload Monitoring**: Integrated Prometheus metrics and alerting
- **Per-Tenant Configuration**: Customizable retention policies and ingestion limits

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OTel Collector  │───▶│   Gateway + Auth     │───▶│   TempoStack    │
│ (dev tenant)    │    │ ┌─────────────────┐  │    │ ┌─────────────┐ │
│ - Bearer Token  │    │ │ X-Scope-OrgID   │  │    │ │ Multi-Tenant│ │
│ - TLS Enabled   │    │ │ Header Routing  │  │    │ │ Components  │ │
└─────────────────┘    │ │ RBAC Validation │  │    │ └─────────────┘ │
                       │ └─────────────────┘  │    └─────────────────┘
┌─────────────────┐    └──────────────────────┘              │
│ Prometheus      │◀──────────────────────────────────────────┘
│ User Workload   │    ┌──────────────────────┐
│ Monitoring      │    │ MinIO Object Storage │
└─────────────────┘    │ (S3 Compatible)      │
                       └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Cluster admin privileges for RBAC setup
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Enable User Workload Monitoring

Configure OpenShift to monitor user-defined workloads:

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

**Configuration Details**:
- `enableUserWorkload`: Enables monitoring for user namespaces
- `enableUserAlertmanagerConfig`: Allows user-defined alerting rules

**Reference**: [`00-workload-monitoring.yaml`](./00-workload-monitoring.yaml)

### Step 2: Deploy MinIO Object Storage

Deploy storage backend with persistent volume:

```bash
# Apply storage configuration from compatibility test
kubectl apply -f ../compatibility/00-install-storage.yaml -n chainsaw-multitenancy
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 3: Deploy Multi-Tenant TempoStack

Create the TempoStack with multi-tenancy and RBAC configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
  namespace: chainsaw-multitenancy
spec:
  observability:
    metrics:
      createPrometheusRules: true
      createServiceMonitors: true
  retention:
    global:
      traces: 20h
    perTenant:
      dev:
        traces: 10h
  limits:
    perTenant:
      dev:
        ingestion:
          ingestionBurstSizeBytes: 1000000
        query:
          maxSearchDuration: 1h
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 3Gi
        cpu: 2000m
  tenants:
    mode: openshift
    authentication:
      - tenantName: dev
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
      - tenantName: prod
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
  template:
    gateway:
      enabled: true
    queryFrontend:
      jaegerQuery:
        enabled: true
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tempostack-traces-reader
rules:
  - apiGroups:
      - 'tempo.grafana.com'
    resources:
      - dev
    resourceNames:
      - traces
    verbs:
      - 'get'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tempostack-traces-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tempostack-traces-reader
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: system:authenticated
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: view
  namespace: chainsaw-multitenancy
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
- kind: ServiceAccount
  name: default
  namespace: chainsaw-multitenancy
EOF
```

**Key Configuration Details**:

#### Multi-Tenancy Settings
- `tenants.mode: openshift`: Uses OpenShift RBAC for authorization
- `tenants.authentication`: Defines tenant mappings with unique IDs
- `gateway.enabled: true`: Enables authentication gateway

#### Per-Tenant Configuration
- **Retention Policies**: Different trace retention per tenant (dev: 10h, global: 20h)
- **Ingestion Limits**: Burst size controls for tenant `dev`
- **Query Limits**: Maximum search duration restrictions

#### Monitoring Integration
- `createPrometheusRules: true`: Creates alerting rules
- `createServiceMonitors: true`: Exposes metrics to Prometheus

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 4: Deploy OpenTelemetry Collector

Create tenant-specific collector with authentication:

```bash
oc apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: dev
  namespace: chainsaw-multitenancy
spec:
  config: |
    extensions:
      bearertokenauth:
        filename: "/var/run/secrets/kubernetes.io/serviceaccount/token"

    receivers:
      otlp/grpc:
        protocols:
          grpc:
      otlp/http:
        protocols:
          http:

    processors:

    exporters:
      otlp:
        endpoint: tempo-simplest-gateway.chainsaw-multitenancy.svc.cluster.local:8090
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
        auth:
          authenticator: bearertokenauth
        headers:
          X-Scope-OrgID: "dev"
      otlphttp:
        endpoint: https://tempo-simplest-gateway.chainsaw-multitenancy.svc.cluster.local:8080/api/traces/v1/dev
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
        auth:
          authenticator: bearertokenauth
        headers:
          X-Scope-OrgID: "dev"

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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tempostack-traces-write
rules:
  - apiGroups:
      - 'tempo.grafana.com'
    resources:
      - dev
    resourceNames:
      - traces
    verbs:
      - 'create'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tempostack-traces
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tempostack-traces-write
subjects:
  - kind: ServiceAccount
    name: dev-collector
    namespace: chainsaw-multitenancy
EOF
```

**Key Configuration Details**:

#### Authentication
- `bearertokenauth`: Uses service account token for authentication
- `ca_file`: Uses OpenShift service CA for TLS verification

#### Tenant Routing
- `X-Scope-OrgID: "dev"`: Routes traces to `dev` tenant
- Gateway endpoints for both gRPC and HTTP protocols

#### RBAC for Trace Ingestion
- `tempostack-traces-write`: Allows trace creation for `dev` tenant
- Service account binding for collector authentication

**Reference**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)

### Step 5: Generate Sample Traces

Create traces using the configured collector:

```bash
oc apply -f - <<EOF
# Trace generation job that sends to the OTel collector
# Reference: 03-generate-traces.yaml
EOF
```

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 6: Verify Multi-Tenant Trace Access

Test tenant isolation and RBAC:

```bash
oc apply -f - <<EOF
# Verification job that tests tenant-specific trace access
# Reference: 04-verify-traces.yaml
EOF
```

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

### Step 7: Validate Monitoring Metrics

Run the metrics validation script:

```bash
./check_metrics.sh
```

This script validates the presence of key Tempo metrics:
- `tempo_query_frontend_queries_total`
- `tempo_request_duration_seconds_*`
- `tempo_build_info`
- `tempo_ingester_bytes_received_total`
- `tempo_operator_tempostack_*`

**Reference**: [`check_metrics.sh`](./check_metrics.sh)

## Key Features Demonstrated

### 1. **Multi-Tenant Architecture**
- **Tenant Isolation**: Complete separation of trace data between tenants
- **Gateway Authentication**: Centralized authentication and authorization
- **Per-Tenant Configuration**: Individual retention and limit policies

### 2. **OpenShift RBAC Integration**
- **ClusterRole Definitions**: Fine-grained permissions for trace operations
- **Service Account Authentication**: Token-based access control
- **Group-based Access**: Integration with OpenShift user groups

### 3. **Production Monitoring**
- **ServiceMonitor Creation**: Automatic Prometheus scraping configuration
- **PrometheusRule Generation**: Built-in alerting rules
- **User Workload Monitoring**: Integration with OpenShift monitoring stack

### 4. **Secure Communication**
- **TLS Everywhere**: Encrypted communication between all components
- **Certificate Management**: Automatic CA certificate handling
- **Token-based Auth**: Service account token authentication

## Tenant Management

### Adding New Tenants

1. **Update TempoStack Configuration**:
```yaml
tenants:
  authentication:
    - tenantName: staging
      tenantId: "new-unique-tenant-id"
```

2. **Create Tenant-Specific RBAC**:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tempostack-traces-staging
rules:
  - apiGroups: ['tempo.grafana.com']
    resources: [staging]
    resourceNames: [traces]
    verbs: ['get', 'create']
```

3. **Configure OTel Collector**:
```yaml
headers:
  X-Scope-OrgID: "staging"
```

### Per-Tenant Limits

Configure different limits for each tenant:

```yaml
limits:
  perTenant:
    dev:
      ingestion:
        ingestionBurstSizeBytes: 1000000
        ingestionRateLimitBytes: 500000
      query:
        maxSearchDuration: 1h
        maxTracesPerTag: 100
    prod:
      ingestion:
        ingestionBurstSizeBytes: 5000000
        ingestionRateLimitBytes: 2000000
      query:
        maxSearchDuration: 6h
        maxTracesPerTag: 500
```

### Retention Policies

Set different retention periods:

```yaml
retention:
  global:
    traces: 30d  # Default for all tenants
  perTenant:
    dev:
      traces: 7d   # Short retention for dev
    prod:
      traces: 90d  # Long retention for prod
```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Ingestion Metrics**:
   - `tempo_ingester_bytes_received_total`
   - `tempo_ingester_failed_flushes_total`

2. **Query Metrics**:
   - `tempo_query_frontend_queries_total`
   - `tempo_request_duration_seconds`

3. **Operator Metrics**:
   - `tempo_operator_tempostack_managed`
   - `tempo_operator_tempostack_storage_backend`

### Accessing Metrics

```bash
# Create service account for metrics access
oc create serviceaccount metrics-reader -n chainsaw-multitenancy
oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:chainsaw-multitenancy:metrics-reader

# Get access token
TOKEN=$(oc create token metrics-reader -n chainsaw-multitenancy)
THANOS_HOST=$(oc get route thanos-querier -n openshift-monitoring -o jsonpath='{.spec.host}')

# Query metrics
curl -k -H "Authorization: Bearer $TOKEN" \
  "https://$THANOS_HOST/api/v1/query?query=tempo_ingester_bytes_received_total"
```

## Troubleshooting

### Check Multi-Tenancy Configuration
```bash
oc describe tempostack simplest -n chainsaw-multitenancy
```

### Verify Gateway Authentication
```bash
oc logs -l app.kubernetes.io/component=gateway -n chainsaw-multitenancy
```

### Test Tenant Access
```bash
# Port-forward to gateway
oc port-forward svc/tempo-simplest-gateway 8080:8080 -n chainsaw-multitenancy

# Test with proper tenant header
curl -H "X-Scope-OrgID: dev" \
     -H "Authorization: Bearer $(oc whoami -t)" \
     "https://localhost:8080/api/search?q={}"
```

### Validate RBAC Permissions
```bash
oc auth can-i get traces --as=system:serviceaccount:chainsaw-multitenancy:dev-collector --as-group=tempo.grafana.com/dev
```

### Monitor Component Health
```bash
# Check all Tempo components
oc get pods -l app.kubernetes.io/managed-by=tempo-operator -n chainsaw-multitenancy

# Check OTel collector status
oc get opentelemetrycollector dev -n chainsaw-multitenancy -o yaml
```

## Production Considerations

### 1. **Resource Planning**
- Plan resources based on expected trace volume per tenant
- Monitor ingestion rates and adjust limits accordingly
- Consider horizontal scaling for high-traffic tenants

### 2. **Security Hardening**
- Use proper TLS certificates in production
- Implement network policies for traffic isolation
- Regular rotation of service account tokens

### 3. **Storage Management**
- Use enterprise object storage (AWS S3, GCS, Azure Blob)
- Implement backup and disaster recovery procedures
- Monitor storage usage and costs per tenant

### 4. **Monitoring Setup**
- Configure alerts for ingestion failures
- Monitor per-tenant resource usage
- Set up SLA monitoring for query performance

## Related Configurations

- [TempoStack Compatibility](../../e2e/compatibility/README.md) - Basic TempoStack setup
- [Monitoring Setup](../monitoring/README.md) - Detailed monitoring configuration
- [RBAC Configuration](../multitenancy-rbac/README.md) - Advanced RBAC patterns
- [AWS Object Store](../../e2e-openshift-object-stores/tempostack-aws/README.md) - Cloud storage integration

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/multitenancy --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs with `concurrent: false` to prevent conflicts with shared OpenShift monitoring resources.