# TempoStack with RED Metrics and Prometheus Alerting

This configuration blueprint demonstrates how to deploy TempoStack with RED (Rate, Errors, Duration) metrics generation and Prometheus alerting integration. This setup showcases production-ready observability monitoring with automated span-to-metrics conversion and alert management for service performance monitoring.

## Overview

This test validates a comprehensive observability stack featuring:
- **RED Metrics Generation**: Automatic conversion of spans to rate, error, and duration metrics
- **Prometheus Integration**: Metrics collection and alerting via OpenShift monitoring
- **Service Performance Monitoring**: Real-time monitoring of service health with Jaeger Monitor tab
- **Automated Alerting**: Alert firing and validation for service performance degradation
- **HotROD Demo Application**: Complete end-to-end trace generation and monitoring

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ HotROD Demo App │───▶│   OTel Collector     │───▶│   TempoStack    │
│ - Frontend      │    │ - Trace Collection   │    │ - Distributors  │
│ - Backend       │    │ - Metrics Export     │    │ - Ingesters     │
│ - Customer      │    └──────────────────────┘    │ - Queriers      │
│ - Driver        │                                └─────────────────┘
└─────────────────┘    ┌──────────────────────┐              │
                       │ Jaeger UI Monitor    │◀─────────────┘
┌─────────────────┐    │ Tab Integration      │
│ Prometheus      │◀───│ - RED Metrics        │
│ - Metrics       │    │ - Thanos Endpoint    │
│ - Alerting      │    └──────────────────────┘
│ - AlertManager  │              │
└─────────────────┘              ▼
          │              ┌──────────────────────┐
          ▼              │ Span-to-Metrics      │
┌─────────────────┐      │ - Duration Buckets   │
│ Alert Firing    │      │ - Call Counts        │
│ Validation      │      │ - Error Rates        │
└─────────────────┘      └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- User Workload Monitoring enabled
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Enable User Workload Monitoring

Configure OpenShift monitoring to collect user-defined metrics:

```bash
oc apply -f 01-install-workload-monitoring.yaml
```

**Reference**: [`01-install-workload-monitoring.yaml`](./01-install-workload-monitoring.yaml)

### Step 2: Deploy MinIO Object Storage

Create the storage backend for trace persistence:

```bash
oc apply -f 00-install-storage.yaml
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 3: Deploy OpenTelemetry Collector

Set up the collector for trace and metrics collection:

```bash
oc apply -f 02-install-otel-collector.yaml
```

This configures the collector for:
- OTLP trace reception
- Metrics export to Prometheus
- Trace forwarding to TempoStack

**Reference**: [`02-install-otel-collector.yaml`](./02-install-otel-collector.yaml)

### Step 4: Deploy TempoStack with RED Metrics

Create TempoStack configured for span-to-metrics conversion:

```bash
oc apply -f 03-install-tempo.yaml
```

**Key RED Metrics Configuration from [`03-install-tempo.yaml`](./03-install-tempo.yaml)**:
- **Monitor Tab Integration**: `monitorTab.enabled: true` enables RED metrics visualization
- **Prometheus Endpoint**: Points to Thanos querier for metrics access
- **Jaeger Query UI**: Enhanced with monitoring capabilities
- **OpenShift Route**: External access for Jaeger UI with monitoring

### Step 5: Deploy HotROD Demo Application

Install the HotROD microservices demo for generating realistic traces:

```bash
oc apply -f 04-install-hotrod.yaml
```

HotROD provides:
- **Realistic Microservices**: Frontend, customer, driver, and route services
- **Trace Generation**: Distributed tracing across service boundaries
- **Performance Patterns**: Normal and error scenarios for testing

**Reference**: [`04-install-hotrod.yaml`](./04-install-hotrod.yaml)

### Step 6: Generate Load and Traces

Create sustained load to generate metrics and trigger alerts:

```bash
oc apply -f 05-install-generate-traces.yaml
```

This creates jobs that:
- Generate continuous traffic to HotROD services
- Create spans with varying latencies
- Trigger error conditions for alert testing

**Reference**: [`05-install-generate-traces.yaml`](./05-install-generate-traces.yaml)

### Step 7: Validate Alert Firing

Deploy job to verify alert generation and firing:

```bash
oc apply -f 06-install-assert-job.yaml
```

This validation job checks for:
- Proper span-to-metrics conversion
- Alert rule evaluation
- AlertManager notification

**Reference**: [`06-install-assert-job.yaml`](./06-install-assert-job.yaml)

### Step 8: Verify RED Metrics Collection

Run the metrics validation script to ensure all metrics are properly collected:

```bash
./check_metrics.sh
```

**Script Details from [`check_metrics.sh`](./check_metrics.sh)**:
- Creates service account for metrics access
- Queries Thanos for span-derived metrics
- Validates presence of RED metrics:
  - `traces_span_metrics_duration_bucket`
  - `traces_span_metrics_duration_count`  
  - `traces_span_metrics_duration_sum`
  - `traces_span_metrics_calls`

### Step 9: Verify Alert Functionality

Run the alert validation script to confirm alert firing:

```bash
./check_alert.sh
```

**Script Details from [`check_alert.sh`](./check_alert.sh)**:
- Queries AlertManager for active alerts
- Validates `SpanREDFrontendAPIRequestLatency` alert
- Confirms proper alert state and firing conditions

## Key Features Demonstrated

### 1. **RED Metrics Generation**
- **Rate**: Request rate per service and endpoint
- **Errors**: Error rate and failure percentage
- **Duration**: Response time distribution and percentiles
- **Automatic Conversion**: Spans automatically converted to metrics

### 2. **Prometheus Integration**
- **Metrics Collection**: Span-derived metrics ingested by Prometheus
- **Query Interface**: PromQL queries for service performance analysis
- **Thanos Integration**: Long-term storage and querying via Thanos
- **Service Discovery**: Automatic discovery of metrics endpoints

### 3. **Alerting and Monitoring**
- **PrometheusRule Creation**: Automatic alert rule generation
- **AlertManager Integration**: Alert routing and notification
- **Performance Thresholds**: Configurable SLA-based alerting
- **Real-time Monitoring**: Immediate notification of performance issues

### 4. **Jaeger UI Enhancement**
- **Monitor Tab**: Integrated metrics view within Jaeger UI
- **Service Performance**: Real-time service health visualization
- **Correlation**: Direct correlation between traces and metrics
- **Unified Experience**: Single interface for traces and metrics

## RED Metrics Details

### Metric Types Generated

1. **Request Rate**:
   ```promql
   rate(traces_span_metrics_calls[5m])
   ```

2. **Error Rate**:
   ```promql
   rate(traces_span_metrics_calls{status_code!="STATUS_CODE_OK"}[5m])
   ```

3. **Duration Percentiles**:
   ```promql
   histogram_quantile(0.95, rate(traces_span_metrics_duration_bucket[5m]))
   ```

### Alert Rules

Example alert for high latency:
```yaml
alert: SpanREDFrontendAPIRequestLatency
expr: histogram_quantile(0.95, rate(traces_span_metrics_duration_bucket[5m])) > 0.1
for: 1m
annotations:
  summary: "High request latency detected"
```

## Monitoring and Troubleshooting

### Verify Metrics Generation

```bash
# Check span-to-metrics processor status
oc logs -l app.kubernetes.io/component=ingester | grep span-metrics

# Query metrics directly
oc port-forward svc/prometheus-user-workload 9090:9090
# Access http://localhost:9090 and query traces_span_metrics_*
```

### Validate Alert Configuration

```bash
# Check PrometheusRule creation
oc get prometheusrule -n chainsaw-redmetrics

# Verify alert rule syntax
oc describe prometheusrule tempo-redmetrics-alerts

# Check AlertManager configuration
oc get alertmanager main -n openshift-monitoring -o yaml
```

### Monitor HotROD Application

```bash
# Check HotROD service health
oc get pods -l app=jaeger-hotrod

# Access HotROD UI
oc port-forward svc/jaeger-hotrod 8080:8080
# Open http://localhost:8080

# Generate manual load
curl http://jaeger-hotrod:8080/dispatch?customer=123
```

### Troubleshoot Common Issues

1. **Missing Metrics**:
   ```bash
   # Check span-to-metrics configuration
   oc describe tempostack redmetrics
   # Look for span-metrics processor configuration
   ```

2. **Alert Not Firing**:
   ```bash
   # Check metric values
   oc exec -n openshift-monitoring prometheus-user-workload-0 -- \
     promtool query instant 'traces_span_metrics_duration_bucket'
   ```

3. **Monitor Tab Not Working**:
   ```bash
   # Verify Thanos endpoint configuration
   oc get route thanos-querier -n openshift-monitoring
   # Check Jaeger UI logs for Prometheus connectivity
   ```

## Production Considerations

### 1. **Metrics Storage and Retention**
- Configure appropriate retention policies for span-derived metrics
- Monitor storage usage for high-cardinality metrics
- Implement metric aggregation for long-term storage

### 2. **Alert Tuning**
- Calibrate alert thresholds based on service SLAs
- Implement alert routing and escalation policies  
- Configure alert suppression during maintenance windows

### 3. **Performance Impact**
- Monitor overhead of span-to-metrics conversion
- Tune metrics generation sample rates
- Consider metrics cardinality limitations

### 4. **Scaling Considerations**
- Scale ingester components for high span volumes
- Configure appropriate resource limits for metrics processing
- Monitor query performance for large metric datasets

## Related Configurations

- [Basic Monitoring](../monitoring/README.md) - Standard monitoring setup without RED metrics
- [Multi-tenancy](../multitenancy/README.md) - Multi-tenant monitoring patterns
- [Component Scaling](../component-replicas/README.md) - Scaling for high-volume metrics
- [TempoStack Compatibility](../../e2e/compatibility/README.md) - Basic TempoStack setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/red-metrics --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs with `concurrent: false` to prevent conflicts with shared OpenShift monitoring resources and requires the HotROD application for realistic load generation.