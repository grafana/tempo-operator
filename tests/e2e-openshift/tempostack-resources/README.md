# TempoStack Resource Management and Allocation

This configuration blueprint demonstrates comprehensive resource management strategies for TempoStack deployments, including global resource allocation and fine-grained per-component resource configuration. This setup provides production-ready resource planning, performance optimization, and capacity management for distributed Tempo deployments.

## Overview

This test validates advanced resource management features:
- **Global Resource Allocation**: Total resource limits distributed across all components
- **Per-Component Resource Control**: Fine-grained resource allocation for each component
- **Multi-Tenant Resource Management**: Resource allocation with OpenShift multitenancy
- **Dynamic Resource Updates**: Runtime resource configuration changes
- **Performance Optimization**: Component-specific resource tuning

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ Global Resource Pool    │───▶│   TempoStack Resource    │───▶│ Component Allocation    │
│ - Total: 2Gi memory     │    │   Distribution           │    │ - Distributor: 521m/226Mi│
│ - Total: 2000m CPU      │    │ ┌─────────────────────┐  │    │ - Ingester: 721m/1013Mi │
│ - Total: 1Gi storage    │    │ │ Automatic           │  │    │ - Querier: 181m/288Mi   │
└─────────────────────────┘    │ │ Distribution        │  │    │ - Query-Frontend: 161m  │
                               │ └─────────────────────┘  │    │ - Compactor: 301m/349Mi │
┌─────────────────────────┐    │                          │    │ - Gateway: 122m/104Mi   │
│ Per-Component Tuning    │───▶│ Component-Specific       │    └─────────────────────────┘
│ - CPU intensive tasks   │    │ Resource Profiles        │
│ - Memory requirements   │    │ ┌─────────────────────┐  │    ┌─────────────────────────┐
│ - I/O characteristics   │    │ │ Requests vs Limits  │  │───▶│ Kubernetes Resource     │
└─────────────────────────┘    │ │ - Requests: Min     │  │    │ Management              │
                               │ │ - Limits: Max       │  │    │ - Pod scheduling        │
┌─────────────────────────┐    │ └─────────────────────┘  │    │ - QoS classes          │
│ Multi-Tenant Isolation  │───▶│                          │    │ - Resource quotas      │
│ - Gateway resources     │    │ Authentication           │    │ - Node capacity        │
│ - Query isolation       │    │ Components               │    └─────────────────────────┘
│ - Tenant-specific       │    │ - Jaeger UI              │
└─────────────────────────┘    │ - Authentication         │    ┌─────────────────────────┐
                               │ - Tempo Query            │───▶│ Performance Monitoring  │
                               └──────────────────────────┘    │ - Resource utilization  │
                                                               │ - Performance metrics   │
                                                               │ - Scaling decisions     │
                                                               └─────────────────────────┘
```

## Prerequisites

- OpenShift cluster with adequate resources
- Tempo Operator installed
- Understanding of Kubernetes resource management
- Knowledge of distributed tracing performance characteristics
- Familiarity with multi-tenant resource isolation

## Step-by-Step Resource Configuration

### Step 1: Deploy Storage Backend

Create the standard MinIO storage backend:

```bash
oc apply -f - <<EOF
# Standard MinIO deployment for resource testing
# Reference: install-storage.yaml
EOF
```

The storage configuration provides a consistent S3-compatible backend for resource testing scenarios.

**Reference**: [`install-storage.yaml`](./install-storage.yaml)

### Step 2: Deploy TempoStack with Global Resource Allocation

Create TempoStack with global resource limits that are automatically distributed:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind:  TempoStack
metadata:
  name: tmrs
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  tenants:
    mode: openshift
    authentication:
      - tenantName: dev
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
      - tenantName: prod
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
  template:
    gateway:
      enabled: true
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```

**Global Resource Configuration Details**:

#### Total Resource Pool
- `memory: 2Gi`: Total memory allocated across all components
- `cpu: 2000m`: Total CPU (2 cores) distributed across components
- **Automatic Distribution**: Operator calculates optimal per-component allocation

#### Multi-Tenant Configuration
- `tenants.mode: openshift`: OpenShift-native multitenancy
- **Gateway Enabled**: Multi-tenant proxy for secure access
- **Authentication**: Separate dev and prod tenant identities

#### Component Template
- `gateway.enabled: true`: Enables multi-tenant gateway component
- `jaegerQuery.enabled: true`: Provides Jaeger UI interface
- **Default Allocation**: Components receive operator-calculated resource shares

**Resource Distribution Algorithm**:
The operator automatically distributes the total resource pool based on:
- **Component Type**: Different resource profiles for each component
- **Expected Load**: Historical performance characteristics
- **Scaling Requirements**: Horizontal scaling considerations
- **Multi-Tenancy Overhead**: Additional resources for gateway and authentication

**Reference**: [`install-tempostack.yaml`](./install-tempostack.yaml)

### Step 3: Validate Global Resource Allocation

Verify that resources are properly allocated across all components:

```bash
# Check TempoStack readiness
oc get tempostack tmrs -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify all components are deployed
oc get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check resource allocation in each component
oc get deployment tempo-tmrs-distributor -o yaml | grep -A10 resources
oc get statefulset tempo-tmrs-ingester -o yaml | grep -A10 resources
oc get deployment tempo-tmrs-querier -o yaml | grep -A10 resources
oc get deployment tempo-tmrs-query-frontend -o yaml | grep -A10 resources
oc get deployment tempo-tmrs-compactor -o yaml | grep -A10 resources
oc get deployment tempo-tmrs-gateway -o yaml | grep -A10 resources

# Verify total resource allocation
oc describe tempostack tmrs | grep -A20 "Resource Allocation"
```

Expected validation results:
- **All Components Ready**: Each component deployment/statefulset in Ready state
- **Resource Distribution**: Automatic allocation based on component requirements
- **Total Compliance**: Sum of allocated resources <= total resource limits
- **Multi-Tenant Gateway**: Additional gateway component with appropriate resources

### Step 4: Update to Per-Component Resource Allocation

Apply detailed per-component resource configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: tmrs
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  tenants:
    mode: openshift
    authentication:
      - tenantName: dev
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
      - tenantName: prod
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
  template:
    compactor:
      resources:
        limits:
          cpu: 301m
          memory: 349Mi
        requests:
          cpu: 91m
          memory: 105Mi
    distributor:
      component:
        resources:
          limits:
            cpu: 521m
            memory: 226Mi
          requests:
            cpu: 157m
            memory: 69Mi
    gateway:
      enabled: true
      component:
        resources:
          limits:
            cpu: 122m
            memory: 104Mi
          requests:
            cpu: 37m
            memory: 32Mi
    ingester:
      resources:
        limits:
          cpu: 721m
          memory: 1013Mi
        requests:
          cpu: 217m
          memory: 302Mi
    querier:
      resources:
        limits:
          cpu: 181m
          memory: 288Mi
        requests:
          cpu: 55m
          memory: 87Mi
    queryFrontend:
      component:
        resources:
          limits:
            cpu: 161m
            memory: 83Mi
          requests:
            cpu: 49m
            memory: 27Mi
      jaegerQuery:
        authentication:
          resources:
            limits:
              cpu: 161m
              memory: 83Mi
            requests:
              cpu: 49m
              memory: 29Mi
        tempoQuery:
          resources:
            limits:
              cpu: 161m
              memory: 83Mi
            requests:
              cpu: 49m
              memory: 29Mi 
        enabled: true
        resources:
          limits:
            cpu: 167m
            memory: 86Mi
          requests:
            cpu: 49m
            memory: 29Mi
EOF
```

**Per-Component Resource Configuration Details**:

#### Compactor Resources
```yaml
compactor:
  resources:
    limits: {cpu: 301m, memory: 349Mi}
    requests: {cpu: 91m, memory: 105Mi}
```
- **CPU-Intensive**: Block compaction and optimization
- **Memory Moderate**: Block metadata and compaction buffers
- **Background Process**: Lower priority requests

#### Distributor Resources
```yaml
distributor:
  component:
    resources:
      limits: {cpu: 521m, memory: 226Mi}
      requests: {cpu: 157m, memory: 69Mi}
```
- **High CPU**: Trace parsing and distribution logic
- **Moderate Memory**: Trace buffering and routing tables
- **High Priority**: Critical ingestion path

#### Gateway Resources
```yaml
gateway:
  component:
    resources:
      limits: {cpu: 122m, memory: 104Mi}
      requests: {cpu: 37m, memory: 32Mi}
```
- **Authentication Overhead**: Token validation and tenant routing
- **Network I/O**: Proxy traffic between tenants and backend
- **Security Processing**: Authorization and audit logging

#### Ingester Resources
```yaml
ingester:
  resources:
    limits: {cpu: 721m, memory: 1013Mi}
    requests: {cpu: 217m, memory: 302Mi}
```
- **Highest Memory**: Trace block storage and WAL management
- **High CPU**: Trace serialization and storage operations
- **Critical Component**: Data persistence and retrieval

#### Querier Resources
```yaml
querier:
  resources:
    limits: {cpu: 181m, memory: 288Mi}
    requests: {cpu: 55m, memory: 87Mi}
```
- **Query Processing**: Block scanning and trace reconstruction
- **Moderate Memory**: Query result buffers and caching
- **Variable Load**: Depends on query complexity

#### Query Frontend Resources
```yaml
queryFrontend:
  component:
    resources:
      limits: {cpu: 161m, memory: 83Mi}
      requests: {cpu: 49m, memory: 27Mi}
```
- **Query Coordination**: Request routing and aggregation
- **UI Hosting**: Jaeger UI serving and API endpoints
- **Caching Layer**: Query result caching and optimization

#### Multi-Component Query Frontend
```yaml
jaegerQuery:
  authentication:
    resources: {limits: {cpu: 161m, memory: 83Mi}}
  tempoQuery:
    resources: {limits: {cpu: 161m, memory: 83Mi}}
  resources: {limits: {cpu: 167m, memory: 86Mi}}
```
- **Authentication Component**: RBAC and tenant authentication
- **Tempo Query**: Backend query processor
- **Jaeger UI**: Frontend interface component

**Reference**: [`update-tempostack.yaml`](./update-tempostack.yaml)

### Step 5: Validate Per-Component Resource Allocation

Verify that the detailed resource configuration is properly applied:

```bash
# Check TempoStack readiness after update
oc get tempostack tmrs -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify specific resource allocations
echo "=== Compactor Resources ==="
oc get deployment tempo-tmrs-compactor -o jsonpath='{.spec.template.spec.containers[0].resources}'

echo "=== Distributor Resources ==="
oc get deployment tempo-tmrs-distributor -o jsonpath='{.spec.template.spec.containers[0].resources}'

echo "=== Ingester Resources ==="
oc get statefulset tempo-tmrs-ingester -o jsonpath='{.spec.template.spec.containers[0].resources}'

echo "=== Querier Resources ==="
oc get deployment tempo-tmrs-querier -o jsonpath='{.spec.template.spec.containers[0].resources}'

echo "=== Query Frontend Resources ==="
oc get deployment tempo-tmrs-query-frontend -o jsonpath='{.spec.template.spec.containers[*].resources}'

echo "=== Gateway Resources ==="
oc get deployment tempo-tmrs-gateway -o jsonpath='{.spec.template.spec.containers[0].resources}'

# Validate total resource consumption
TOTAL_CPU_LIMITS=$(oc get deployment,statefulset -l app.kubernetes.io/managed-by=tempo-operator -o jsonpath='{range .items[*]}{.spec.template.spec.containers[*].resources.limits.cpu}{"\n"}{end}' | grep -v '^$' | sed 's/m//' | awk '{sum += $1} END {print sum "m"}')
echo "Total CPU Limits: $TOTAL_CPU_LIMITS"

TOTAL_MEMORY_LIMITS=$(oc get deployment,statefulset -l app.kubernetes.io/managed-by=tempo-operator -o jsonpath='{range .items[*]}{.spec.template.spec.containers[*].resources.limits.memory}{"\n"}{end}' | grep -v '^$')
echo "Memory Limits per component: $TOTAL_MEMORY_LIMITS"
```

**Resource Validation Checks**:
- **Individual Limits**: Each component has specific CPU and memory limits
- **Request vs Limits**: Appropriate request/limit ratios for QoS
- **Total Compliance**: Total allocated resources within global limits
- **Multi-Container**: Complex components with multiple container resources

## Resource Management Strategies

### 1. **Global vs Per-Component Resource Allocation**

#### Global Resource Strategy
```yaml
# Simple global allocation
spec:
  resources:
    total:
      limits:
        memory: 4Gi
        cpu: 4000m
      requests:
        memory: 2Gi
        cpu: 2000m
```

**When to Use Global**:
- **Rapid Prototyping**: Quick deployment without detailed analysis
- **Uniform Workloads**: Consistent trace volumes and patterns
- **Operator-Managed**: Trust operator's resource distribution algorithm
- **Simplified Management**: Easier resource planning and scaling

#### Per-Component Resource Strategy
```yaml
# Detailed per-component allocation
spec:
  template:
    ingester:
      resources:
        limits: {memory: 2Gi, cpu: 1000m}
        requests: {memory: 1Gi, cpu: 500m}
    distributor:
      component:
        resources:
          limits: {memory: 1Gi, cpu: 1500m}
          requests: {memory: 512Mi, cpu: 750m}
```

**When to Use Per-Component**:
- **Production Workloads**: Precise resource control required
- **Variable Loads**: Different components have different utilization patterns
- **Cost Optimization**: Minimize resource waste through targeted allocation
- **Performance Tuning**: Component-specific optimization requirements

### 2. **Quality of Service (QoS) Classes**

#### Guaranteed QoS
```yaml
# Requests = Limits for guaranteed resources
resources:
  requests: {cpu: 500m, memory: 1Gi}
  limits: {cpu: 500m, memory: 1Gi}
```
- **Use Cases**: Critical components (ingester, distributor)
- **Benefits**: Guaranteed resource availability
- **Drawbacks**: Higher resource cost

#### Burstable QoS
```yaml
# Requests < Limits for burstable performance
resources:
  requests: {cpu: 100m, memory: 256Mi}
  limits: {cpu: 500m, memory: 1Gi}
```
- **Use Cases**: Variable load components (querier, query frontend)
- **Benefits**: Efficient resource utilization
- **Considerations**: Potential resource contention

#### Best Effort QoS
```yaml
# No requests specified
resources:
  limits: {cpu: 200m, memory: 512Mi}
```
- **Use Cases**: Non-critical background processes (compactor)
- **Benefits**: Lowest resource cost
- **Risks**: No guaranteed resources

### 3. **Horizontal Pod Autoscaling Integration**

#### Component-Specific HPA
```yaml
# HPA for distributor based on CPU
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tempo-distributor-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tempo-tmrs-distributor
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

#### Custom Metrics HPA
```yaml
# HPA based on trace ingestion rate
metrics:
- type: Pods
  pods:
    metric:
      name: tempo_distributor_spans_received_total
    target:
      type: AverageValue
      averageValue: "1000"
```

## Production Resource Planning

### 1. **Capacity Planning Guidelines**

#### Component Resource Ratios
```yaml
# Typical production ratios
distributor:  20% CPU, 15% Memory  # High CPU for processing
ingester:     35% CPU, 50% Memory  # High memory for storage
querier:      20% CPU, 20% Memory  # Balanced for queries
query-frontend: 15% CPU, 10% Memory  # UI and coordination
compactor:    10% CPU, 5% Memory   # Background processing
```

#### Workload-Based Sizing
```yaml
# High ingestion workload
distributor:
  replicas: 5
  resources:
    limits: {cpu: 1000m, memory: 512Mi}
ingester:
  replicas: 5
  resources:
    limits: {cpu: 1500m, memory: 4Gi}

# High query workload  
querier:
  replicas: 8
  resources:
    limits: {cpu: 800m, memory: 1Gi}
query-frontend:
  replicas: 3
  resources:
    limits: {cpu: 500m, memory: 512Mi}
```

### 2. **Multi-Tenant Resource Isolation**

#### Tenant-Specific Resource Limits
```yaml
# Resource quotas per tenant namespace
apiVersion: v1
kind: ResourceQuota
metadata:
  name: dev-tenant-quota
  namespace: dev-traces
spec:
  hard:
    requests.cpu: "2"
    requests.memory: 4Gi
    limits.cpu: "4"
    limits.memory: 8Gi
    persistentvolumeclaims: "10"
```

#### Gateway Resource Scaling
```yaml
# Scale gateway based on tenant count
template:
  gateway:
    component:
      resources:
        # Base resources + (tenant_count * overhead)
        limits:
          cpu: 200m + (tenant_count * 50m)
          memory: 256Mi + (tenant_count * 64Mi)
```

### 3. **Performance Monitoring and Optimization**

#### Resource Utilization Metrics
```bash
# Monitor component resource usage
oc top pods -l app.kubernetes.io/managed-by=tempo-operator

# Check for resource constraints
oc get events --field-selector reason=FailedScheduling
oc get events --field-selector reason=OOMKilled

# Analyze resource efficiency
oc port-forward deployment/tempo-tmrs-distributor 3200:3200 &
curl http://localhost:3200/metrics | grep -E "(cpu|memory|resource)"
```

#### Alerting for Resource Issues
```yaml
# Prometheus alerts for resource problems
alert: TempoComponentHighMemoryUsage
expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9
for: 10m
labels:
  component: "{{ $labels.container }}"
annotations:
  summary: "Tempo component {{ $labels.container }} memory usage is high"

alert: TempoComponentCPUThrottling
expr: rate(container_cpu_cfs_throttled_seconds_total[5m]) > 0.1
for: 5m
annotations:
  summary: "Tempo component CPU is being throttled"
```

## Advanced Resource Configuration

### 1. **Node Affinity and Resource Allocation**

#### CPU-Intensive Components on High-Performance Nodes
```yaml
template:
  distributor:
    nodeSelector:
      node-type: compute-optimized
    component:
      resources:
        limits: {cpu: 2000m, memory: 1Gi}

  compactor:
    nodeSelector:
      node-type: compute-optimized
    resources:
      limits: {cpu: 1500m, memory: 2Gi}
```

#### Memory-Intensive Components on Memory-Optimized Nodes
```yaml
template:
  ingester:
    nodeSelector:
      node-type: memory-optimized
    resources:
      limits: {cpu: 1000m, memory: 8Gi}
    tolerations:
    - key: memory-intensive
      operator: Equal
      value: "true"
      effect: NoSchedule
```

### 2. **Storage Resource Management**

#### Per-Component Storage Requirements
```yaml
template:
  ingester:
    volumeClaimTemplate:
      spec:
        accessModes: [ReadWriteOnce]
        resources:
          requests:
            storage: 100Gi
        storageClassName: fast-ssd

  compactor:
    extraVolumes:
    - name: tmp-storage
      spec:
        accessModes: [ReadWriteOnce]
        resources:
          requests:
            storage: 50Gi
        storageClassName: standard-ssd
```

#### Storage Performance Optimization
```yaml
# NVMe storage for high-performance workloads
spec:
  storageSize: 500Gi
  storageClassName: nvme-ssd
  
  template:
    ingester:
      # Additional ephemeral storage for WAL
      resources:
        limits:
          ephemeral-storage: 20Gi
```

### 3. **Resource Limits and Security**

#### Pod Security Context with Resource Limits
```yaml
template:
  ingester:
    podSecurityContext:
      runAsNonRoot: true
      runAsUser: 10001
      fsGroup: 10001
      seccompProfile:
        type: RuntimeDefault
    resources:
      limits:
        cpu: 2000m
        memory: 4Gi
        ephemeral-storage: 10Gi
```

#### Network Resource Management
```yaml
# Network policies with resource considerations
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-bandwidth-limit
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: tempo-operator
  policyTypes:
  - Ingress
  - Egress
  annotations:
    kubernetes.io/ingress-bandwidth: 100M
    kubernetes.io/egress-bandwidth: 100M
```

## Troubleshooting Resource Issues

### 1. **Resource Allocation Problems**

#### Over-Allocation Detection
```bash
# Check if total requested resources exceed global limits
kubectl get tempostack tmrs -o yaml | yq '.spec.resources.total'

# Calculate actual component resource requests
TOTAL_REQUESTS=$(kubectl get deployment,statefulset -l app.kubernetes.io/managed-by=tempo-operator -o jsonpath='{range .items[*]}{.spec.template.spec.containers[*].resources.requests.cpu}{"\n"}{end}' | sed 's/m//' | awk '{sum += $1} END {print sum}')
echo "Total CPU requests: ${TOTAL_REQUESTS}m"

# Check for validation errors
kubectl get events --field-selector reason=FailedValidation
```

#### Under-Allocation Issues
```bash
# Monitor for OOM kills
kubectl get events --field-selector reason=OOMKilled

# Check for CPU throttling
kubectl top pods -l app.kubernetes.io/managed-by=tempo-operator

# Analyze resource utilization trends
kubectl port-forward deployment/tempo-tmrs-ingester 3200:3200 &
curl http://localhost:3200/metrics | grep go_memstats
```

### 2. **Performance Degradation**

#### Resource Bottleneck Analysis
```bash
# Check node resource availability
kubectl describe nodes | grep -A5 "Allocated resources"

# Monitor component-specific metrics
kubectl port-forward deployment/tempo-tmrs-distributor 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_distributor_spans_received_total

# Check for resource contention
kubectl top nodes
kubectl get events --field-selector reason=NodeNotReady
```

#### Quality of Service Issues
```bash
# Check QoS class assignments
kubectl get pods -l app.kubernetes.io/managed-by=tempo-operator -o custom-columns=NAME:.metadata.name,QOS:.status.qosClass

# Monitor for evictions
kubectl get events --field-selector reason=Evicted

# Check resource guarantees
kubectl describe pod tempo-tmrs-ingester-0 | grep -A10 "Limits:\|Requests:"
```

### 3. **Scaling and Capacity Issues**

#### Horizontal Scaling Validation
```bash
# Check if components can scale horizontally
kubectl get hpa -l app.kubernetes.io/managed-by=tempo-operator

# Verify resource availability for scaling
kubectl describe nodes | grep "cpu.*memory"

# Test manual scaling
kubectl scale deployment tempo-tmrs-querier --replicas=3
kubectl get pods -l app.kubernetes.io/component=querier
```

## Related Configurations

- [Basic TempoStack](../../e2e/compatibility/README.md) - Standard resource allocation
- [Performance Tuning](../monitoring/README.md) - Monitoring resource utilization
- [Multi-Tenant Resources](../multitenancy/README.md) - Tenant-specific resource management

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/tempostack-resources
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test demonstrates both global and per-component resource allocation strategies, validating dynamic resource updates and multi-tenant resource management. The test includes comprehensive resource validation and performance optimization techniques for production deployments.

