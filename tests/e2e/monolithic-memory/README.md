# TempoMonolithic with In-Memory Storage

This configuration blueprint demonstrates how to deploy a simple, single-component Tempo observability stack using TempoMonolithic with in-memory storage. This setup is ideal for development, testing, and small-scale deployments where persistence is not required.

## Overview

This test validates a lightweight observability stack featuring:
- **TempoMonolithic**: Single-pod Tempo deployment with all components integrated
- **In-Memory Storage**: No external storage dependencies
- **Jaeger UI Integration**: Built-in Jaeger UI for trace visualization
- **Simplified Architecture**: Minimal resource requirements and configuration

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐
│ Trace Generator │───▶│   TempoMonolithic    │
│ (telemetrygen)  │    │ ┌─────────────────┐  │
└─────────────────┘    │ │ All Components  │  │
                       │ │ - Distributor   │  │
┌─────────────────┐    │ │ - Ingester      │  │◀─ In-Memory
│ Query Clients   │◀───│ │ - Querier       │  │   Storage
│ - Tempo API     │    │ │ - Compactor     │  │
│ - Jaeger UI     │    │ └─────────────────┘  │
└─────────────────┘    └──────────────────────┘
```

## Prerequisites

- Kubernetes cluster with basic resources
- Tempo Operator installed
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Deploy TempoMonolithic

Create the monolithic Tempo deployment with Jaeger UI:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  jaegerui:
    enabled: true
EOF
```

**Key Configuration Details**:
- **TempoMonolithic**: Single-pod deployment containing all Tempo components
- **jaegerui.enabled**: Enables integrated Jaeger UI for trace visualization
- **No storage configuration**: Uses in-memory storage by default
- **Minimal spec**: Uses default resource allocations

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 2: Verify Deployment

Wait for TempoMonolithic to be ready:

```bash
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

Check the pod is running:

```bash
kubectl get pods -l app.kubernetes.io/managed-by=tempo-operator
```

Verify services are created:

```bash
kubectl get services -l app.kubernetes.io/managed-by=tempo-operator
```

Expected services:
- `tempo-simplest`: Main Tempo service (port 3200 for queries, 4317 for OTLP ingestion)
- `tempo-simplest-jaegerui`: Jaeger UI service (port 16686)

### Step 3: Generate Sample Traces

Create traces using OpenTelemetry's telemetrygen tool:

```bash
kubectl apply -f - <<EOF
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
        - --otlp-endpoint=tempo-simplest:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Configuration Notes**:
- `--otlp-endpoint=tempo-simplest:4317`: Points directly to the monolithic service
- `--otlp-insecure`: Uses unencrypted connection (suitable for testing)
- `--traces=10`: Generates exactly 10 traces for verification

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 4: Verify Traces via APIs

Test both Tempo native API and Jaeger-compatible API:

```bash
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces-jaeger
spec:
  template:
    spec:
      containers:
      - name: verify-traces-jaeger
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Query Tempo's native search API
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Tempo API returned \$num_traces instead of 10 traces."
            exit 1
          fi

          # Query Jaeger-compatible API
          curl -v -G \
            http://tempo-simplest-jaegerui:16686/api/traces \
            --data-urlencode "service=telemetrygen" | tee /tmp/jaeger.out
          
          num_traces=\$(jq ".data | length" /tmp/jaeger.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Jaeger API returned \$num_traces instead of 10 traces."
            exit 1
          fi
      restartPolicy: Never
EOF
```

**API Endpoints Tested**:
- **Tempo Native API**: `http://tempo-simplest:3200/api/search`
- **Jaeger Compatible API**: `http://tempo-simplest-jaegerui:16686/api/traces`

**Reference**: [`04-verify-traces-jaeger.yaml`](./04-verify-traces-jaeger.yaml)

### Step 5: Access Jaeger UI

To access the Jaeger UI locally:

```bash
# Port-forward to Jaeger UI
kubectl port-forward svc/tempo-simplest-jaegerui 16686:16686

# Open browser to http://localhost:16686
```

The Jaeger UI provides:
- Service dependency graph
- Trace search and filtering
- Detailed trace timeline view
- Service performance metrics

## Key Features Demonstrated

### 1. **Simplified Deployment**
- Single resource (TempoMonolithic) creates complete observability stack
- No external dependencies or storage configuration required
- Automatic service creation and configuration

### 2. **In-Memory Storage Benefits**
- **Fast Performance**: No disk I/O bottlenecks
- **No Persistence**: Traces are lost on pod restart (suitable for testing)
- **Resource Efficient**: Minimal storage overhead
- **Quick Startup**: No storage initialization delays

### 3. **Integrated Jaeger UI**
- Built-in trace visualization
- Compatible with existing Jaeger workflows
- No additional UI deployment required
- Consistent branding and experience

### 4. **Development-Friendly**
- Minimal configuration required
- Quick iteration cycles
- Easy debugging and troubleshooting
- No cleanup of persistent data needed

## Configuration Options

### Resource Customization

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  jaegerui:
    enabled: true
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "512Mi"
      cpu: "500m"
```

### Jaeger UI Configuration

```yaml
spec:
  jaegerui:
    enabled: true
    ingress:
      type: ingress
      host: jaeger.example.com
      annotations:
        nginx.ingress.kubernetes.io/ssl-redirect: "false"
```

### Observability Configuration

```yaml
spec:
  observability:
    metrics:
      createServiceMonitors: true
    tracing:
      sampling_fraction: 1.0
```

## Troubleshooting

### Check TempoMonolithic Status
```bash
kubectl describe tempomonolithic simplest
```

### View Logs
```bash
kubectl logs -l app.kubernetes.io/name=tempo
```

### Test Direct API Access
```bash
# Port-forward to Tempo API
kubectl port-forward svc/tempo-simplest 3200:3200

# Test search API
curl "http://localhost:3200/api/search?q={}"

# Test metrics endpoint
curl "http://localhost:3200/metrics"
```

### Verify Trace Ingestion
```bash
# Check if traces are being received
kubectl logs -l app.kubernetes.io/name=tempo | grep -i "received"
```

### Common Issues

1. **Pod Not Starting**
   ```bash
   kubectl describe pod -l app.kubernetes.io/name=tempo
   kubectl logs -l app.kubernetes.io/name=tempo
   ```

2. **No Traces Visible**
   - Verify telemetrygen job completed successfully
   - Check if OTLP endpoint is reachable
   - Ensure ports 4317 (OTLP) and 3200 (HTTP) are accessible

3. **Jaeger UI Not Accessible**
   ```bash
   kubectl get svc tempo-simplest-jaegerui
   kubectl port-forward svc/tempo-simplest-jaegerui 16686:16686
   ```

## Use Cases

### 1. **Development Environment**
- Local development and testing
- CI/CD pipeline validation
- Feature development and debugging

### 2. **Demo and Training**
- Quick observability stack setup
- Educational purposes
- Proof of concept deployments

### 3. **Edge Deployments**
- Resource-constrained environments
- Temporary monitoring setups
- Edge computing scenarios

### 4. **Migration Testing**
- Testing migration from other tracing systems
- Validation of instrumentation changes
- Performance benchmarking

## Production Considerations

⚠️ **Important**: This configuration is not recommended for production use due to:

- **No Persistence**: All traces are lost on pod restart
- **Single Point of Failure**: No redundancy or high availability
- **Resource Limitations**: Single pod cannot scale to handle high traffic
- **Memory Constraints**: In-memory storage is limited by pod memory

For production deployments, consider:
- [TempoStack with Object Storage](../compatibility/README.md)
- [Multi-tenant Setup](../../e2e-openshift/multitenancy/README.md)
- [TLS-enabled Configuration](../tls-singletenant/README.md)

## Related Configurations

- [TempoStack Compatibility](../compatibility/README.md) - Distributed deployment with persistence
- [Custom Storage Class](../monolithic-custom-storage-class/README.md) - Persistent volume configuration
- [TLS Configuration](../tls-singletenant/README.md) - Secure communications setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-memory
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)