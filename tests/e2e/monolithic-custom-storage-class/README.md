# TempoMonolithic with Custom Storage Class

This configuration blueprint demonstrates how to deploy TempoMonolithic with a custom Kubernetes storage class for persistent volume provisioning. This setup is essential for environments requiring specific storage performance characteristics, encryption policies, or storage backend types.

## Overview

This test validates persistent storage customization features:
- **Custom Storage Classes**: Integration with specific storage provisioners
- **Storage Size Configuration**: Customizable persistent volume sizing
- **Security Context**: Pod security configuration for storage access
- **Storage Backend Selection**: PV (Persistent Volume) backend for trace storage

## Architecture

```
┌─────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Trace Ingestion │───▶│   TempoMonolithic        │───▶│ Custom Storage      │
│ (Multiple       │    │   StatefulSet            │    │ Class PV            │
│  Protocols)     │    │ ┌─────────────────────┐  │    │ - my-custom-storage │
└─────────────────┘    │ │ Single Pod          │  │    │ - 5Gi Volume        │
                       │ │ - Tempo Binary      │  │◀───│ - Specific          │
┌─────────────────┐    │ │ - Security Context  │  │    │   Provisioner       │
│ Query Interfaces│◀───│ │ - Persistent Data   │  │    └─────────────────────┘
│ - Tempo API     │    │ └─────────────────────┘  │
│ - Health Checks │    └──────────────────────────┘
└─────────────────┘
```

## Prerequisites

- Kubernetes cluster with persistent volume support
- Custom storage class pre-configured (`my-custom-storage` in this example)
- Tempo Operator installed
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Verify Storage Class Availability

Ensure your custom storage class exists and is properly configured:

```bash
# List available storage classes
kubectl get storageclass

# Verify custom storage class details
kubectl describe storageclass my-custom-storage

# Example custom storage class (if not exists)
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: my-custom-storage
provisioner: kubernetes.io/aws-ebs  # or your preferred provisioner
parameters:
  type: gp3
  encrypted: "true"
  fsType: ext4
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
EOF
```

### Step 2: Deploy TempoMonolithic with Custom Storage

Create the TempoMonolithic resource with custom storage configuration:

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
      storageClassName: my-custom-storage
      size: 5Gi
  podSecurityContext:
    runAsUser: 10001
    runAsGroup: 10001
    fsGroup: 10001
EOF
```

**Key Configuration Details**:

#### Storage Configuration
- `backend: pv`: Specifies persistent volume storage instead of in-memory
- `storageClassName: my-custom-storage`: References the custom storage class
- `size: 5Gi`: Allocates 5GB for trace data storage

#### Security Context
- `runAsUser: 10001`: Non-root user ID for enhanced security
- `runAsGroup: 10001`: Non-root group ID 
- `fsGroup: 10001`: File system group for volume permissions

**Reference**: [`01-tempo.yaml`](./01-tempo.yaml)

### Step 3: Verify Deployment and Storage

Wait for the TempoMonolithic to be ready and validate storage configuration:

```bash
# Check TempoMonolithic status
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify StatefulSet creation
kubectl get statefulset tempo-simplest

# Check PVC creation with custom storage class
kubectl get pvc -l app.kubernetes.io/instance=simplest

# Validate security context application
kubectl get statefulset tempo-simplest -o jsonpath='{.spec.template.spec.securityContext}'
```

Expected validation results:
- **StatefulSet**: `tempo-simplest` with 1 ready replica
- **PVC**: Named with custom storage class `my-custom-storage`
- **Security Context**: Applied with specified user/group IDs
- **Volume Size**: 5Gi allocated per the specification

**Reference**: [`02-assert.yaml`](./02-assert.yaml)

## Storage Class Integration Features

### 1. **Custom Provisioner Support**

#### AWS EBS Example
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tempo-high-iops
provisioner: ebs.csi.aws.com
parameters:
  type: io2
  iops: "1000"
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
```

#### GCE Persistent Disk Example
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: tempo-ssd-storage
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-ssd
  replication-type: regional-pd
volumeBindingMode: WaitForFirstConsumer
```

### 2. **Performance Optimization**

#### High-Performance Storage
```yaml
# TempoMonolithic with high-performance storage
spec:
  storage:
    traces:
      backend: pv
      storageClassName: high-performance-ssd
      size: 20Gi
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi
```

#### Large-Scale Storage
```yaml
# TempoMonolithic for high-volume environments
spec:
  storage:
    traces:
      backend: pv
      storageClassName: large-capacity-storage
      size: 100Gi
  template:
    retention: 72h  # 3-day retention
```

### 3. **Security-Enhanced Configuration**

#### Encrypted Storage with Restricted Access
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: secure-tempo
spec:
  storage:
    traces:
      backend: pv
      storageClassName: encrypted-storage
      size: 10Gi
  podSecurityContext:
    runAsUser: 10001
    runAsGroup: 10001
    fsGroup: 10001
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containerSecurityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - ALL
    readOnlyRootFilesystem: true
```

## Storage Configuration Options

### 1. **Backend Types**

#### Persistent Volume (Recommended for Production)
```yaml
storage:
  traces:
    backend: pv
    storageClassName: production-storage
    size: 50Gi
```

#### Memory (Development/Testing Only)
```yaml
storage:
  traces:
    backend: memory
    # No persistent storage - data lost on restart
```

### 2. **Storage Class Selection Criteria**

| Use Case | Provisioner Type | Performance | Cost | Durability |
|----------|------------------|-------------|------|-----------|
| Development | Local/HostPath | Low | Low | Low |
| Testing | Standard SSD | Medium | Medium | Medium |
| Production | Premium SSD | High | High | High |
| Archive | Cold Storage | Low | Very Low | Very High |

### 3. **Volume Sizing Guidelines**

#### Calculation Formula
```
Required Storage = (Trace Rate × Average Trace Size × Retention Period) × Safety Factor

Example:
- Trace Rate: 1000 traces/second
- Average Trace Size: 10KB
- Retention: 24 hours (86400 seconds)
- Safety Factor: 2x

Storage = (1000 × 10KB × 86400) × 2 = ~1.7TB
```

#### Size Recommendations
```yaml
# Small deployment (< 100 spans/second)
size: 5Gi

# Medium deployment (100-1000 spans/second)  
size: 20Gi

# Large deployment (1000+ spans/second)
size: 100Gi
```

## Security Context Configuration

### 1. **Non-Root User Configuration**
```yaml
podSecurityContext:
  runAsUser: 10001        # Non-root user ID
  runAsGroup: 10001       # Non-root group ID  
  fsGroup: 10001          # File system group for volumes
  runAsNonRoot: true      # Enforce non-root execution
```

### 2. **Enhanced Security Profile**
```yaml
podSecurityContext:
  runAsUser: 10001
  runAsGroup: 10001
  fsGroup: 10001
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault   # Apply seccomp profile
  seLinuxOptions:
    level: "s0:c123,c456"  # SELinux context
```

### 3. **Container Security Context**
```yaml
containerSecurityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true
  seccompProfile:
    type: RuntimeDefault
```

## Monitoring and Troubleshooting

### Storage Health Checks

```bash
# Monitor PVC status
kubectl get pvc -l app.kubernetes.io/instance=simplest -w

# Check storage utilization
kubectl exec tempo-simplest-0 -- df -h /var/tempo

# View storage events
kubectl get events --field-selector involvedObject.kind=PersistentVolumeClaim
```

### Performance Monitoring

```bash
# Monitor pod resource usage
kubectl top pod tempo-simplest-0

# Check storage I/O metrics (if node exporter available)
kubectl port-forward tempo-simplest-0 3200:3200
curl localhost:3200/metrics | grep -E "(tempo_ingester_bytes|tempo_storage)"
```

### Common Issues and Solutions

#### 1. **PVC Stuck in Pending**
```bash
# Check storage class availability
kubectl describe pvc tempo-storage-tempo-simplest-0

# Verify node resources
kubectl describe nodes

# Check storage provisioner logs
kubectl logs -n kube-system -l app=ebs-csi-controller
```

#### 2. **Permission Denied Errors**
```bash
# Verify fsGroup application
kubectl exec tempo-simplest-0 -- ls -la /var/tempo

# Check security context
kubectl get pod tempo-simplest-0 -o jsonpath='{.spec.securityContext}'

# Fix ownership if needed (emergency only)
kubectl exec tempo-simplest-0 -- chown -R 10001:10001 /var/tempo
```

#### 3. **Storage Full**
```bash
# Check current usage
kubectl exec tempo-simplest-0 -- du -sh /var/tempo

# Expand PVC if storage class supports it
kubectl patch pvc tempo-storage-tempo-simplest-0 -p '{"spec":{"resources":{"requests":{"storage":"10Gi"}}}}'
```

## Production Considerations

### 1. **Storage Class Design**
- Use provisioners with high IOPS for write-heavy workloads
- Enable encryption for sensitive environments
- Configure appropriate volume binding modes
- Plan for volume expansion capabilities

### 2. **Backup and Recovery**
```bash
# Create volume snapshots (if supported)
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

### 3. **Resource Planning**
- Monitor storage growth trends
- Set up alerts for storage utilization (>80%)
- Plan for retention policy management
- Consider data compression options

### 4. **High Availability**
- Use storage classes with replication
- Consider multi-zone persistent volumes
- Implement backup strategies
- Plan for disaster recovery scenarios

## Related Configurations

- [TempoMonolithic Memory Storage](../monolithic-memory/README.md) - In-memory storage setup
- [TempoMonolithic PV Storage](../monolithic-pv/README.md) - Standard persistent volume setup  
- [TempoStack with Object Storage](../compatibility/README.md) - Distributed deployment with S3

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-custom-storage-class
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: Ensure the `my-custom-storage` storage class exists in your cluster before running this test, or modify the configuration to use an available storage class.

