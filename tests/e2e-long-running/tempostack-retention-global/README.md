# TempoStack Global Retention Policy Validation

This configuration blueprint demonstrates and validates TempoStack's global retention policy functionality for automatic trace data lifecycle management. This long-running test ensures that trace data is automatically purged according to configured retention policies, which is critical for storage management and compliance in production environments.

## Overview

This test validates comprehensive data lifecycle management features:
- **Global Retention Policy**: Automatic deletion of traces after specified time period
- **Storage Lifecycle**: Verification of trace creation, persistence, and deletion
- **Retention Enforcement**: Validation that retention policies are actively enforced
- **Long-Running Validation**: 45-minute test to verify actual retention behavior

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Trace Generation    │───▶│   TempoStack (Global)    │───▶│ MinIO Storage       │
│ - 10 traces         │    │   Retention: 32 minutes  │    │ - S3 Compatible     │
│ - OTLP ingestion    │    │ ┌─────────────────────┐  │    │ - Temporary blocks  │
└─────────────────────┘    │ │ Components          │  │    └─────────────────────┘
                           │ │ - Distributor       │  │
┌─────────────────────┐    │ │ - Ingester          │  │    ┌─────────────────────┐
│ Verification        │◀───│ │ - Querier           │  │───▶│ Retention Timeline  │
│ - Initial: 10 traces│    │ │ - Query Frontend    │  │    │ t=0:  Create traces │
│ - After 45m: 0      │    │ │ - Compactor         │  │    │ t=32m: Auto-delete  │
└─────────────────────┘    │ └─────────────────────┘  │    │ t=45m: Verify empty │
                           │ Jaeger UI (External)     │    └─────────────────────┘
┌─────────────────────┐    └──────────────────────────┘
│ Query Interfaces    │
│ - Jaeger API        │    Retention Process:
│ - Grafana API       │    1. Traces ingested normally
│ - External Access   │    2. Background compaction marks old blocks
└─────────────────────┘    3. Retention policy triggers deletion
                           4. Storage space reclaimed
```

## Prerequisites

- Kubernetes cluster with persistent volume support
- Tempo Operator installed and running
- **Time Availability**: Test requires 45+ minutes to complete
- `kubectl` CLI access
- Understanding of data retention compliance requirements

## Test Timeline and Phases

### Timeline Overview
```
t=0     : Deploy TempoStack with 32m retention
t=5     : Generate and verify traces (10 traces stored)
t=10    : Verify traces via Jaeger API
t=15    : Verify traces via Grafana API
t=45    : Wait period complete (retention + safety margin)
t=50    : Verify traces are automatically deleted (0 traces)
t=55    : Final verification via all query interfaces
```

## Step-by-Step Deployment and Testing

### Step 1: Deploy Storage Backend

Create MinIO for persistent trace storage:

```bash
kubectl apply -f - <<EOF
# Standard MinIO deployment for retention testing
# Reference: 00-install-storage.yaml
EOF
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Deploy TempoStack with Global Retention Policy

Create TempoStack with comprehensive retention configuration:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: global
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 6Gi
        cpu: 2000m
  retention:
    global:
      traces: "32m"
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: ingress
EOF
```

**Key Configuration Elements**:

#### Retention Policy
- `retention.global.traces: "32m"`: Global retention period of 32 minutes
- **Automatic Enforcement**: Background processes handle deletion
- **All Tenants**: Global policy applies to all trace data

#### Resource Allocation
- `memory: 6Gi`: Higher memory allocation for retention processing
- `cpu: 2000m`: Adequate CPU for compaction and deletion operations
- `storageSize: 200M`: Compact storage for test efficiency

#### Query Interface
- **Jaeger UI**: External access for verification
- **Ingress**: External connectivity for retention validation

**Reference**: [`01-install.yaml`](./01-install.yaml)

### Step 3: Wait for TempoStack Readiness

Ensure all components are ready before trace generation:

```bash
# Wait for TempoStack to be fully ready
kubectl get tempostack global -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify all components are running
kubectl get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check retention configuration propagation
kubectl get configmap tempo-global-compactor -o jsonpath='{.data.tempo\.yaml}' | grep -A5 retention
```

### Step 4: Generate Traces for Retention Testing

Create a controlled set of traces for lifecycle validation:

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
        - --otlp-endpoint=tempo-global-distributor:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Generation Details**:
- **Controlled Volume**: Exactly 10 traces for precise validation
- **Timestamp Tracking**: All traces generated at approximately the same time
- **Retention Calculation**: Generated at t=0, should be deleted at t=32m

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 5: Verify Initial Trace Existence

Validate that traces are properly stored before retention period:

```bash
# Verify via Jaeger API
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
          curl -v -G \
            http://tempo-global-query-frontend:16686/api/traces \
            --data-urlencode "service=telemetrygen" | tee /tmp/jaeger.out
          
          num_traces=\$(jq ".data | length" /tmp/jaeger.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          echo "✓ Verified \$num_traces traces present before retention"
      restartPolicy: Never
EOF

# Verify via Grafana API
kubectl apply -f - <<EOF
# Similar verification via Grafana API endpoint
# Reference: 05-verify-traces-grafana.yaml
EOF
```

**Verification Points**:
- **Jaeger API**: Confirms traces accessible via standard Jaeger interface
- **Grafana API**: Validates traces available for dashboard integration
- **Count Validation**: Exactly 10 traces should be present

**References**: [`04-verify-traces-jaeger.yaml`](./04-verify-traces-jaeger.yaml), [`05-verify-traces-grafana.yaml`](./05-verify-traces-grafana.yaml)

### Step 6: Wait for Retention Period

Execute the critical waiting period for retention enforcement:

```bash
# Wait 45 minutes for retention to complete
# (32m retention + 13m safety margin)
echo "Waiting 45 minutes for retention period..."
sleep 2700  # 45 minutes in seconds
```

**Waiting Period Details**:
- **Retention Time**: 32 minutes as configured
- **Safety Margin**: Additional 13 minutes to ensure completion
- **Background Processes**: Compaction and deletion occur automatically
- **No Manual Intervention**: Retention is fully automated

### Step 7: Verify Trace Deletion Post-Retention

Validate that traces are automatically deleted after retention period:

```bash
# Verify deletion via Jaeger API
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces-jaeger-ret
spec:
  template:
    spec:
      containers:
      - name: verify-traces-jaeger-ret
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          curl -v -G \
            http://tempo-global-query-frontend:16686/api/traces \
            --data-urlencode "service=telemetrygen" | tee /tmp/jaeger.out
          
          num_traces=\$(jq ".data | length" /tmp/jaeger.out)
          if [[ "\$num_traces" -ne 0 ]]; then
            echo "Expected 0 traces after retention, got \$num_traces"
            exit 1
          fi
          echo "✓ Verified \$num_traces traces remain after retention (SUCCESS)"
      restartPolicy: Never
EOF

# Verify deletion via Grafana API
kubectl apply -f - <<EOF
# Similar verification via Grafana API to confirm deletion
# Reference: verify-traces-grafana-ret.yaml
EOF
```

**Post-Retention Validation**:
- **Zero Traces**: Exactly 0 traces should remain
- **API Consistency**: Both Jaeger and Grafana APIs should return empty results
- **Storage Cleanup**: Underlying storage blocks should be removed

**References**: [`verify-traces-jaeger-ret.yaml`](./verify-traces-jaeger-ret.yaml), [`verify-traces-grafana-ret.yaml`](./verify-traces-grafana-ret.yaml)

## Retention Policy Configuration

### 1. **Global Retention Settings**

#### Basic Global Retention
```yaml
spec:
  retention:
    global:
      traces: "24h"      # 24-hour retention for all traces
```

#### Extended Retention Periods
```yaml
spec:
  retention:
    global:
      traces: "7d"       # 7-day retention
      # Formats supported: s, m, h, d (seconds, minutes, hours, days)
```

### 2. **Per-Tenant Retention (Advanced)**

```yaml
spec:
  retention:
    perTenant:
      - tenant: "team-a"
        traces: "48h"     # Extended retention for team-a
      - tenant: "team-b"
        traces: "12h"     # Shorter retention for team-b
    global:
      traces: "24h"       # Default for all other tenants
```

### 3. **Retention Policy Inheritance**

```yaml
# Priority order (highest to lowest):
# 1. Per-tenant specific policies
# 2. Global retention policy
# 3. Operator default (48h if not specified)

spec:
  retention:
    global:
      traces: "72h"       # Applied when no per-tenant policy matches
```

## Retention Implementation Details

### 1. **Compaction and Deletion Process**

#### Background Processing
The retention process operates through several phases:

1. **Block Creation**: Ingester creates time-based blocks
2. **Compaction**: Compactor merges and optimizes blocks
3. **Age Evaluation**: Compactor identifies blocks exceeding retention
4. **Deletion**: Old blocks marked for deletion and removed from storage
5. **Cleanup**: Storage space reclaimed

#### Configuration Example
```yaml
spec:
  extraConfig:
    tempo:
      compactor:
        compaction:
          block_retention: 32m           # Must match or exceed trace retention
          compacted_block_retention: 1h  # Retention for compacted blocks
```

### 2. **Timing and Precision**

#### Retention Timing Factors
- **Block Boundaries**: Deletion occurs at block level, not individual traces
- **Compaction Intervals**: Background compaction runs periodically
- **Safety Margins**: System may retain data slightly beyond specified time
- **Clock Synchronization**: Depends on cluster time synchronization

#### Monitoring Retention
```bash
# Check compactor logs for retention activity
kubectl logs tempo-global-compactor-0 | grep -i "retention\|delete\|expired"

# Monitor block count over time
kubectl exec tempo-global-compactor-0 -- \
  find /var/tempo -name "*.gz" | wc -l

# Check storage utilization
kubectl exec tempo-global-compactor-0 -- du -sh /var/tempo
```

### 3. **Storage Impact and Optimization**

#### Storage Efficiency
```yaml
spec:
  storageSize: 200M               # Adequate for retention testing
  extraConfig:
    tempo:
      compactor:
        compaction:
          chunk_size_bytes: 5242880    # 5MB chunks for faster processing
          max_compaction_objects: 100000
      storage:
        trace:
          blocklist_poll: 300s         # 5-minute polling for deletion
```

## Production Retention Strategies

### 1. **Compliance-Driven Retention**

#### GDPR Compliance Example
```yaml
spec:
  retention:
    global:
      traces: "30d"       # 30-day retention for GDPR compliance
    perTenant:
      - tenant: "pii-sensitive"
        traces: "7d"      # Shorter retention for PII-sensitive data
      - tenant: "audit-logs"
        traces: "7y"      # Extended retention for audit requirements
```

#### Financial Services Example
```yaml
spec:
  retention:
    global:
      traces: "2555d"     # 7-year retention (Sarbanes-Oxley)
    perTenant:
      - tenant: "development"
        traces: "24h"     # Short retention for dev environments
      - tenant: "production"
        traces: "2555d"   # Full compliance retention
```

### 2. **Cost Optimization Strategies**

#### Tiered Retention
```yaml
# Use multiple TempoStack instances for different retention tiers
# Short-term (hot): Fast SSD storage, 7-day retention
# Medium-term (warm): Standard storage, 30-day retention  
# Long-term (cold): Archive storage, 365-day retention
```

#### Resource-Based Retention
```yaml
spec:
  retention:
    perTenant:
      - tenant: "high-volume-service"
        traces: "6h"      # Shorter retention for high-volume data
      - tenant: "critical-service"
        traces: "30d"     # Extended retention for critical services
```

### 3. **Monitoring and Alerting**

#### Retention Monitoring
```yaml
# Prometheus alerts for retention policy validation
alert: TempoRetentionPolicyViolation
expr: tempo_compactor_blocks_retention_duration_seconds > (retention_policy_seconds * 1.2)
for: 15m
annotations:
  summary: "Tempo retention policy may not be working correctly"

alert: TempoStorageGrowthUnexpected
expr: increase(tempo_storage_size_bytes[24h]) > expected_daily_growth_bytes
for: 1h
annotations:
  summary: "Tempo storage growing faster than expected - check retention"
```

#### Compliance Reporting
```bash
# Generate retention compliance reports
kubectl exec tempo-global-compactor-0 -- \
  find /var/tempo -name "*.gz" -exec stat -f "%Y %N" {} \; | \
  awk -v now=$(date +%s) -v retention=1920 '{ if ((now - $1) > retention) print "VIOLATION: " $2 }'
```

## Troubleshooting Retention Issues

### 1. **Retention Not Working**

#### Check Compactor Health
```bash
# Verify compactor is running and healthy
kubectl get pods -l app.kubernetes.io/component=compactor

# Check compactor logs for errors
kubectl logs tempo-global-compactor-0 | grep -i "error\|failed\|retention"

# Verify compactor configuration
kubectl get configmap tempo-global-compactor -o jsonpath='{.data.tempo\.yaml}' | yq '.compactor'
```

#### Validate Retention Configuration
```bash
# Check TempoStack retention settings
kubectl get tempostack global -o jsonpath='{.spec.retention}'

# Verify configuration propagation
kubectl get configmap tempo-global-compactor -o jsonpath='{.data.tempo\.yaml}' | grep -A5 retention
```

### 2. **Traces Not Being Deleted**

#### Block Management Issues
```bash
# Check block ages in storage
kubectl exec tempo-global-compactor-0 -- \
  find /var/tempo -name "*.gz" -exec stat -f "%Y %N" {} \; | \
  sort -n

# Monitor compaction activity
kubectl logs tempo-global-compactor-0 | grep -i "compaction\|block"

# Check for storage errors
kubectl logs tempo-global-compactor-0 | grep -i "s3\|storage\|error"
```

#### Configuration Validation
```bash
# Verify retention math
echo "Retention: 32 minutes = $((32 * 60)) seconds"
date -u  # Check current time
kubectl exec tempo-global-compactor-0 -- date -u  # Check pod time

# Test storage connectivity
kubectl exec tempo-global-compactor-0 -- \
  curl -v http://minio:9000/minio/health/live
```

### 3. **Performance Impact**

#### Compaction Performance
```bash
# Monitor compaction metrics
kubectl port-forward svc/tempo-global-compactor 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_compactor

# Check resource utilization during retention
kubectl top pod tempo-global-compactor-0

# Monitor storage I/O
kubectl exec tempo-global-compactor-0 -- iostat -x 1 3
```

## Test Execution

### Manual Test Execution

```bash
# Full test (requires 45+ minutes)
chainsaw test --test-dir ./tests/e2e-long-running/tempostack-retention-global

# Accelerated testing (modify retention to 2m for faster validation)
# Edit 01-install.yaml: change "32m" to "2m"
# Edit chainsaw-test.yaml: change "45m" to "5m"
chainsaw test --test-dir ./tests/e2e-long-running/tempostack-retention-global
```

### CI/CD Considerations

```yaml
# Example CI configuration for retention testing
name: Long-Running Retention Test
on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly at 2 AM Sunday
jobs:
  retention-test:
    timeout-minutes: 60   # Account for 45m wait + setup time
    steps:
      - name: Run retention test
        run: chainsaw test --test-dir ./tests/e2e-long-running/tempostack-retention-global
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This is a time-intensive test requiring 45+ minutes to complete. The test validates critical data lifecycle functionality essential for production compliance and storage management.

