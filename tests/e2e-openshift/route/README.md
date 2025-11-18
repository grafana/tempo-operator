# TempoStack with OpenShift Route and Must-Gather Validation

This configuration blueprint demonstrates deploying a distributed TempoStack with OpenShift Route for external access and validates comprehensive must-gather functionality for distributed Tempo deployments. This setup provides scalable, multi-component trace processing with native OpenShift routing and operational tooling for production support scenarios.

## Overview

This test validates distributed TempoStack with OpenShift integration:
- **Distributed Architecture**: Multi-component TempoStack deployment
- **OpenShift Route**: Native external access to Jaeger query interface
- **Persistent Storage**: MinIO S3-compatible backend with persistent volumes
- **Must-Gather Validation**: Comprehensive data collection for distributed components
- **Production Readiness**: Scalable deployment with operational tooling

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ External Users          │───▶│   OpenShift Route        │───▶│ Query Frontend          │
│ - Web Browsers          │    │   - HAProxy Router       │    │ - Jaeger UI             │
│ - Direct Access         │    │   - TLS Termination      │    │ - Query Aggregation     │
│ - HTTPS/HTTP            │    │   - Load Balancing       │    │ - Load Distribution     │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘
                                                               ┌─────────────────────────┐
┌─────────────────────────┐    ┌──────────────────────────┐    │ Distributed Components │
│ Trace Ingestion         │───▶│   Distributor            │───▶│ ┌─────────────────────┐ │
│ - OTLP gRPC/HTTP        │    │   - Trace Reception      │    │ │ Ingester            │ │
│ - Jaeger formats        │    │   - Load Balancing       │    │ │ - Trace Storage     │ │
│ - OpenTelemetry         │    │   - Preprocessing        │    │ │ - Block Creation    │ │
└─────────────────────────┘    └──────────────────────────┘    │ └─────────────────────┘ │
                                                               │ ┌─────────────────────┐ │
┌─────────────────────────┐    ┌──────────────────────────┐    │ │ Querier             │ │
│ MinIO Storage           │◀───│   Storage Backend        │◀───│ │ - Trace Retrieval   │ │
│ - S3 Compatible         │    │   - Block Storage        │    │ │ - Query Processing  │ │
│ - Persistent Volumes    │    │   - Object Store         │    │ └─────────────────────┘ │
│ - Bucket: tempo         │    │   - Compaction           │    │ ┌─────────────────────┐ │
└─────────────────────────┘    └──────────────────────────┘    │ │ Compactor           │ │
                                                               │ │ - Block Compaction  │ │
┌─────────────────────────┐    ┌──────────────────────────┐    │ │ - Retention Policy  │ │
│ Must-Gather Tool        │───▶│   Distributed Data       │    │ └─────────────────────┘ │
│ - All Components        │    │   Collection             │    └─────────────────────────┘
│ - Configuration Dump    │    │ ┌─────────────────────┐  │
│ - Troubleshooting Info  │    │ │ Per-Component Data  │  │
└─────────────────────────┘    │ │ - Deployments       │  │
                               │ │ - StatefulSets      │  │
                               │ │ - Services          │  │
                               │ │ - ConfigMaps        │  │
                               │ │ - Routes            │  │
                               │ └─────────────────────┘  │
                               └──────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed via OLM
- Persistent volume support for MinIO
- Cluster administrator privileges for must-gather operations
- `oc` CLI access
- Understanding of distributed tracing architectures

## Step-by-Step Configuration

### Step 1: Deploy MinIO Storage Backend

Create persistent MinIO storage for the distributed TempoStack:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    app.kubernetes.io/name: minio
  name: minio
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: minio
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: minio
    spec:
      containers:
        - command:
            - /bin/sh
            - -c
            - |
              mkdir -p /storage/tempo && \
              minio server /storage
          env:
            - name: MINIO_ACCESS_KEY
              value: tempo
            - name: MINIO_SECRET_KEY
              value: supersecret
          image: quay.io/minio/minio:latest
          name: minio
          ports:
            - containerPort: 9000
          volumeMounts:
            - mountPath: /storage
              name: storage
      volumes:
        - name: storage
          persistentVolumeClaim:
            claimName: minio
---
apiVersion: v1
kind: Service
metadata:
  name: minio
spec:
  ports:
    - port: 9000
      protocol: TCP
      targetPort: 9000
  selector:
    app.kubernetes.io/name: minio
  type: ClusterIP
---
apiVersion: v1
kind: Secret
metadata:
  name: minio
stringData:
  endpoint: http://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**MinIO Configuration Details**:

#### Persistent Storage
- **PVC**: 2Gi persistent volume for durable trace storage
- **Access Mode**: ReadWriteOnce for single-node access
- **Recreation Strategy**: Ensures persistent data across pod restarts

#### S3 Compatibility
- **Bucket Setup**: Automatic `tempo` bucket creation
- **Access Credentials**: Standard S3-compatible authentication
- **Internal Endpoint**: Cluster-internal S3 API access

**Reference**: [`install-storage.yaml`](./install-storage.yaml)

### Step 2: Deploy Distributed TempoStack with OpenShift Route

Create a comprehensive distributed TempoStack with route-enabled query interface:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
   name: minio-test
stringData:
  endpoint: http://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
---
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  timeout: 2m
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 200M
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: route
          host: example.com
          annotations:
            example_annotation: example_value
EOF
```

**Key Configuration Elements**:

#### Distributed TempoStack Architecture
- `timeout: 2m`: Fast timeout for testing scenarios
- `storageSize: 200M`: Compact storage allocation for testing
- **Multi-Component**: Automatic deployment of all Tempo components

#### Storage Backend Integration
- `storage.secret.name: minio`: References MinIO storage secret
- `storage.secret.type: s3`: S3-compatible object storage
- **Block Storage**: Distributed block-based trace storage

#### OpenShift Route Configuration
```yaml
template:
  queryFrontend:
    jaegerQuery:
      enabled: true
      ingress:
        type: route
        host: example.com
        annotations:
          example_annotation: example_value
```

**Route Configuration Details**:
- `type: route`: Uses OpenShift Route instead of Kubernetes Ingress
- `host: example.com`: Custom hostname for external access
- **Annotations**: Custom route annotations for HAProxy configuration
- **Jaeger UI**: External access to distributed query interface

**Generated Distributed Components**:
- **Distributor**: Trace ingestion and load balancing (Deployment)
- **Ingester**: Trace storage and block creation (StatefulSet)
- **Query Frontend**: Query aggregation and Jaeger UI (Deployment)
- **Querier**: Trace retrieval and search (Deployment)
- **Compactor**: Block compaction and retention (Deployment)
- **Services**: Inter-component communication and external access
- **Route**: External access to query frontend

**Reference**: [`install-tempo.yaml`](./install-tempo.yaml)

### Step 3: Validate Distributed Deployment

Verify that all TempoStack components are properly deployed and ready:

```bash
# Check TempoStack readiness
oc get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify all distributed components
oc get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check individual component status
oc get deployment tempo-simplest-distributor
oc get statefulset tempo-simplest-ingester
oc get deployment tempo-simplest-query-frontend
oc get deployment tempo-simplest-querier
oc get deployment tempo-simplest-compactor

# Verify services and networking
oc get services -l app.kubernetes.io/managed-by=tempo-operator
oc get route tempo-simplest-query-frontend

# Check storage connectivity
oc get secret minio
oc get configmap tempo-simplest
```

Expected validation results:
- **All Components Ready**: 5 deployments/statefulsets in Ready state
- **Services**: Multiple services for inter-component communication
- **Route**: External route created for query frontend
- **Storage**: MinIO integration properly configured

### Step 4: Test External Route Access

Validate that the OpenShift Route provides proper external access:

```bash
# Get route information
ROUTE_HOST=$(oc get route tempo-simplest-query-frontend -o jsonpath='{.spec.host}')
echo "Jaeger UI accessible at: https://$ROUTE_HOST"

# Test route accessibility
curl -k https://$ROUTE_HOST/

# Verify route configuration details
oc describe route tempo-simplest-query-frontend

# Check route annotations
oc get route tempo-simplest-query-frontend -o yaml | grep -A5 annotations

# Test Jaeger UI functionality
curl -k https://$ROUTE_HOST/api/services
```

**Route Access Validation**:
- **External Hostname**: Proper DNS resolution and accessibility
- **TLS Termination**: Automatic HTTPS with OpenShift certificates
- **Jaeger UI**: Full functionality through external route
- **Custom Annotations**: Applied route customizations

### Step 5: Execute Comprehensive Must-Gather

Run the must-gather tool to collect data for the distributed deployment:

```bash
# Create temporary directory for must-gather output
MUST_GATHER_DIR=$(mktemp -d)

# Determine operator namespace
TEMPO_NAMESPACE=$(oc get pods -A \
  -l control-plane=controller-manager \
  -l app.kubernetes.io/name=tempo-operator \
  -o jsonpath='{.items[0].metadata.namespace}')

# Execute comprehensive must-gather
oc adm must-gather \
  --dest-dir=$MUST_GATHER_DIR \
  --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest \
  -- /usr/bin/must-gather --operator-namespace $TEMPO_NAMESPACE
```

**Distributed Must-Gather Collection**:
- **All Components**: Data for distributor, ingester, querier, query-frontend, compactor
- **Deployment Resources**: All Kubernetes resources for each component
- **Configuration**: Complete configuration for distributed setup
- **Networking**: Services, routes, and inter-component communication
- **Storage**: Object storage configuration and status

### Step 6: Validate Must-Gather Completeness for Distributed Components

Verify that all distributed components are properly captured:

```bash
# Define required items for distributed TempoStack
REQUIRED_ITEMS=(
  "event-filter.html"                    # Event analysis dashboard
  "timestamp"                            # Collection timestamp
  "*sha*/deployment-tempo-operator-controller.yaml"  # Operator deployment
  "*sha*/olm/installplan-install-*"                  # OLM install plans
  "*sha*/olm/clusterserviceversion-tempo-operator-*.yaml"  # CSV
  "*sha*/olm/operator-opentelemetry-product-openshift-opentelemetry-operator.yaml"  # OTel operator
  "*sha*/olm/operator-*-tempo-operator.yaml"         # Tempo operator OLM
  "*sha*/olm/subscription-tempo-*.yaml"              # OLM subscriptions
  
  # Distributed component resources
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-distributor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-ingester.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-distributor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-querier.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/configmap-tempo-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-compactor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-querier.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/tempostack-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/serviceaccount-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/statefulset-tempo-simplest-ingester.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/route-tempo-simplest-query-frontend.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-gossip-ring.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/configmap-tempo-simplest-ca-bundle.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/serviceaccount-tempo-simplest.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/deployment-tempo-simplest-compactor.yaml"
  "*sha*/namespaces/chainsaw-route/tempostack/simplest/service-tempo-simplest-query-frontend-discovery.yaml"
  
  # Operator logs and information
  "*sha*/tempo-operator-controller-*"
)

# Verify each distributed component is captured
for item in "${REQUIRED_ITEMS[@]}"; do
  if ! find "$MUST_GATHER_DIR" -path "$MUST_GATHER_DIR/$item" -print -quit | grep -q .; then
    echo "Missing: $item"
    exit 1
  else
    echo "Found: $item"
  fi
done

echo "✓ All distributed TempoStack components captured in must-gather"
```

**Distributed Component Validation**:

#### Core Services
- **Distributor Service**: Load balancing and trace ingestion
- **Ingester Service**: Trace storage and block management
- **Querier Service**: Trace retrieval and search
- **Query Frontend Service**: Query aggregation and UI
- **Compactor Service**: Block compaction and retention

#### Workload Resources
- **Deployments**: Stateless components (distributor, querier, query-frontend, compactor)
- **StatefulSet**: Stateful ingester component with persistent storage
- **ServiceAccounts**: Component-specific service accounts
- **ConfigMaps**: Configuration and certificate bundles

#### Networking and Access
- **Route**: OpenShift route for external access
- **Gossip Ring Service**: Inter-component communication
- **Discovery Service**: Service discovery for components
- **CA Bundle**: Certificate authority for internal TLS

**Reference**: [`check-must-gather.sh`](./check-must-gather.sh)

## Distributed TempoStack Features

### 1. **Component Architecture**

#### Distributor Component
```yaml
# Handles trace ingestion and load balancing
deployment-tempo-simplest-distributor:
  replicas: 1
  ports:
    - 4317 (OTLP gRPC)
    - 4318 (OTLP HTTP)
    - 14268 (Jaeger HTTP)
  responsibilities:
    - Trace reception
    - Load balancing to ingesters
    - Trace preprocessing
```

#### Ingester Component
```yaml
# Manages trace storage and block creation
statefulset-tempo-simplest-ingester:
  replicas: 1
  storage: persistent
  ports:
    - 3200 (HTTP API)
    - 7946 (Gossip)
  responsibilities:
    - Trace storage
    - Block creation
    - WAL management
```

#### Query Frontend Component
```yaml
# Provides query interface and aggregation
deployment-tempo-simplest-query-frontend:
  replicas: 1
  ports:
    - 3200 (HTTP API)
    - 16686 (Jaeger UI)
  responsibilities:
    - Query aggregation
    - Jaeger UI hosting
    - Query caching
```

#### Querier Component
```yaml
# Executes trace queries and searches
deployment-tempo-simplest-querier:
  replicas: 1
  ports:
    - 3200 (HTTP API)
  responsibilities:
    - Trace retrieval
    - Query processing
    - Block scanning
```

#### Compactor Component
```yaml
# Handles block compaction and retention
deployment-tempo-simplest-compactor:
  replicas: 1
  ports:
    - 3200 (HTTP API)
  responsibilities:
    - Block compaction
    - Retention enforcement
    - Storage optimization
```

### 2. **Inter-Component Communication**

#### Gossip Ring Protocol
- **Service**: `tempo-simplest-gossip-ring`
- **Purpose**: Component discovery and state sharing
- **Protocol**: Memberlist gossip protocol for coordination
- **Ports**: 7946 (gossip communication)

#### Service Discovery
- **Service**: `tempo-simplest-query-frontend-discovery`
- **Purpose**: Query frontend discovery for queriers
- **Load Balancing**: Automatic distribution of queries
- **Health Checking**: Component health monitoring

#### Configuration Distribution
- **ConfigMap**: `tempo-simplest`
- **Content**: Complete Tempo configuration for all components
- **Synchronization**: Automatic configuration updates across components

### 3. **Scaling and Performance**

#### Horizontal Scaling
```yaml
# Scale individual components independently
spec:
  template:
    distributor:
      replicas: 3
    ingester:
      replicas: 3
    querier:
      replicas: 2
    queryFrontend:
      replicas: 2
    compactor:
      replicas: 1  # Usually single instance
```

#### Resource Allocation
```yaml
# Component-specific resource allocation
spec:
  resources:
    total:
      limits:
        memory: 4Gi
        cpu: 2000m
  template:
    distributor:
      resources:
        limits:
          memory: 1Gi
          cpu: 500m
    ingester:
      resources:
        limits:
          memory: 2Gi
          cpu: 1000m
```

## Advanced Route Configuration

### 1. **Custom Route Annotations**

#### Performance Tuning
```yaml
spec:
  template:
    queryFrontend:
      jaegerQuery:
        ingress:
          type: route
          annotations:
            haproxy.router.openshift.io/timeout: 30s
            haproxy.router.openshift.io/balance: roundrobin
            haproxy.router.openshift.io/disable_cookies: "true"
```

#### Security Configuration
```yaml
spec:
  template:
    queryFrontend:
      jaegerQuery:
        ingress:
          type: route
          annotations:
            haproxy.router.openshift.io/hsts_header: max-age=31536000;includeSubDomains;preload
            haproxy.router.openshift.io/ip_whitelist: 10.0.0.0/8 192.168.0.0/16
```

#### Rate Limiting
```yaml
spec:
  template:
    queryFrontend:
      jaegerQuery:
        ingress:
          type: route
          annotations:
            haproxy.router.openshift.io/rate-limit-connections: "true"
            haproxy.router.openshift.io/rate-limit-connections.concurrent-tcp: "10"
            haproxy.router.openshift.io/rate-limit-connections.rate-http: "100"
```

### 2. **TLS Configuration**

#### Custom TLS Certificates
```yaml
spec:
  template:
    queryFrontend:
      jaegerQuery:
        ingress:
          type: route
          tls:
            termination: edge
            certificate: |
              -----BEGIN CERTIFICATE-----
              # Custom certificate content
              -----END CERTIFICATE-----
            key: |
              -----BEGIN PRIVATE KEY-----
              # Private key content
              -----END PRIVATE KEY-----
```

#### TLS Termination Options
```yaml
# Edge termination (default)
tls:
  termination: edge
  insecureEdgeTerminationPolicy: Redirect

# Passthrough termination
tls:
  termination: passthrough

# Re-encryption termination
tls:
  termination: reencrypt
  destinationCACertificate: |
    -----BEGIN CERTIFICATE-----
    # Destination CA certificate
    -----END CERTIFICATE-----
```

## Production Deployment Considerations

### 1. **Distributed Scaling Strategy**

#### Component Scaling Guidelines
```yaml
# Production scaling recommendations
spec:
  template:
    distributor:
      replicas: 3      # Scale based on ingestion load
    ingester:
      replicas: 3      # Scale for storage capacity
    querier:
      replicas: 3      # Scale for query load
    queryFrontend:
      replicas: 2      # Usually 2 for HA
    compactor:
      replicas: 1      # Single instance recommended
```

#### Resource Planning
```yaml
# Production resource allocation
spec:
  resources:
    total:
      limits:
        memory: 16Gi
        cpu: 8000m
  template:
    distributor:
      resources:
        requests:
          memory: 2Gi
          cpu: 1000m
        limits:
          memory: 4Gi
          cpu: 2000m
    ingester:
      resources:
        requests:
          memory: 4Gi
          cpu: 2000m
        limits:
          memory: 8Gi
          cpu: 4000m
```

### 2. **Storage Backend Configuration**

#### Production MinIO Setup
```yaml
# High-availability MinIO cluster
apiVersion: minio.min.io/v2
kind: Tenant
metadata:
  name: tempo-storage
spec:
  image: quay.io/minio/minio:latest
  pools:
  - servers: 4
    volumesPerServer: 4
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 100Gi
        storageClassName: fast-ssd
  requestAutoCert: true
```

#### External S3 Integration
```yaml
# Production S3 configuration
apiVersion: v1
kind: Secret
metadata:
  name: production-s3
stringData:
  endpoint: https://s3.amazonaws.com
  bucket: tempo-production-traces
  access_key_id: AKIA...
  access_key_secret: secret...
  region: us-west-2
```

### 3. **Monitoring and Observability**

#### Component-Level Monitoring
```yaml
# ServiceMonitor for each component
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tempo-distributed-components
spec:
  selector:
    matchLabels:
      app.kubernetes.io/managed-by: tempo-operator
  endpoints:
  - port: http
    interval: 30s
    path: /metrics
```

#### Distributed Tracing Alerts
```yaml
# Component-specific alerts
alert: TempoDistributorDown
expr: up{job="tempo-simplest-distributor"} == 0
for: 1m
annotations:
  summary: "Tempo distributor component is down"

alert: TempoIngesterHighMemory
expr: container_memory_usage_bytes{pod=~"tempo-simplest-ingester-.*"} / container_spec_memory_limit_bytes > 0.9
for: 10m
annotations:
  summary: "Tempo ingester memory usage is high"
```

## Troubleshooting Distributed TempoStack

### 1. **Component Health Validation**

#### Individual Component Status
```bash
# Check each component individually
oc get deployment tempo-simplest-distributor -o yaml | grep -A5 conditions
oc get statefulset tempo-simplest-ingester -o yaml | grep -A5 conditions
oc get deployment tempo-simplest-query-frontend -o yaml | grep -A5 conditions
oc get deployment tempo-simplest-querier -o yaml | grep -A5 conditions
oc get deployment tempo-simplest-compactor -o yaml | grep -A5 conditions

# Check component logs
oc logs deployment/tempo-simplest-distributor
oc logs statefulset/tempo-simplest-ingester
oc logs deployment/tempo-simplest-query-frontend
```

#### Inter-Component Communication
```bash
# Test gossip ring connectivity
oc exec tempo-simplest-ingester-0 -- curl http://localhost:3200/memberlist

# Verify service discovery
oc exec deployment/tempo-simplest-querier -- curl http://tempo-simplest-query-frontend-discovery:3200/ready

# Check gossip ring health
oc get service tempo-simplest-gossip-ring
oc describe endpoints tempo-simplest-gossip-ring
```

### 2. **Route and External Access Issues**

#### Route Troubleshooting
```bash
# Check route configuration
oc get route tempo-simplest-query-frontend -o yaml

# Verify route admits
oc describe route tempo-simplest-query-frontend | grep -A10 "Route Status"

# Test external connectivity
ROUTE_HOST=$(oc get route tempo-simplest-query-frontend -o jsonpath='{.spec.host}')
curl -v https://$ROUTE_HOST/api/services

# Check router logs
oc logs -n openshift-ingress -l ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default
```

### 3. **Storage and Performance Issues**

#### Storage Connectivity
```bash
# Test MinIO connectivity from components
oc exec deployment/tempo-simplest-compactor -- curl http://minio:9000/minio/health/live

# Check S3 configuration
oc get secret minio -o yaml
oc exec deployment/tempo-simplest-compactor -- cat /etc/tempo/tempo.yaml | grep -A10 storage

# Monitor storage operations
oc logs deployment/tempo-simplest-compactor | grep -i "s3\|storage\|block"
```

#### Performance Analysis
```bash
# Check component resource usage
oc top pods -l app.kubernetes.io/managed-by=tempo-operator

# Monitor query performance
oc port-forward deployment/tempo-simplest-query-frontend 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_query

# Check ingester block metrics
oc port-forward statefulset/tempo-simplest-ingester 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_ingester_blocks
```

## Related Configurations

- [TempoMonolithic Route](../monolithic-route/README.md) - Single-component route setup
- [Basic TempoStack](../../e2e/compatibility/README.md) - Distributed deployment without routes
- [TempoStack Monitoring](../monitoring/README.md) - Distributed monitoring setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/route
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates a distributed TempoStack deployment with OpenShift Route integration and comprehensive must-gather validation. The test requires cluster administrator privileges and demonstrates the complete operational tooling needed for production distributed Tempo deployments.

