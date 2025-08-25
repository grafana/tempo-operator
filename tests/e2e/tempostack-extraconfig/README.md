# TempoStack with Extra Configuration Override

This configuration blueprint demonstrates how to use the `extraConfig` feature in TempoStack to customize Tempo's behavior beyond the standard operator-managed settings. This capability enables fine-tuning of performance parameters, timeout settings, and advanced features for distributed Tempo deployments in production environments.

## Overview

This test validates advanced TempoStack configuration customization features:
- **Distributed Configuration Override**: Custom Tempo configuration for multi-component architecture
- **Performance Tuning**: Server timeout and query optimization settings
- **Component-Specific Settings**: Targeted configuration for querier and query frontend
- **Jaeger UI Integration**: Query interface configuration with ingress support

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ TempoStack          │───▶│   Configuration Merge     │───▶│ Component ConfigMaps│
│ extraConfig:        │    │   Process                │    │ - Distributor       │
│ - server timeouts   │    │ ┌─────────────────────┐  │    │ - Ingester          │
│ - query settings    │    │ │ Operator Defaults   │  │    │ - Querier           │
│ - retry config      │    │ │ +                   │  │    │ - Query Frontend    │
└─────────────────────┘    │ │ User Extra Config   │  │    │ - Compactor         │
                           │ └─────────────────────┘  │    └─────────────────────┘
┌─────────────────────┐    │                          │
│ Component Services  │◀───│ Applied Configuration:    │    ┌─────────────────────┐
│ - 10m timeouts      │    │ - 180s query timeout     │───▶│ MinIO Storage       │
│ - 3 max retries     │    │ - 10m server timeouts    │    │ - S3 Compatible     │
│ - Optimized search  │    │ - Custom retry logic     │    │ - 200M allocation   │
└─────────────────────┘    └──────────────────────────┘    └─────────────────────┘

┌─────────────────────┐
│ Jaeger UI + Ingress │
│ - External Access   │
│ - Query Interface   │
└─────────────────────┘
```

## Prerequisites

- Kubernetes cluster with ingress controller
- Tempo Operator installed
- Persistent volume support for MinIO
- `kubectl` CLI access
- Understanding of Tempo distributed architecture

## Step-by-Step Configuration

### Step 1: Deploy Persistent MinIO Storage

Create MinIO with persistent storage for distributed TempoStack:

```bash
kubectl apply -f - <<EOF
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

**Storage Configuration Details**:
- **Persistent Volume**: 2Gi PVC for durable storage
- **MinIO Server**: S3-compatible object storage
- **Bucket Setup**: Automatic `tempo` bucket creation
- **Access Credentials**: Simple authentication for testing

**Reference**: [`install-storage.yaml`](./install-storage.yaml)

### Step 2: Deploy TempoStack with Extra Configuration

Create the distributed TempoStack with comprehensive extra configuration:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  timeout: 70s
  extraConfig:
    tempo:
      server:
        http_server_write_timeout: 10m
        http_server_read_timeout: 10m
      querier:
        search:
          query_timeout: 180s
      query_frontend:
        max_retries: 3
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: ingress
EOF
```

**Key Configuration Elements**:

#### Global Settings
- `timeout: 70s`: Default timeout for TempoStack operations
- `storageSize: 200M`: Compact storage allocation for testing
- **Resource Limits**: Total 2Gi memory, 2000m CPU across all components

#### Extra Configuration (`extraConfig.tempo`)

##### Server Configuration
```yaml
server:
  http_server_write_timeout: 10m
  http_server_read_timeout: 10m
```
- **Extended Timeouts**: 10-minute timeouts for large request handling
- **Write Operations**: Accommodates slow storage or large trace ingestion
- **Read Operations**: Supports complex queries over large datasets

##### Querier Configuration
```yaml
querier:
  search:
    query_timeout: 180s
```
- **Query Timeout**: 3-minute timeout for trace search operations
- **Search Optimization**: Balances performance vs completeness
- **Complex Queries**: Supports deep trace analysis

##### Query Frontend Configuration
```yaml
query_frontend:
  max_retries: 3
```
- **Retry Logic**: Up to 3 retry attempts for failed queries
- **Resilience**: Improves reliability in distributed environments
- **Performance**: Reduces impact of transient failures

#### Jaeger UI Integration
```yaml
template:
  queryFrontend:
    jaegerQuery:
      enabled: true
      ingress:
        type: ingress
```
- **Jaeger Interface**: Enables Jaeger-compatible query API
- **Ingress Access**: External access via Kubernetes ingress
- **UI Integration**: Supports existing Jaeger UI workflows

**Reference**: [`install-tempostack.yaml`](./install-tempostack.yaml)

### Step 3: Verify Distributed Configuration

Validate that extra configuration is properly applied across all components:

```bash
# Check TempoStack readiness
kubectl get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify all distributed components are running
kubectl get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check configuration in each component
kubectl get configmap tempo-simplest-distributor -o jsonpath='{.data.tempo\.yaml}' | grep -A5 server
kubectl get configmap tempo-simplest-querier -o jsonpath='{.data.tempo\.yaml}' | grep -A5 search
kubectl get configmap tempo-simplest-query-frontend -o jsonpath='{.data.tempo\.yaml}' | grep -A5 max_retries

# Verify Jaeger query interface
kubectl get service tempo-simplest-query-frontend
kubectl get ingress tempo-simplest-query-frontend-jaeger
```

Expected validation results:
- **Component Readiness**: All distributor, ingester, querier, query-frontend, compactor pods ready
- **Configuration Propagation**: Custom timeouts and settings in respective ConfigMaps
- **Service Creation**: Query frontend service with Jaeger ports exposed
- **Ingress Setup**: External access configured for Jaeger UI

## Distributed Extra Configuration Features

### 1. **Component-Specific Configuration**

#### Distributor Settings
```yaml
extraConfig:
  tempo:
    distributor:
      receivers:
        otlp:
          protocols:
            grpc:
              max_recv_msg_size: 67108864  # 64MB
            http:
              max_request_body_size: 67108864
      ring:
        kvstore:
          store: memberlist
```

#### Ingester Optimization
```yaml
extraConfig:
  tempo:
    ingester:
      max_block_duration: 30m
      max_block_bytes: 104857600        # 100MB
      flush_check_period: 10s
      trace_idle_period: 10s
      lifecycler:
        ring:
          replication_factor: 3
```

#### Querier Performance Tuning
```yaml
extraConfig:
  tempo:
    querier:
      search:
        query_timeout: 300s              # 5 minutes
        concurrent_jobs: 1000
        max_duration: 24h
        default_result_limit: 50
      max_concurrent_queries: 50
      frontend_worker:
        parallelism: 10
```

#### Query Frontend Advanced Settings
```yaml
extraConfig:
  tempo:
    query_frontend:
      search:
        concurrent_jobs: 2000
        max_duration: 0s                 # No limit
        default_result_limit: 20
        max_result_limit: 1000
      max_retries: 5
      retry_delay: 100ms
      log_queries_longer_than: 5s
```

#### Compactor Configuration
```yaml
extraConfig:
  tempo:
    compactor:
      compaction:
        block_retention: 168h            # 7 days
        compacted_block_retention: 1h
        max_compaction_objects: 1000000
        chunk_size_bytes: 10485760       # 10MB
        flush_size_bytes: 52428800       # 50MB
      ring:
        kvstore:
          store: memberlist
```

### 2. **Global Performance Settings**

#### Storage Optimization
```yaml
extraConfig:
  tempo:
    storage:
      trace:
        blocklist_poll: 300s             # 5 minutes
        cache: redis                     # Optional Redis cache
        background_cache:
          writeback_goroutines: 10
          writeback_buffer: 10000
        wal:
          path: /var/tempo/wal
          completedFilePath: /var/tempo/completed
```

#### Memberlist Configuration
```yaml
extraConfig:
  tempo:
    memberlist:
      node_name: tempo-${POD_NAME}
      randomize_node_name: false
      stream_timeout: 10s
      retransmit_mult: 4
      gossip_interval: 200ms
      gossip_nodes: 3
      dead_node_reclaim_time: 0s
```

### 3. **Monitoring and Observability**

#### Metrics Configuration
```yaml
extraConfig:
  tempo:
    metrics_generator:
      registry:
        external_labels:
          cluster: production
          region: us-west-2
      storage:
        path: /var/tempo/generator
        wal:
          path: /var/tempo/generator/wal
      traces_storage:
        path: /var/tempo/generator/traces
```

#### Usage Reporting
```yaml
extraConfig:
  tempo:
    usage_report:
      reporting_enabled: false           # Disable for privacy
```

## Configuration Validation and Testing

### 1. **Configuration Syntax Validation**

Before deployment, validate configuration syntax:

```bash
# Test YAML syntax
kubectl apply -f tempostack-config.yaml --dry-run=client

# Validate against TempoStack schema
kubectl explain tempostack.spec.extraConfig

# Check for configuration conflicts
kubectl get tempostack simplest -o yaml | yq '.spec.extraConfig'
```

### 2. **Component Configuration Verification**

Verify configuration propagation to each component:

```bash
# Check distributor configuration
kubectl get configmap tempo-simplest-distributor -o jsonpath='{.data.tempo\.yaml}' | yq '.distributor'

# Verify querier settings
kubectl get configmap tempo-simplest-querier -o jsonpath='{.data.tempo\.yaml}' | yq '.querier'

# Validate query frontend config
kubectl get configmap tempo-simplest-query-frontend -o jsonpath='{.data.tempo\.yaml}' | yq '.query_frontend'

# Check compactor configuration
kubectl get configmap tempo-simplest-compactor -o jsonpath='{.data.tempo\.yaml}' | yq '.compactor'
```

### 3. **Runtime Behavior Testing**

Test that configuration changes affect runtime behavior:

```bash
# Test extended query timeout
time kubectl exec tempo-simplest-querier-0 -- \
  curl -G "http://localhost:3200/api/search" \
  --data-urlencode "q={}" \
  --max-time 200

# Verify retry behavior
kubectl logs tempo-simplest-query-frontend-0 | grep -i retry

# Check server timeout behavior
curl -X POST http://tempo-simplest-distributor:3200/v1/traces \
  -H "Content-Type: application/json" \
  --max-time 600 \
  -d '{}'
```

## Advanced Configuration Patterns

### 1. **Environment-Specific Configuration**

#### Development Environment
```yaml
extraConfig:
  tempo:
    server:
      log_level: debug
      log_format: logfmt
    querier:
      search:
        query_timeout: 30s
    query_frontend:
      max_retries: 1
    usage_report:
      reporting_enabled: false
```

#### Production Environment
```yaml
extraConfig:
  tempo:
    server:
      log_level: info
      log_format: json
    querier:
      search:
        query_timeout: 300s
        concurrent_jobs: 2000
    query_frontend:
      max_retries: 5
      retry_delay: 500ms
    compactor:
      compaction:
        block_retention: 2160h  # 90 days
```

### 2. **High-Throughput Configuration**

```yaml
extraConfig:
  tempo:
    distributor:
      ring:
        kvstore:
          store: memberlist
    ingester:
      max_block_duration: 10m
      flush_check_period: 5s
    querier:
      max_concurrent_queries: 100
      search:
        concurrent_jobs: 3000
    query_frontend:
      search:
        concurrent_jobs: 5000
```

### 3. **Security-Hardened Configuration**

```yaml
extraConfig:
  tempo:
    server:
      http_server_read_timeout: 30s
      http_server_write_timeout: 30s
      http_server_idle_timeout: 120s
      grpc_server_max_recv_msg_size: 4194304    # 4MB limit
      grpc_server_max_send_msg_size: 4194304
    usage_report:
      reporting_enabled: false
```

## Troubleshooting Configuration Issues

### 1. **Configuration Merge Problems**

```bash
# Check if custom config appears in component ConfigMaps
kubectl get configmap -l app.kubernetes.io/managed-by=tempo-operator

# Compare expected vs actual configuration
kubectl get configmap tempo-simplest-querier -o jsonpath='{.data.tempo\.yaml}' | yq '.querier.search'

# Verify configuration inheritance
kubectl describe tempostack simplest | grep -A20 "Extra Config"
```

### 2. **Component Startup Issues**

```bash
# Check for configuration-related startup errors
kubectl logs tempo-simplest-querier-0 | grep -i "config\|error\|failed"

# Validate configuration syntax in running pods
kubectl exec tempo-simplest-querier-0 -- \
  tempo -config.file=/conf/tempo.yaml -config.check-syntax

# Check for resource constraint issues
kubectl describe pod tempo-simplest-querier-0 | grep -A5 "Limits:"
```

### 3. **Performance Impact Assessment**

```bash
# Monitor query performance after configuration changes
kubectl port-forward svc/tempo-simplest-query-frontend 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_query_frontend

# Check timeout effectiveness
kubectl logs tempo-simplest-querier-0 | grep -i timeout

# Monitor resource usage changes
kubectl top pods -l app.kubernetes.io/managed-by=tempo-operator
```

## Production Deployment Considerations

### 1. **Configuration Management**
- Version control all extraConfig specifications
- Test configuration changes in staging environments
- Implement gradual rollout procedures for configuration updates
- Document all custom configuration purposes and impacts

### 2. **Monitoring and Alerting**
```yaml
# Prometheus alerts for configuration-related issues
alert: TempoQueryTimeouts
expr: increase(tempo_query_frontend_queries_total{status_code="timeout"}[5m]) > 0
for: 2m
annotations:
  summary: "Tempo queries are timing out"

alert: TempoHighRetryRate
expr: rate(tempo_query_frontend_retries_total[5m]) > 0.1
for: 5m
annotations:
  summary: "High retry rate detected in Tempo query frontend"
```

### 3. **Resource Planning**
- Monitor resource utilization after applying extra configuration
- Plan for increased memory/CPU usage with extended timeouts
- Consider storage implications of longer retention periods
- Scale components based on configuration requirements

### 4. **Security Considerations**
- Review timeout values for DoS protection
- Validate resource limits prevent resource exhaustion
- Ensure logging configuration doesn't expose sensitive data
- Regular security assessments of configuration changes

## Related Configurations

- [TempoMonolithic Extra Config](../monolithic-extraconfig/README.md) - Single-pod configuration overrides
- [TempoStack Basic Setup](../compatibility/README.md) - Standard distributed deployment
- [Performance Monitoring](../../e2e-openshift/monitoring/README.md) - Monitoring configuration impact

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/tempostack-extraconfig
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs in namespace `chainsaw-tempoextcfg` and includes a comprehensive readiness check to ensure all configuration changes are properly applied before test completion.

