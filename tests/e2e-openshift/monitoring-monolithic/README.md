# TempoMonolithic with OpenShift User Workload Monitoring

This configuration blueprint demonstrates how to integrate TempoMonolithic with OpenShift's User Workload Monitoring system for comprehensive observability and metrics collection. This setup enables automatic scraping of Tempo metrics by OpenShift's built-in Prometheus/Thanos monitoring stack, providing seamless integration with cluster-wide monitoring infrastructure.

## Overview

This test validates OpenShift-specific monitoring integration features:
- **User Workload Monitoring**: Enable OpenShift's monitoring for user applications
- **ServiceMonitor Integration**: Automatic Prometheus scraping configuration
- **PrometheusRules**: Built-in alerting rules for Tempo metrics
- **Thanos Querier Access**: Validate metrics through OpenShift's monitoring API

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ TempoMonolithic     │───▶│   OpenShift Monitoring    │───▶│ Thanos Querier      │
│ - Metrics Endpoint  │    │   Stack                  │    │ - Unified Query API │
│ - ServiceMonitor    │    │ ┌─────────────────────┐  │    │ - External Access   │
│ - PrometheusRules   │    │ │ User Workload       │  │    └─────────────────────┘
└─────────────────────┘    │ │ Monitoring          │  │
                           │ │ - Prometheus        │  │    ┌─────────────────────┐
┌─────────────────────┐    │ │ - ServiceMonitor    │  │───▶│ Cluster Monitoring  │
│ Trace Generation    │───▶│ │ - PrometheusRule    │  │    │ - Grafana           │
│ - OTLP Ingestion    │    │ └─────────────────────┘  │    │ - AlertManager      │
│ - Metrics Creation  │    │                          │    │ - Long-term Storage │
└─────────────────────┘    └──────────────────────────┘    └─────────────────────┘

┌─────────────────────┐    
│ Jaeger UI           │    
│ - OpenShift Route   │    
│ - External Access   │    
└─────────────────────┘    
```

## Prerequisites

- OpenShift cluster (4.13+)
- Tempo Operator installed
- Cluster administrator privileges (for enabling user workload monitoring)
- `oc` CLI access
- Understanding of OpenShift monitoring concepts

## Step-by-Step Configuration

### Step 1: Enable OpenShift User Workload Monitoring

Configure the cluster to enable monitoring for user-defined projects:

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
- `enableUserWorkload: true`: Enables Prometheus for user applications
- `enableUserAlertmanagerConfig: true`: Allows user-defined alerting rules
- **Global Scope**: Affects entire cluster monitoring configuration

**Impact and Verification**:
```bash
# Verify user workload monitoring is enabled
oc get pods -n openshift-user-workload-monitoring

# Check for user workload monitoring components
oc get prometheus -n openshift-user-workload-monitoring
oc get thanos-ruler -n openshift-user-workload-monitoring

# Verify cluster monitoring configuration
oc get configmap cluster-monitoring-config -n openshift-monitoring -o yaml
```

**Reference**: [`workload-monitoring.yaml`](./workload-monitoring.yaml)

### Step 2: Deploy TempoMonolithic with Monitoring Features

Create TempoMonolithic with comprehensive monitoring integration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: monitor
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
EOF
```

**Key Configuration Elements**:

#### Jaeger UI with OpenShift Route
- `jaegerui.enabled: true`: Enables Jaeger query interface
- `route.enabled: true`: Creates OpenShift Route for external access
- **External Access**: Direct access to Jaeger UI through OpenShift routing

#### Monitoring Integration
- `prometheusRules.enabled: true`: Creates alerting rules for Tempo
- `serviceMonitors.enabled: true`: Configures Prometheus scraping

**Generated Monitoring Resources**:
```bash
# ServiceMonitor for metrics scraping
oc get servicemonitor -l app.kubernetes.io/managed-by=tempo-operator

# PrometheusRule for alerting
oc get prometheusrule -l app.kubernetes.io/managed-by=tempo-operator

# OpenShift Route for Jaeger UI
oc get route -l app.kubernetes.io/managed-by=tempo-operator
```

**Reference**: [`install-monolithic.yaml`](./install-monolithic.yaml)

### Step 3: Generate Traces for Metrics Creation

Create traces to produce meaningful metrics for monitoring validation:

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
        - --otlp-endpoint=tempo-monitor:4317
        - --otlp-insecure
        - --traces=100
        - --rate=10
        - --duration=30s
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Generation for Metrics**:
- **Volume**: 100 traces at 10 traces/second for 30 seconds
- **Sustained Load**: Creates meaningful metrics for monitoring validation
- **Endpoint**: Direct connection to TempoMonolithic distributor

**Reference**: [`generate-traces.yaml`](./generate-traces.yaml)

### Step 4: Verify Traces and Metrics Creation

Validate that traces are properly ingested and metrics are being generated:

```bash
# Verify traces via Tempo API
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
spec:
  template:
    spec:
      containers:
      - name: verify-traces
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          curl -v -G \
            http://tempo-monitor:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -lt 50 ]]; then
            echo "Expected at least 50 traces, got \$num_traces"
            exit 1
          fi
          echo "✓ Verified \$num_traces traces present for metrics generation"
      restartPolicy: Never
EOF
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

### Step 5: Validate OpenShift Monitoring Integration

Execute comprehensive monitoring validation through OpenShift's Thanos Querier:

```bash
# Run the monitoring validation script
./check_metrics.sh
```

**Metrics Validation Script Details**:

#### Service Account and RBAC Setup
```bash
# Create service account for metrics access
oc create serviceaccount e2e-test-metrics-reader -n $NAMESPACE

# Grant cluster monitoring view permissions
oc adm policy add-cluster-role-to-user cluster-monitoring-view \
  system:serviceaccount:$NAMESPACE:e2e-test-metrics-reader

# Generate access token for Thanos Querier
TOKEN=$(oc create token e2e-test-metrics-reader -n $NAMESPACE)
```

#### Thanos Querier Access
```bash
# Get Thanos Querier route
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

# Query metrics via Thanos API
curl -k -H "Authorization: Bearer $TOKEN" \
  -H "Content-type: application/json" \
  "https://$THANOS_QUERIER_HOST/api/v1/query?query=tempo_build_info"
```

#### Validated Metrics
The script validates these key Tempo metrics:
- `tempo_query_frontend_queries_total`: Query operation counts
- `tempo_distributor_bytes_received_total`: Ingestion volume metrics
- `tempo_distributor_spans_received_total`: Span ingestion counts
- `tempo_ingester_bytes_received_total`: Ingester throughput
- `tempo_distributor_traces_per_batch_count`: Batch size metrics
- `tempo_build_info`: Version and build information

**Reference**: [`check_metrics.sh`](./check_metrics.sh)

## OpenShift Monitoring Integration Features

### 1. **ServiceMonitor Configuration**

The operator automatically creates ServiceMonitor resources:

```yaml
# Generated ServiceMonitor example
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tempo-monitor
  labels:
    app.kubernetes.io/managed-by: tempo-operator
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: tempo
  endpoints:
  - port: http
    interval: 30s
    path: /metrics
```

**ServiceMonitor Features**:
- **Automatic Discovery**: Prometheus automatically discovers and scrapes
- **Label Selectors**: Targets correct Tempo services
- **Scrape Configuration**: Optimized intervals and paths

### 2. **PrometheusRule for Alerting**

Built-in alerting rules for Tempo monitoring:

```yaml
# Generated PrometheusRule example
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tempo-monitor-alerts
spec:
  groups:
  - name: tempo.rules
    rules:
    - alert: TempoRequestLatency
      expr: histogram_quantile(0.99, sum(rate(tempo_request_duration_seconds_bucket[5m])) by (le)) > 3
      for: 5m
      annotations:
        summary: "Tempo request latency is high"
    - alert: TempoIngestorUnhealthy
      expr: up{job="tempo-monitor"} == 0
      for: 1m
      annotations:
        summary: "Tempo ingester is down"
```

### 3. **Metrics Exposed by TempoMonolithic**

#### Core Application Metrics
- **Ingestion Metrics**: Bytes and spans received, processing rates
- **Query Metrics**: Query counts, latency, error rates
- **Storage Metrics**: Block operations, compaction status

#### Infrastructure Metrics
- **Go Runtime**: Memory usage, GC statistics, goroutines
- **HTTP Server**: Request/response metrics, connection stats
- **Process**: CPU usage, file descriptors, uptime

#### Custom Tempo Metrics
- **Trace Processing**: Trace completion rates, batch statistics
- **Component Health**: Component status, dependency checks
- **Business Logic**: Application-specific operational metrics

## Advanced Monitoring Configuration

### 1. **Custom Scrape Configuration**

```yaml
spec:
  observability:
    metrics:
      serviceMonitors:
        enabled: true
        scrapeInterval: "15s"        # Custom scrape interval
        scrapeTimeout: "10s"         # Timeout for scrapes
        metricRelabelings:           # Custom metric processing
        - sourceLabels: [__name__]
          regex: 'tempo_.*'
          targetLabel: component
          replacement: 'tempo'
```

### 2. **Enhanced PrometheusRules**

```yaml
spec:
  observability:
    metrics:
      prometheusRules:
        enabled: true
        additionalRuleLabels:        # Custom labels for rules
          team: "observability"
          environment: "production"
        namespace: "tempo-monitoring" # Custom namespace for rules
```

### 3. **Resource Limits for Monitoring**

```yaml
spec:
  resources:
    total:
      limits:
        memory: 4Gi
        cpu: 2000m
    # Ensure adequate resources for metrics collection
  observability:
    metrics:
      serviceMonitors:
        enabled: true
      # Monitor resource impact of metrics collection
```

## Troubleshooting Monitoring Issues

### 1. **User Workload Monitoring Not Working**

#### Check Cluster Configuration
```bash
# Verify user workload monitoring is enabled
oc get configmap cluster-monitoring-config -n openshift-monitoring -o yaml

# Check user workload monitoring pods
oc get pods -n openshift-user-workload-monitoring

# Verify Prometheus is running
oc get prometheus -n openshift-user-workload-monitoring
```

#### Enable User Workload Monitoring
```bash
# If not enabled, apply the configuration
oc patch configmap cluster-monitoring-config -n openshift-monitoring \
  --patch '{"data":{"config.yaml":"enableUserWorkload: true\nalertmanagerMain:\n  enableUserAlertmanagerConfig: true"}}'

# Wait for components to start
oc wait --for=condition=Ready pod -l app.kubernetes.io/name=prometheus -n openshift-user-workload-monitoring --timeout=300s
```

### 2. **ServiceMonitor Not Discovered**

#### Check ServiceMonitor Creation
```bash
# Verify ServiceMonitor exists
oc get servicemonitor -l app.kubernetes.io/managed-by=tempo-operator

# Check ServiceMonitor configuration
oc describe servicemonitor tempo-monitor

# Verify service labels match ServiceMonitor selector
oc get service tempo-monitor -o yaml | grep -A10 labels
```

#### Validate Prometheus Configuration
```bash
# Check Prometheus targets
oc port-forward svc/prometheus-operated 9090:9090 -n openshift-user-workload-monitoring &
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="tempo-monitor")'
```

### 3. **Metrics Not Available in Thanos**

#### Check Metric Scraping
```bash
# Verify metrics endpoint is accessible
oc port-forward svc/tempo-monitor 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_

# Check Prometheus scraping
oc logs -n openshift-user-workload-monitoring -l app.kubernetes.io/name=prometheus | grep tempo
```

#### Validate Thanos Configuration
```bash
# Check Thanos Querier access
THANOS_HOST=$(oc get route thanos-querier -n openshift-monitoring -o jsonpath='{.spec.host}')
oc create token e2e-test-metrics-reader | xargs -I {} \
  curl -k -H "Authorization: Bearer {}" \
  "https://$THANOS_HOST/api/v1/label/__name__/values" | grep tempo
```

### 4. **Permission Issues**

#### RBAC Troubleshooting
```bash
# Check service account permissions
oc auth can-i get metrics --as=system:serviceaccount:$NAMESPACE:e2e-test-metrics-reader

# Verify cluster role binding
oc get clusterrolebinding | grep e2e-test-metrics-reader

# Check token validity
oc create token e2e-test-metrics-reader -n $NAMESPACE --duration=1h
```

## Production Monitoring Best Practices

### 1. **Resource Planning**

#### Monitoring Overhead
```yaml
# Account for monitoring resource usage
spec:
  resources:
    total:
      limits:
        memory: 6Gi  # +20% for metrics collection
        cpu: 3000m   # +50% for metrics processing
```

#### Storage Requirements
```bash
# Estimate metrics storage needs
# Default retention: 15 days
# Metrics volume: ~50KB per scrape interval per target
# Calculate: scrapes_per_day * targets * 50KB * retention_days
```

### 2. **Alerting Strategy**

#### Critical Alerts
```yaml
# High-priority alerts for production
- alert: TempoDown
  expr: up{job="tempo-monitor"} == 0
  for: 1m
  severity: critical

- alert: TempoHighErrorRate
  expr: rate(tempo_request_total{status="error"}[5m]) > 0.1
  for: 5m
  severity: warning
```

#### Capacity Planning Alerts
```yaml
- alert: TempoHighMemoryUsage
  expr: container_memory_usage_bytes{pod=~"tempo-monitor-.*"} / container_spec_memory_limit_bytes > 0.8
  for: 10m
  severity: warning

- alert: TempoHighCPUUsage  
  expr: rate(container_cpu_usage_seconds_total{pod=~"tempo-monitor-.*"}[5m]) / container_spec_cpu_quota > 0.8
  for: 15m
  severity: warning
```

### 3. **Dashboard Integration**

#### Grafana Dashboard Configuration
```bash
# Create Grafana data source for OpenShift monitoring
oc get route grafana -n openshift-monitoring
# Use OpenShift monitoring Prometheus as data source

# Import Tempo dashboards
# Use community dashboards: https://grafana.com/orgs/grafana/dashboards
```

### 4. **Compliance and Governance**

#### Metric Retention Policies
```yaml
# Configure retention based on compliance requirements
spec:
  observability:
    metrics:
      retention: "90d"  # Adjust based on compliance needs
```

#### Access Control
```bash
# Implement role-based access to monitoring data
oc create role tempo-metrics-reader --verb=get,list --resource=pods,services
oc create rolebinding tempo-metrics-access --role=tempo-metrics-reader --user=monitoring-team
```

## Related Configurations

- [TempoStack Monitoring](../monitoring/README.md) - Distributed monitoring setup
- [Basic TempoMonolithic](../../e2e/monolithic-memory/README.md) - Non-monitoring setup
- [OpenShift Routes](../monolithic-route/README.md) - External access patterns

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monitoring-monolithic
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires OpenShift cluster administrator privileges to enable user workload monitoring. The test runs sequentially (`concurrent: false`) to avoid conflicts with shared monitoring resources. The monitoring validation may take several minutes as metrics need time to be scraped and become available in Thanos.

