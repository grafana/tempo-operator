# TempoMonolithic with Persistent Volume Storage

This configuration blueprint demonstrates how to deploy TempoMonolithic with persistent volume storage using the default storage class. This setup provides data persistence for trace storage while maintaining simplicity in configuration, making it ideal for development and testing environments that require data durability.

## Overview

This test validates persistent storage fundamentals:
- **Default Storage Class**: Uses cluster's default storage provisioner
- **Data Persistence**: Traces survive pod restarts and rescheduling
- **Simplified Configuration**: Minimal persistent volume setup
- **Production Readiness**: Foundation for production-grade deployments

## Architecture

```
┌─────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Trace Generator │───▶│   TempoMonolithic        │───▶│ Persistent Volume   │
│ (telemetrygen)  │    │   StatefulSet            │    │ - Default Storage   │
└─────────────────┘    │ ┌─────────────────────┐  │    │ - Auto-provisioned  │
                       │ │ Single Pod          │  │    │ - Durable Storage   │
┌─────────────────┐    │ │ - Tempo Binary      │  │◀───│                     │
│ Query Interface │◀───│ │ - Persistent Data   │  │    └─────────────────────┘
│ - Search API    │    │ │ - WAL + Blocks      │  │
│ - Trace Storage │    │ └─────────────────────┘  │
└─────────────────┘    └──────────────────────────┘
```

## Prerequisites

- Kubernetes cluster with a default storage class configured
- Persistent volume provisioning capability
- Tempo Operator installed
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Verify Default Storage Class

Ensure your cluster has a default storage class for automatic volume provisioning:

```bash
# Check available storage classes
kubectl get storageclass

# Identify default storage class (marked with "default" annotation)
kubectl get storageclass -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'

# If no default storage class exists, set one
kubectl patch storageclass <storage-class-name> \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

### Step 2: Deploy TempoMonolithic with Persistent Storage

Create the TempoMonolithic resource with persistent volume backend:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  storage:
    traces:
      backend: pv
EOF
```

**Key Configuration Details**:

#### Storage Configuration
- `backend: pv`: Specifies persistent volume storage instead of in-memory
- **No storage class specified**: Uses cluster's default storage class
- **No size specified**: Uses operator's default size allocation
- **No security context**: Uses default pod security settings

#### Automatic Provisioning
- **PVC Creation**: Operator automatically creates PersistentVolumeClaim
- **Volume Mounting**: Storage mounted at `/var/tempo` in the pod
- **Default Sizing**: Typically 10Gi (operator default)

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Verify Deployment and Storage

Validate that TempoMonolithic is ready with persistent storage:

```bash
# Check TempoMonolithic status
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify StatefulSet creation
kubectl get statefulset tempo-simplest

# Check PVC creation and binding
kubectl get pvc -l app.kubernetes.io/instance=simplest

# Verify volume mount in pod
kubectl describe pod tempo-simplest-0 | grep -A5 "Mounts:"
```

Expected validation results:
- **StatefulSet**: `tempo-simplest` with 1 ready replica
- **PVC**: Auto-created with default storage class
- **Volume Status**: Bound to a persistent volume
- **Mount Point**: Storage accessible at `/var/tempo`

### Step 4: Generate Sample Traces

Create traces to populate the persistent storage:

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
- `--otlp-endpoint=tempo-simplest:4317`: Direct connection to TempoMonolithic
- `--traces=10`: Generates exactly 10 traces for verification
- `--otlp-insecure`: Uses unencrypted connection for testing

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 5: Verify Trace Persistence

Test that traces are properly stored and retrievable:

```bash
kubectl apply -f - <<EOF
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
          curl \
            -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" \
            | tee /tmp/tempo.out
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo && echo "The Tempo API returned \$num_traces instead of 10 traces."
            exit 1
          fi
          echo "✓ Successfully retrieved \$num_traces traces from persistent storage"
      restartPolicy: Never
EOF
```

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

### Step 6: Test Data Persistence (Optional)

Verify that data survives pod restarts:

```bash
# Delete the TempoMonolithic pod to trigger restart
kubectl delete pod tempo-simplest-0

# Wait for pod to restart
kubectl wait --for=condition=Ready pod/tempo-simplest-0 --timeout=300s

# Re-run trace verification to ensure data persisted
kubectl delete job verify-traces
kubectl apply -f 04-verify-traces.yaml

# Check that traces are still available after restart
kubectl logs job/verify-traces
```

## Persistent Volume Configuration

### 1. **Default Storage Behavior**

#### Automatic PVC Creation
```yaml
# Generated PVC (created by operator)
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tempo-storage-tempo-simplest-0
  labels:
    app.kubernetes.io/instance: simplest
    app.kubernetes.io/managed-by: tempo-operator
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi  # Default size
  # storageClassName: (default storage class)
```

#### Volume Mount Configuration
```yaml
# Volume mount in StatefulSet
volumeMounts:
- mountPath: /var/tempo
  name: storage
volumes:
- name: storage
  persistentVolumeClaim:
    claimName: tempo-storage-tempo-simplest-0
```

### 2. **Storage Structure**

#### Tempo Data Layout
```
/var/tempo/
├── blocks/           # Compressed trace blocks
├── wal/             # Write-ahead log
├── generator/       # Metrics generator data (if enabled)
└── overrides/       # Runtime configuration overrides
```

#### Block Storage Organization
```
/var/tempo/blocks/
├── single-tenant/
│   ├── 2024-01-15/
│   │   ├── 14:00:00-block1.tar.gz
│   │   ├── 14:30:00-block2.tar.gz
│   │   └── ...
│   └── 2024-01-16/
└── meta.json        # Block metadata
```

### 3. **Storage Monitoring**

#### Disk Usage Monitoring
```bash
# Check storage utilization
kubectl exec tempo-simplest-0 -- df -h /var/tempo

# Monitor storage growth
kubectl exec tempo-simplest-0 -- du -sh /var/tempo/*

# Check available space
kubectl exec tempo-simplest-0 -- df /var/tempo | awk 'NR==2{printf "%.2f%% used\n", $3/$2*100}'
```

#### Storage Performance Metrics
```bash
# Check I/O statistics
kubectl exec tempo-simplest-0 -- iostat -x 1 3

# Monitor storage latency through Tempo metrics
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_ingester_flush
```

## Configuration Customization

### 1. **Storage Size Configuration**

#### Custom Storage Size
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  storage:
    traces:
      backend: pv
      size: 50Gi  # Custom storage size
```

#### Size Planning Guidelines
```yaml
# Small deployment (< 100 spans/second)
size: 10Gi

# Medium deployment (100-1000 spans/second)
size: 50Gi

# Large deployment (1000+ spans/second)
size: 200Gi
```

### 2. **Performance Optimization**

#### Resource Allocation
```yaml
spec:
  storage:
    traces:
      backend: pv
      size: 100Gi
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi
```

#### Block Configuration
```yaml
spec:
  extraConfig:
    tempo:
      ingester:
        max_block_duration: 1h     # Larger blocks for better compression
        max_block_bytes: 104857600 # 100MB blocks
      compactor:
        compaction:
          block_retention: 168h    # 7-day retention
```

### 3. **Security Configuration**

#### Pod Security Context
```yaml
spec:
  storage:
    traces:
      backend: pv
  podSecurityContext:
    runAsUser: 10001
    runAsGroup: 10001
    fsGroup: 10001
    runAsNonRoot: true
```

#### Read-Only Root Filesystem
```yaml
spec:
  containerSecurityContext:
    readOnlyRootFilesystem: true
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - ALL
```

## Troubleshooting

### 1. **PVC Issues**

#### PVC Stuck in Pending State
```bash
# Check PVC status
kubectl describe pvc tempo-storage-tempo-simplest-0

# Check storage class availability
kubectl get storageclass

# Verify node resources
kubectl describe nodes | grep -A5 "Allocatable:"

# Check storage provisioner logs
kubectl logs -n kube-system -l app=efs-csi-controller  # AWS EFS example
```

#### Volume Mount Failures
```bash
# Check pod events
kubectl describe pod tempo-simplest-0

# Verify mount point
kubectl exec tempo-simplest-0 -- mount | grep tempo

# Check permissions
kubectl exec tempo-simplest-0 -- ls -la /var/tempo
```

### 2. **Storage Performance Issues**

#### Slow I/O Performance
```bash
# Test write performance
kubectl exec tempo-simplest-0 -- dd if=/dev/zero of=/var/tempo/test bs=1M count=100

# Test read performance
kubectl exec tempo-simplest-0 -- dd if=/var/tempo/test of=/dev/null bs=1M

# Monitor I/O wait time
kubectl exec tempo-simplest-0 -- top -bn1 | grep "wa"
```

#### High Storage Utilization
```bash
# Check disk usage by directory
kubectl exec tempo-simplest-0 -- du -sh /var/tempo/blocks/*/

# Clean up old blocks (if needed)
kubectl exec tempo-simplest-0 -- find /var/tempo/blocks -name "*.gz" -mtime +7 -delete

# Check compaction status
kubectl exec tempo-simplest-0 -- ls -la /var/tempo/blocks/
```

### 3. **Data Persistence Verification**

#### Test Pod Restart Persistence
```bash
# Record current trace count
TRACE_COUNT=$(kubectl exec tempo-simplest-0 -- find /var/tempo/blocks -name "*.gz" | wc -l)

# Restart pod
kubectl delete pod tempo-simplest-0
kubectl wait --for=condition=Ready pod/tempo-simplest-0

# Verify trace count unchanged
NEW_COUNT=$(kubectl exec tempo-simplest-0 -- find /var/tempo/blocks -name "*.gz" | wc -l)
echo "Before: $TRACE_COUNT, After: $NEW_COUNT"
```

## Production Considerations

### 1. **Storage Planning**
- Calculate storage requirements based on trace volume and retention
- Plan for storage growth over time
- Implement storage monitoring and alerting
- Consider storage tiering for cost optimization

### 2. **Backup and Recovery**
```bash
# Create volume snapshots (if supported by storage class)
kubectl apply -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: tempo-backup-$(date +%Y%m%d)
spec:
  source:
    persistentVolumeClaimName: tempo-storage-tempo-simplest-0
  volumeSnapshotClassName: csi-snapclass
EOF
```

### 3. **High Availability**
- Consider multiple replicas with shared storage (if supported)
- Implement regular backup procedures
- Plan for disaster recovery scenarios
- Use regional persistent volumes for multi-zone deployments

### 4. **Performance Optimization**
- Use SSD storage for better I/O performance
- Configure appropriate block sizes and retention policies
- Monitor storage metrics and adjust as needed
- Consider storage caching solutions

## Storage Class Examples

### 1. **AWS EBS**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tempo-ssd
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

### 2. **GCP Persistent Disk**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tempo-ssd
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-ssd
  replication-type: regional-pd
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

## Related Configurations

- [Custom Storage Class](../monolithic-custom-storage-class/README.md) - Specific storage class configuration
- [TempoMonolithic Memory](../monolithic-memory/README.md) - In-memory storage setup
- [TempoStack Persistent Storage](../compatibility/README.md) - Distributed persistent storage

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-pv
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires a cluster with a default storage class configured for automatic volume provisioning.

