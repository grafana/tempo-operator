# TempoMonolithic with Extra Configuration Override

This configuration blueprint demonstrates how to use the `extraConfig` feature in TempoMonolithic to customize Tempo's behavior beyond the standard operator-managed settings. This capability is essential for fine-tuning performance, adjusting timeouts, and configuring advanced features not exposed through the standard TempoMonolithic spec.

## Overview

This test validates advanced configuration customization features:
- **Configuration Override**: Custom Tempo configuration merged with operator defaults
- **Performance Tuning**: Query timeout and retry configuration
- **Jaeger UI Integration**: Combined with extra configuration capabilities
- **Config Inheritance**: Proper merging of custom and default configurations

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ TempoMonolithic     │───▶│   Configuration Merge     │───▶│ Final ConfigMap     │
│ extraConfig:        │    │   Process                │    │ - Default values    │
│ - query_timeout     │    │ ┌─────────────────────┐  │    │ - Custom overrides  │
│ - max_retries       │    │ │ Operator Defaults   │  │    │ - Merged result     │
└─────────────────────┘    │ │ +                   │  │    └─────────────────────┘
                           │ │ User Extra Config   │  │
┌─────────────────────┐    │ └─────────────────────┘  │    ┌─────────────────────┐
│ Trace Processing    │◀───│                          │    │ TempoMonolithic     │
│ - 180s timeout      │    └──────────────────────────┘    │ StatefulSet         │
│ - 3 max retries     │                                    │ + Jaeger UI         │
└─────────────────────┘                                    └─────────────────────┘
```

## Prerequisites

- Kubernetes cluster with basic resources
- Tempo Operator installed
- Understanding of Tempo configuration structure
- `kubectl` CLI access

## Step-by-Step Configuration

### Step 1: Deploy TempoMonolithic with Extra Configuration

Create the TempoMonolithic resource with custom configuration overrides:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  extraConfig:
    tempo:
      querier:
        search:
          query_timeout: 180s
      query_frontend:
        max_retries: 3
  jaegerui:
    enabled: true
EOF
```

**Key Configuration Details**:

#### Extra Configuration Structure
- `extraConfig.tempo`: Root for Tempo-specific configuration overrides
- `querier.search.query_timeout: 180s`: Extends default query timeout from 30s to 180s  
- `query_frontend.max_retries: 3`: Sets maximum query retry attempts to 3

#### Jaeger UI Integration  
- `jaegerui.enabled: true`: Enables integrated Jaeger UI alongside extra config

**Reference**: [`install-tempo.yaml`](./install-tempo.yaml)

### Step 2: Verify Configuration Merge

Validate that the extra configuration has been properly merged into the final ConfigMap:

```bash
# Check TempoMonolithic status
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Examine merged configuration
kubectl get configmap tempo-simplest-config -o yaml

# Verify specific custom values in the config
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | grep -A5 "querier:"
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | grep -A5 "query_frontend:"
```

Expected configuration merge results:
- **Query Timeout**: `query_timeout: 180s` appears in querier.search section
- **Max Retries**: `max_retries: 3` appears in query_frontend section  
- **Default Values**: All other operator-managed defaults remain intact

**Reference**: [`install-tempo-assert.yaml`](./install-tempo-assert.yaml)

### Step 3: Verify Component Functionality

Test that the deployment is functional with the custom configuration:

```bash
# Check StatefulSet readiness
kubectl get statefulset tempo-simplest

# Verify services are available
kubectl get services -l app.kubernetes.io/instance=simplest

# Test Tempo API endpoint
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/ready

# Test Jaeger UI endpoint
kubectl port-forward svc/tempo-simplest-jaegerui 16686:16686 &
curl http://localhost:16686/
```

## Configuration Override Capabilities

### 1. **Query Performance Tuning**

#### Extended Timeouts
```yaml
extraConfig:
  tempo:
    querier:
      search:
        query_timeout: 300s              # 5-minute timeout
        external_hedge_requests_at: 8s   # Hedge request timing
        external_hedge_requests_up_to: 2 # Max hedge requests
    query_frontend:
      search:
        concurrent_jobs: 1000            # Concurrent search jobs
        max_duration: 24h                # Maximum search duration
```

#### Trace Search Optimization
```yaml
extraConfig:
  tempo:
    query_frontend:
      search:
        default_result_limit: 50         # Default trace result limit
        max_result_limit: 1000           # Maximum allowed results
      max_retries: 5                     # Query retry attempts
      retry_delay: 100ms                 # Delay between retries
```

### 2. **Ingestion Configuration**

#### Receiver Customization
```yaml
extraConfig:
  tempo:
    distributor:
      receivers:
        jaeger:
          protocols:
            thrift_http:
              endpoint: 0.0.0.0:14268
              max_request_body_size: 10485760  # 10MB limit
            grpc:
              endpoint: 0.0.0.0:14250
              max_recv_msg_size: 16777216      # 16MB gRPC limit
        otlp:
          protocols:
            grpc:
              endpoint: 0.0.0.0:4317
              max_recv_msg_size: 67108864      # 64MB OTLP limit
```

#### Ring and Memberlist Configuration  
```yaml
extraConfig:
  tempo:
    memberlist:
      node_name: custom-node-name
      bind_port: 7946
      gossip_nodes: 3
      gossip_interval: 200ms
      retransmit_mult: 4
```

### 3. **Storage Optimization**

#### WAL Configuration
```yaml
extraConfig:
  tempo:
    ingester:
      trace_idle_period: 10s             # Trace idle time
      max_block_bytes: 104857600         # 100MB block size
      max_block_duration: 1h             # Block duration limit
      flush_check_period: 10s            # Flush check interval
      complete_block_timeout: 15m        # Block completion timeout
```

#### Block Management
```yaml
extraConfig:
  tempo:
    compactor:
      compaction:
        chunk_size_bytes: 10485760       # 10MB chunks
        flush_size_bytes: 52428800       # 50MB flush size
        max_compaction_objects: 1000000  # Max objects per compaction
        block_retention: 168h            # 7-day retention
```

### 4. **Advanced Server Configuration**

#### HTTP Server Tuning
```yaml
extraConfig:
  tempo:
    server:
      http_listen_port: 3200
      http_server_read_timeout: 60s      # Extended read timeout
      http_server_write_timeout: 60s     # Extended write timeout
      http_server_idle_timeout: 120s     # Idle connection timeout
      grpc_server_max_recv_msg_size: 104857600  # 100MB gRPC
      grpc_server_max_send_msg_size: 104857600
```

#### Metrics and Observability
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
      traces_storage:
        path: /var/tempo/generator/traces
```

## Configuration Validation and Testing

### 1. **Configuration Syntax Validation**

Before applying changes, validate YAML structure:

```bash
# Test configuration syntax
kubectl apply -f tempo-config.yaml --dry-run=client

# Validate against Tempo schema (if available)
tempo -config.file=/tmp/tempo.yaml -config.check-syntax
```

### 2. **Incremental Configuration Testing**

```yaml
# Start with minimal overrides
extraConfig:
  tempo:
    querier:
      search:
        query_timeout: 60s

# Gradually add more complex configurations
extraConfig:
  tempo:
    querier:
      search:
        query_timeout: 180s
        concurrent_jobs: 500
    query_frontend:
      max_retries: 3
      retry_delay: 500ms
```

### 3. **Configuration Impact Monitoring**

```bash
# Monitor query performance after configuration changes
kubectl port-forward svc/tempo-simplest 3200:3200 &

# Check metrics for query duration
curl http://localhost:3200/metrics | grep tempo_query_frontend

# Monitor resource usage
kubectl top pod tempo-simplest-0

# Watch for errors in logs
kubectl logs tempo-simplest-0 | grep -i error
```

## Common Configuration Patterns

### 1. **High-Volume Environment**
```yaml
extraConfig:
  tempo:
    distributor:
      ring:
        kvstore:
          store: memberlist
    ingester:
      max_block_duration: 30m
      flush_check_period: 5s
    querier:
      search:
        concurrent_jobs: 2000
        query_timeout: 300s
    query_frontend:
      max_retries: 5
      search:
        max_result_limit: 2000
```

### 2. **Long-Term Retention Setup**
```yaml
extraConfig:
  tempo:
    compactor:
      compaction:
        block_retention: 2160h  # 90 days
    storage:
      trace:
        blocklist_poll: 300s    # 5-minute polling
```

### 3. **Development Environment**
```yaml
extraConfig:
  tempo:
    querier:
      search:
        query_timeout: 30s
    query_frontend:
      max_retries: 1
    usage_report:
      reporting_enabled: false
    server:
      log_level: debug
```

## Troubleshooting Configuration Issues

### 1. **Configuration Merge Problems**

```bash
# Check if custom config appears in final ConfigMap
kubectl describe configmap tempo-simplest-config

# Compare with expected configuration
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | yq eval '.'

# Verify no YAML syntax errors
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | yq eval '.' > /dev/null
```

### 2. **Performance Issues After Configuration**

```bash
# Monitor query latency
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_request_duration

# Check for timeout errors
kubectl logs tempo-simplest-0 | grep -i timeout

# Verify resource limits aren't exceeded
kubectl describe pod tempo-simplest-0 | grep -A5 "Limits:"
```

### 3. **Invalid Configuration Detection**

```bash
# Check pod restart loops
kubectl get pods -l app.kubernetes.io/instance=simplest

# Review startup errors
kubectl logs tempo-simplest-0 --previous

# Validate configuration syntax
kubectl exec tempo-simplest-0 -- tempo -config.file=/conf/tempo.yaml -config.check-syntax
```

## Security Considerations

### 1. **Configuration Sensitivity**
- Avoid putting secrets in extraConfig (use proper secret management)
- Review exposed endpoints and authentication requirements
- Validate network security policies with custom configurations

### 2. **Resource Constraints**
- Set appropriate limits for query timeouts and retries
- Monitor memory usage with larger buffer sizes
- Implement rate limiting for high-concurrency settings

### 3. **Configuration Validation**
- Test configuration changes in non-production environments first
- Implement configuration review processes
- Monitor system behavior after configuration updates

## Production Best Practices

### 1. **Change Management**
- Version control all extraConfig specifications
- Test configuration changes in staging environments
- Implement gradual rollout procedures
- Document all custom configuration purposes

### 2. **Monitoring and Alerting**
- Set up alerts for configuration-related errors
- Monitor query performance metrics
- Track resource utilization changes
- Alert on excessive retry rates or timeouts

### 3. **Backup and Recovery**
- Backup working configurations before changes
- Document rollback procedures
- Test configuration restore processes
- Maintain configuration change logs

## Related Configurations

- [TempoMonolithic Memory Storage](../monolithic-memory/README.md) - Basic monolithic setup
- [TempoStack Extra Configuration](../tempostack-extraconfig/README.md) - Distributed deployment extra config
- [Performance Monitoring](../../e2e-openshift/monitoring/README.md) - Monitoring setup for custom configurations

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-extraconfig
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs in namespace `chainsaw-monoextcfg` to isolate configuration testing.

