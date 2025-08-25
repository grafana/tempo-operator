# TempoStack with OpenShift Monitoring Integration

This configuration blueprint demonstrates how to deploy TempoStack with comprehensive OpenShift monitoring integration, including Prometheus metrics collection, service monitoring, and alerting rules. This setup showcases production-ready observability monitoring for enterprise environments.

## Overview

This test validates a fully monitored observability stack featuring:
- **OpenShift User Workload Monitoring**: Integration with OpenShift's monitoring stack
- **Automated ServiceMonitor Creation**: Prometheus target discovery for all Tempo components
- **PrometheusRule Generation**: Built-in alerting rules for Tempo health monitoring
- **Operator Monitoring**: Comprehensive monitoring of the Tempo Operator itself
- **Route-based Access**: OpenShift route integration for external access

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ Trace Generator │───▶│    TempoStack        │───▶│ MinIO Storage   │
│ (telemetrygen)  │    │ ┌─────────────────┐  │    │ (S3 Compatible) │
└─────────────────┘    │ │ All Components  │  │    └─────────────────┘
                       │ │ with Metrics    │  │
┌─────────────────┐    │ └─────────────────┘  │
│ External Access │◀───│ OpenShift Route      │
│ via Route       │    │ (Jaeger UI)          │
└─────────────────┘    └──────────────────────┘
                                │
┌─────────────────┐            │ metrics
│ OpenShift       │◀───────────┘
│ Monitoring      │    ┌──────────────────────┐
│ - Prometheus    │    │   ServiceMonitors    │
│ - Thanos        │◀───│ - Distributor        │
│ - Alertmanager  │    │ - Ingester           │
└─────────────────┘    │ - Querier            │
                       │ - Query Frontend     │
                       │ - Compactor          │
                       │ - Operator           │
                       └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+)
- Tempo Operator installed
- User Workload Monitoring enabled
- Cluster admin privileges for monitoring setup
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

Verify monitoring is enabled:

```bash
# Check user workload monitoring pods
oc get pods -n openshift-user-workload-monitoring

# Verify Thanos querier is accessible
oc get route thanos-querier -n openshift-monitoring
```

**Reference**: [`01-workload-monitoring.yaml`](./01-workload-monitoring.yaml)

### Step 2: Deploy MinIO Object Storage

Create the storage backend:

```bash
# Apply storage configuration
oc apply -f - <<EOF
# MinIO deployment with PVC, service, and secret
# Reference: 00-install-storage.yaml
EOF
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 3: Enable Operator Monitoring

Configure monitoring for the Tempo Operator namespace:

```bash
# Get operator namespace
TEMPO_NAMESPACE=$(oc get pods -A -l control-plane=controller-manager -l app.kubernetes.io/name=tempo-operator -o jsonpath='{.items[0].metadata.namespace}')

# Enable monitoring for operator namespace
oc label namespace $TEMPO_NAMESPACE openshift.io/cluster-monitoring=true
```

This enables monitoring of:
- Tempo Operator controller metrics
- Operator resource utilization
- Controller reconciliation performance

### Step 4: Deploy TempoStack with Monitoring

Create TempoStack with comprehensive monitoring configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: tempostack
  namespace: chainsaw-monitoring
spec:
  observability:
    metrics:
      createPrometheusRules: true
      createServiceMonitors: true
  resources:
    total:
      limits:
        cpu: 2000m
        memory: 2Gi
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          route:
            termination: edge
          type: route
  storage:
    secret:
      type: s3
      name: minio-secret
  storageSize: 10Gi
EOF
```

**Key Monitoring Configuration**:

#### Observability Settings
- `createPrometheusRules: true`: Creates alerting rules for Tempo health
- `createServiceMonitors: true`: Enables automatic metrics collection

#### Route Configuration
- `ingress.type: route`: Creates OpenShift route for external access
- `route.termination: edge`: TLS termination at the route level

**Reference**: [`02-install-tempostack.yaml`](./02-install-tempostack.yaml)

### Step 5: Verify Monitoring Resources

Check that monitoring resources are created:

```bash
# Verify ServiceMonitors
oc get servicemonitor -n chainsaw-monitoring

# Check PrometheusRules
oc get prometheusrule -n chainsaw-monitoring

# Verify operator ServiceMonitor
oc get servicemonitor -n $TEMPO_NAMESPACE
```

Expected ServiceMonitors:
- `tempo-tempostack-compactor`
- `tempo-tempostack-distributor`
- `tempo-tempostack-ingester`
- `tempo-tempostack-querier`
- `tempo-tempostack-query-frontend`

### Step 6: Generate Sample Traces

Create traces to populate metrics:

```bash
oc apply -f - <<EOF
# Trace generation job
# Reference: 03-generate-traces.yaml
EOF
```

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 7: Verify Traces and Metrics

Test trace functionality and metrics collection:

```bash
oc apply -f - <<EOF
# Trace verification job
# Reference: 04-verify-traces.yaml
EOF
```

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

### Step 8: Validate Metrics Collection

Run the comprehensive metrics validation script:

```bash
#!/bin/bash

# Create service account for metrics access
oc create serviceaccount e2e-test-metrics-reader -n $NAMESPACE
oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:$NAMESPACE:e2e-test-metrics-reader

# Get access credentials
TOKEN=$(oc create token e2e-test-metrics-reader -n $NAMESPACE)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

# Validate key Tempo metrics
metrics=(
  "tempo_query_frontend_queries_total"
  "tempo_request_duration_seconds_count"
  "tempo_request_duration_seconds_sum"
  "tempo_request_duration_seconds_bucket"
  "tempo_build_info"
  "tempo_ingester_bytes_received_total"
  "tempo_ingester_flush_failed_retries_total"
  "tempo_ingester_failed_flushes_total"
  "tempo_ring_members"
  "tempo_operator_tempostack_managed"
  "tempo_operator_tempostack_storage_backend"
  "tempo_operator_tempostack_multi_tenancy"
)

for metric in "${metrics[@]}"; do
  echo "Checking metric: $metric"
  
  while true; do
    response=$(curl -k -H "Authorization: Bearer $TOKEN" \
      -H "Content-type: application/json" \
      "https://$THANOS_QUERIER_HOST/api/v1/query?query=$metric")
    
    count=$(echo "$response" | jq -r '.data.result | length')
    
    if [[ $count -eq 0 ]]; then
      echo "No metric '$metric' with value present. Retrying..."
      sleep 5
    else
      echo "✓ Metric '$metric' with value is present."
      break
    fi
  done
done

echo "All metrics validation completed successfully!"
```

**Reference**: [`check_metrics.sh`](./check_metrics.sh)

## Key Features Demonstrated

### 1. **Comprehensive Metrics Collection**
- **Component Metrics**: All Tempo components expose Prometheus metrics
- **Operator Metrics**: Tempo Operator exposes management and reconciliation metrics
- **Custom Metrics**: Application-specific observability metrics

### 2. **Automated Service Discovery**
- **ServiceMonitor Resources**: Automatic Prometheus target configuration
- **Label-based Discovery**: Consistent labeling for metric collection
- **Multi-namespace Support**: Metrics from operator and workload namespaces

### 3. **Built-in Alerting**
- **PrometheusRule Creation**: Pre-configured alerting rules
- **Health Monitoring**: Component availability and performance alerts
- **Threshold-based Alerts**: Configurable warning and critical thresholds

### 4. **OpenShift Integration**
- **User Workload Monitoring**: Integration with OpenShift monitoring stack
- **Route-based Access**: Native OpenShift routing for external access
- **Cluster Monitoring**: Operator monitoring via cluster monitoring

## Monitoring Metrics Reference

### Core Tempo Metrics

#### Query Frontend Metrics
```promql
# Total queries processed
tempo_query_frontend_queries_total

# Query duration distribution
tempo_request_duration_seconds_bucket
tempo_request_duration_seconds_count
tempo_request_duration_seconds_sum
```

#### Ingester Metrics
```promql
# Bytes received for ingestion
tempo_ingester_bytes_received_total

# Flush operation metrics
tempo_ingester_flush_failed_retries_total
tempo_ingester_failed_flushes_total

# Ring membership
tempo_ring_members
```

#### Operator Metrics
```promql
# Number of managed TempoStacks
tempo_operator_tempostack_managed

# Storage backend information
tempo_operator_tempostack_storage_backend

# Multi-tenancy configuration
tempo_operator_tempostack_multi_tenancy
```

### Alerting Rules

The following alerts are automatically created:

#### TempoRequestErrors
```yaml
alert: TempoRequestErrors
expr: |
  (
    sum(rate(tempo_request_duration_seconds_count{status_code!~"2.."}[5m])) by (namespace)
    /
    sum(rate(tempo_request_duration_seconds_count[5m])) by (namespace)
  ) > 0.10
for: 15m
annotations:
  summary: "Tempo request error rate is above 10%"
```

#### TempoIngesterFlushes
```yaml
alert: TempoIngesterFlushes
expr: |
  rate(tempo_ingester_failed_flushes_total[5m]) > 0
for: 5m
annotations:
  summary: "Tempo ingester is experiencing failed flushes"
```

## Accessing Monitoring Data

### Prometheus Queries

Access Prometheus via OpenShift console or API:

```bash
# Get Thanos querier route
THANOS_HOST=$(oc get route thanos-querier -n openshift-monitoring -o jsonpath='{.spec.host}')

# Create metrics reader token
TOKEN=$(oc create token e2e-test-metrics-reader -n chainsaw-monitoring)

# Query specific metrics
curl -k -H "Authorization: Bearer $TOKEN" \
  "https://$THANOS_HOST/api/v1/query?query=tempo_ingester_bytes_received_total"
```

### Grafana Integration

To visualize metrics in Grafana:

1. **Add Prometheus Data Source**:
   ```
   URL: https://thanos-querier-openshift-monitoring.apps.cluster.example.com
   Auth: Bearer Token
   ```

2. **Import Tempo Dashboard**:
   - Use Grafana dashboard ID: 11906 (Tempo Distributed)
   - Configure data source as OpenShift Prometheus

3. **Custom Queries**:
   ```promql
   # Ingestion rate by component
   sum(rate(tempo_ingester_bytes_received_total[5m])) by (instance)
   
   # Query latency percentiles
   histogram_quantile(0.95, 
     sum(rate(tempo_request_duration_seconds_bucket[5m])) by (le)
   )
   ```

## Troubleshooting

### Check Monitoring Configuration

```bash
# Verify user workload monitoring is enabled
oc get configmap cluster-monitoring-config -n openshift-monitoring -o yaml

# Check monitoring operator logs
oc logs -n openshift-user-workload-monitoring deployment/prometheus-operator
```

### Validate ServiceMonitors

```bash
# List all ServiceMonitors
oc get servicemonitor -A

# Check ServiceMonitor configuration
oc describe servicemonitor tempo-tempostack-distributor -n chainsaw-monitoring
```

### Test Metrics Endpoints

```bash
# Port-forward to component
oc port-forward svc/tempo-tempostack-distributor 3200:3200 -n chainsaw-monitoring

# Test metrics endpoint
curl http://localhost:3200/metrics
```

### Debug Missing Metrics

```bash
# Check Prometheus targets
oc get prometheus -n openshift-user-workload-monitoring -o yaml

# View Prometheus configuration
oc exec -n openshift-user-workload-monitoring prometheus-user-workload-0 -- \
  cat /etc/prometheus/config_out/prometheus.env.yaml
```

### Common Issues

1. **ServiceMonitor Not Discovered**:
   ```bash
   # Check label selectors
   oc get servicemonitor tempo-tempostack-distributor -o yaml
   
   # Verify service labels
   oc get svc tempo-tempostack-distributor -o yaml
   ```

2. **Metrics Not Appearing**:
   ```bash
   # Check if metrics endpoint is reachable
   oc exec deployment/tempo-tempostack-distributor -- \
     curl localhost:3200/metrics
   ```

3. **Permission Issues**:
   ```bash
   # Verify RBAC for monitoring
   oc auth can-i get servicemonitors --as=system:serviceaccount:openshift-monitoring:prometheus-user-workload
   ```

## Production Considerations

### 1. **Resource Planning**
- Monitor Prometheus storage requirements
- Plan for metrics retention (default 15 days)
- Consider long-term storage solutions

### 2. **Alert Management**
- Configure AlertManager routing rules
- Set up notification channels (email, Slack, PagerDuty)
- Implement escalation policies

### 3. **Performance Optimization**
- Tune scrape intervals based on traffic
- Configure metric filtering for high-cardinality metrics
- Use recording rules for expensive queries

### 4. **Security**
- Secure Prometheus endpoints with proper RBAC
- Use service accounts for automated access
- Implement network policies for monitoring traffic

## Related Configurations

- [Multi-tenancy Monitoring](../multitenancy/README.md) - Multi-tenant monitoring setup
- [Basic TempoStack](../../e2e/compatibility/README.md) - Base TempoStack configuration
- [OpenShift OSSM Integration](../../e2e-openshift-ossm/README.md) - Service mesh monitoring

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monitoring --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs with `concurrent: false` to prevent conflicts with shared OpenShift monitoring resources.