# TempoStack Operator Reconciliation Behavior

This configuration blueprint demonstrates and validates the Tempo Operator's reconciliation capabilities, which are fundamental to maintaining desired state in Kubernetes environments. This test ensures the operator properly detects changes, recovers from resource deletion, and responds to configuration updates while maintaining system stability.

## Overview

This test validates critical operator reconciliation features:
- **Resource Recreation**: Automatic restoration of deleted Kubernetes resources
- **Configuration Updates**: Dynamic response to secret and configuration changes
- **Reconciliation Control**: Ability to enable/disable reconciliation behavior
- **State Consistency**: Maintaining desired state across system changes

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Storage Secret      │───▶│   TempoStack             │───▶│ MinIO Storage       │
│ - Initial: tempo    │    │   Reconciliation Loop    │    │ - Bucket: tempo     │
│ - Updated: tempo2   │    │ ┌─────────────────────┐  │    │ - Bucket: tempo2    │
└─────────────────────┘    │ │ Watch Events        │  │    └─────────────────────┘
                           │ │ - Secret Changes    │  │
┌─────────────────────┐    │ │ - Resource Delete   │  │    ┌─────────────────────┐
│ Resource Deletion   │───▶│ │ - Config Updates    │  │───▶│ Configuration       │
│ (Service)           │    │ └─────────────────────┘  │    │ Update              │
└─────────────────────┘    │ Auto-Recreation         │    │ - Bucket Change     │
                           └──────────────────────────┘    │ - Resource Sync     │
┌─────────────────────┐                                    └─────────────────────┘
│ Reconciliation      │    Reconciliation States:
│ Control             │    ✅ Enabled  → Active monitoring and updates
│ - Enable/Disable    │    ❌ Disabled → No automatic changes
└─────────────────────┘
```

## Prerequisites

- Kubernetes cluster with persistent volume support
- Tempo Operator installed and running
- `kubectl` CLI access
- Understanding of Kubernetes operator patterns

## Step-by-Step Reconciliation Testing

### Step 1: Deploy Storage Backend with Multiple Buckets

Create MinIO with support for bucket switching:

```bash
kubectl apply -f - <<EOF
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
              mkdir -p /storage/tempo2 && \
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
          emptyDir: {}
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
  name: minio-test
stringData:
  endpoint: http://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**Key Features for Testing**:
- **Dual Buckets**: Both `tempo` and `tempo2` buckets created
- **Initial Configuration**: Secret points to `tempo` bucket initially
- **Update Capability**: Secret can be modified to point to `tempo2` bucket

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Deploy TempoStack for Reconciliation Testing

Create a TempoStack that will be used to test reconciliation:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 200M
EOF
```

**Configuration Elements**:
- **Minimal Specification**: Simple setup to focus on reconciliation behavior
- **Secret Reference**: Links to storage secret that will be updated
- **Compact Size**: 200MB storage for efficient testing

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Test Resource Recreation (Reconciliation)

Validate that deleted resources are automatically recreated:

```bash
# Wait for TempoStack to be ready
kubectl get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# List all services created by the operator
kubectl get services -l app.kubernetes.io/managed-by=tempo-operator

# Delete a critical service to test reconciliation
kubectl delete service tempo-simplest-querier

# Verify the service is automatically recreated
kubectl get service tempo-simplest-querier
# Should show the service exists again
```

**Reconciliation Verification**:
- **Deletion Detection**: Operator detects the service deletion via watch events
- **Automatic Recreation**: Service is recreated with identical configuration
- **State Consistency**: All labels, annotations, and specs match original

### Step 4: Test Configuration Update Reconciliation

Validate that secret changes trigger configuration updates:

```bash
# Update the storage secret to point to a different bucket
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
   name: minio-test
stringData:
  endpoint: http://minio:9000
  bucket: tempo2
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF

# Verify the operator updates the Tempo configuration
kubectl get configmap tempo-simplest -o jsonpath='{.data.tempo\.yaml}' | grep "bucket: tempo2"
# Should show the new bucket configuration
```

**Configuration Update Process**:
1. **Secret Watch**: Operator detects secret changes via Kubernetes watch API
2. **Configuration Regeneration**: New Tempo config generated with updated bucket
3. **Pod Restart**: StatefulSet/Deployment pods restarted to pick up new config
4. **Health Check**: Operator verifies new configuration is working

**Reference**: [`03-update-storage-secret.yaml`](./03-update-storage-secret.yaml)

### Step 5: Test Reconciliation Disabling

Validate that reconciliation can be controlled:

```bash
# Disable reconciliation on the TempoStack
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
  annotations:
    tempo.grafana.com/reconcile: "false"
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 200M
EOF

# Test that configuration is no longer updated automatically
# (Manual testing would involve making changes and verifying they're not applied)
```

**Reconciliation Control Features**:
- **Annotation-Based**: Uses `tempo.grafana.com/reconcile: "false"` annotation
- **Selective Control**: Can disable reconciliation per resource
- **Manual Override**: Allows manual management when needed

**Reference**: [`05-disable-reconciliation.yaml`](./05-disable-reconciliation.yaml)

## Reconciliation Behavior Deep Dive

### 1. **Watch-Based Event Processing**

The operator uses Kubernetes watch APIs to detect changes:

```go
// Simplified operator watch logic
func (r *TempoStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&tempov1alpha1.TempoStack{}).
        Owns(&appsv1.StatefulSet{}).
        Owns(&corev1.Service{}).
        Owns(&corev1.ConfigMap{}).
        Watches(&source.Kind{Type: &corev1.Secret{}}, 
            handler.EnqueueRequestsFromMapFunc(r.secretToTempoStack)).
        Complete(r)
}
```

**Event Sources**:
- **Primary Resource**: TempoStack custom resource changes
- **Owned Resources**: StatefulSets, Services, ConfigMaps managed by operator
- **Referenced Resources**: Secrets, ConfigMaps referenced by TempoStack

### 2. **Reconciliation Loop Logic**

The operator follows standard Kubernetes controller patterns:

```
┌─────────────────┐
│ Event Received  │
└─────────┬───────┘
          │
┌─────────▼───────┐
│ Get Current     │
│ State           │
└─────────┬───────┘
          │
┌─────────▼───────┐    ┌─────────────────┐
│ Compare with    │───▶│ No Changes      │
│ Desired State   │    │ Return          │
└─────────┬───────┘    └─────────────────┘
          │
┌─────────▼───────┐
│ Apply Changes   │
│ - Create        │
│ - Update        │
│ - Delete        │
└─────────┬───────┘
          │
┌─────────▼───────┐
│ Update Status   │
│ Return Result   │
└─────────────────┘
```

### 3. **Resource Ownership and Garbage Collection**

The operator establishes ownership relationships:

```yaml
# Example of owner reference in managed resource
apiVersion: v1
kind: Service
metadata:
  name: tempo-simplest-querier
  ownerReferences:
  - apiVersion: tempo.grafana.com/v1alpha1
    kind: TempoStack
    name: simplest
    uid: 550e8400-e29b-41d4-a716-446655440000
    controller: true
    blockOwnerDeletion: true
```

**Ownership Benefits**:
- **Automatic Cleanup**: Managed resources deleted when parent is deleted
- **Reconciliation Scope**: Only resources owned by the operator are managed
- **Conflict Prevention**: Prevents conflicts with other controllers

## Advanced Reconciliation Features

### 1. **Conditional Reconciliation**

Control reconciliation behavior with annotations:

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: example
  annotations:
    # Disable reconciliation entirely
    tempo.grafana.com/reconcile: "false"
    
    # Disable specific resource types
    tempo.grafana.com/reconcile.services: "false"
    tempo.grafana.com/reconcile.configmaps: "false"
```

### 2. **Reconciliation Frequency Control**

Adjust reconciliation timing:

```yaml
spec:
  # Custom reconciliation interval (if supported)
  reconciliation:
    interval: 30s
    retryBackoff: exponential
    maxRetries: 5
```

### 3. **Status Reporting**

Monitor reconciliation status:

```bash
# Check TempoStack status
kubectl get tempostack simplest -o yaml | yq '.status'

# Key status fields:
# - conditions: Ready, Progressing, Degraded
# - observedGeneration: Last processed resource version
# - componentImages: Currently deployed component versions
```

## Troubleshooting Reconciliation Issues

### 1. **Reconciliation Not Working**

#### Check Operator Logs
```bash
# Find operator pod
OPERATOR_POD=$(kubectl get pods -n tempo-operator-system -l control-plane=controller-manager -o name | head -1)

# Check reconciliation logs
kubectl logs $OPERATOR_POD -n tempo-operator-system | grep -i "reconcil\|error\|failed"

# Look for specific errors
kubectl logs $OPERATOR_POD -n tempo-operator-system | grep "simplest"
```

#### Verify RBAC Permissions
```bash
# Check if operator has necessary permissions
kubectl auth can-i "*" "*" --as=system:serviceaccount:tempo-operator-system:tempo-operator-controller-manager

# Check specific resource permissions
kubectl auth can-i create services --as=system:serviceaccount:tempo-operator-system:tempo-operator-controller-manager
kubectl auth can-i update configmaps --as=system:serviceaccount:tempo-operator-system:tempo-operator-controller-manager
```

#### Check Resource Events
```bash
# Look for events related to the TempoStack
kubectl describe tempostack simplest

# Check events for managed resources
kubectl get events --field-selector involvedObject.name=tempo-simplest-querier
```

### 2. **Configuration Not Updating**

#### Verify Secret Changes
```bash
# Check if secret was actually updated
kubectl get secret minio-test -o yaml

# Verify secret is referenced correctly
kubectl get tempostack simplest -o yaml | yq '.spec.storage.secret'

# Check for secret watch events
kubectl logs $OPERATOR_POD -n tempo-operator-system | grep "secret.*minio-test"
```

#### Validate Configuration Generation
```bash
# Check current ConfigMap content
kubectl get configmap tempo-simplest -o yaml

# Verify configuration syntax
kubectl get configmap tempo-simplest -o jsonpath='{.data.tempo\.yaml}' | yq eval '.'

# Compare with expected configuration
kubectl get configmap tempo-simplest -o jsonpath='{.data.tempo\.yaml}' | grep -E "(bucket|endpoint)"
```

### 3. **Resource Recreation Issues**

#### Check Owner References
```bash
# Verify owner references are set correctly
kubectl get service tempo-simplest-querier -o yaml | yq '.metadata.ownerReferences'

# Validate controller reference
kubectl get service tempo-simplest-querier -o jsonpath='{.metadata.ownerReferences[0].controller}'
# Should return: true
```

#### Monitor Resource Deletion and Recreation
```bash
# Watch for resource changes
kubectl get services -l app.kubernetes.io/managed-by=tempo-operator -w

# Track specific resource lifecycle
kubectl describe service tempo-simplest-querier
```

## Performance and Scalability Considerations

### 1. **Reconciliation Frequency**

Monitor and optimize reconciliation performance:

```bash
# Check reconciliation frequency metrics
kubectl port-forward $OPERATOR_POD 8080:8080 -n tempo-operator-system &
curl http://localhost:8080/metrics | grep controller_runtime_reconcile

# Key metrics:
# - controller_runtime_reconcile_total
# - controller_runtime_reconcile_duration_seconds
# - controller_runtime_reconcile_errors_total
```

### 2. **Resource Watch Efficiency**

Optimize watch resource consumption:

```yaml
# Operator configuration for watch optimization
spec:
  watchNamespace: tempo-system  # Limit watch scope
  leaderElection:
    enabled: true               # Enable leader election for HA
  resources:
    limits:
      memory: 512Mi
      cpu: 500m
```

### 3. **Batch Updates**

Configure operator for efficient batch processing:

```yaml
# Operator tuning parameters
env:
- name: MAX_CONCURRENT_RECONCILES
  value: "5"
- name: RECONCILE_TIMEOUT
  value: "300s"
```

## Production Best Practices

### 1. **Monitoring and Alerting**

Set up comprehensive monitoring:

```yaml
# Prometheus alert for reconciliation failures
alert: TempoOperatorReconcileFailed
expr: increase(controller_runtime_reconcile_errors_total{controller="tempostack"}[5m]) > 0
for: 2m
annotations:
  summary: "Tempo operator reconciliation failures detected"
  description: "TempoStack reconciliation has failed {{ $value }} times in the last 5 minutes"
```

### 2. **Backup and Recovery**

Implement backup strategies for configuration:

```bash
# Backup TempoStack configurations
kubectl get tempostack -o yaml > tempostack-backup.yaml

# Backup related secrets and configmaps
kubectl get secret -l app.kubernetes.io/managed-by=tempo-operator -o yaml > tempo-secrets-backup.yaml
```

### 3. **Change Management**

Implement controlled change processes:

```yaml
# GitOps approach for TempoStack management
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: tempo-stack
spec:
  syncPolicy:
    automated:
      prune: false      # Prevent automatic deletion
      selfHeal: true    # Enable self-healing via reconciliation
```

## Related Concepts

- [Kubernetes Controller Pattern](https://kubernetes.io/docs/concepts/architecture/controller/)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/reconcile
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates core operator functionality and may take several minutes to complete as it tests actual reconciliation timing and behavior.

