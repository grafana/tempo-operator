# Operator Metrics - Max Loops Test

This test validates that the Tempo Operator's reconciliation controllers are operating within acceptable limits by monitoring their metrics for excessive reconciliation activity. It ensures the operator doesn't enter infinite reconciliation loops or exhibit performance degradation.

## Test Overview

### Purpose
- **Performance Monitoring**: Validates controller reconciliation frequency
- **Loop Detection**: Ensures no infinite reconciliation loops occur
- **Stability Verification**: Confirms operator behaves predictably under normal conditions

### Metrics Monitored
- `controller_runtime_reconcile_total`: Total number of reconciliations per controller
- Success counts for `tempomonolithic` and `tempostack` controllers
- Configurable thresholds for acceptable reconciliation counts

## Test Components

### 1. Service Account Setup
From [`00-metrics-service.yaml`](00-metrics-service.yaml):
- Creates `sa-assert-metrics` ServiceAccount in the operator namespace
- Grants `tempo-operator-metrics-reader` ClusterRole for metrics access
- Enables secure metrics endpoint access

### 2. Metrics Verification Job
From [`01-verify-metrics.yaml`](01-verify-metrics.yaml):
- Deploys Job to scrape operator metrics endpoint
- Validates reconciliation counts against configurable thresholds
- Uses `ghcr.io/grafana/tempo-operator/test-utils:main` image

## Deployment Steps

### 1. Configure Metrics Access
```bash
kubectl apply -f 00-metrics-service.yaml
```

This step:
- Creates necessary RBAC for metrics access
- Sets up ServiceAccount for the verification job
- Ensures secure communication with metrics endpoint

### 2. Execute Metrics Verification
```bash
kubectl apply -f 01-verify-metrics.yaml
```

This step:
- Scrapes metrics from `tempo-operator-controller-manager-metrics-service:8443/metrics`
- Parses `controller_runtime_reconcile_total` metrics
- Validates success counts against thresholds

## Threshold Configuration

### Default Thresholds
- **TempoMonolithic Controller**: 1000 successful reconciliations
- **TempoStack Controller**: 1000 successful reconciliations

### Environment Variables
```yaml
env:
  - name: TEMPOMONOLITHIC_THRESHOLD
    value: "1000"
  - name: TEMPOSTACK_THRESHOLD
    value: "1000"
```

These can be adjusted based on test duration and expected operator activity.

## Metrics Analysis

The test examines the following metrics pattern:
```
controller_runtime_reconcile_total{controller="tempomonolithic",result="success"} 245
controller_runtime_reconcile_total{controller="tempostack",result="success"} 198
```

### Success Criteria
- ✅ All controller success counts below configured thresholds
- ✅ No excessive reconciliation activity detected
- ✅ Operator performance within acceptable limits

### Failure Scenarios
- ❌ Controller success count exceeds threshold (indicates potential loop)
- ❌ Unable to access metrics endpoint (RBAC or networking issues)
- ❌ Unexpected metrics format or missing controllers

## Key Validations

### Controller Health
- ✅ TempoMonolithic controller reconciliation frequency
- ✅ TempoStack controller reconciliation frequency
- ✅ Overall operator stability and performance

### Performance Baseline
- ✅ Establishes acceptable reconciliation count baselines
- ✅ Provides early warning for performance degradation
- ✅ Validates controller efficiency over time

### Monitoring Integration
- ✅ Demonstrates proper metrics endpoint configuration
- ✅ Validates RBAC setup for monitoring access
- ✅ Confirms metrics format and availability

## Architecture

```
[Test Job] 
    ↓ (HTTPS + Bearer Token)
[Metrics Service:8443]
    ↓
[Controller Manager Metrics]
    ↓
[Prometheus Metrics Format]
    ↓
[Reconciliation Counters]
```

## Troubleshooting

### High Reconciliation Counts
If reconciliation counts exceed thresholds:
1. Check for resource conflicts or validation issues
2. Review operator logs for error patterns
3. Verify Custom Resource specifications
4. Monitor resource update frequency

### Metrics Access Issues
If metrics cannot be accessed:
1. Verify RBAC permissions for ServiceAccount
2. Check metrics service endpoint availability
3. Validate TLS certificate configuration
4. Confirm operator pod health and status

This test serves as a performance guardian for the Tempo Operator, ensuring it maintains efficient and stable reconciliation behavior across different controller types and operational scenarios.

