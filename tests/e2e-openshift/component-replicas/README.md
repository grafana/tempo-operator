# TempoStack Component Scaling and Replica Management

This configuration blueprint demonstrates how to dynamically scale TempoStack components for high availability and load distribution. This setup showcases production-ready scaling patterns with multi-tenant configuration, gateway routing, and validation of scaling operations without service disruption.

## Overview

This test validates a scalable observability stack featuring:
- **Dynamic Component Scaling**: Runtime scaling of all TempoStack components from 1 to 2 replicas
- **Multi-Tenant Architecture**: OpenShift RBAC-based tenant isolation with gateway routing
- **High Availability**: Load distribution across multiple component replicas
- **Zero-Downtime Scaling**: Scaling operations without service interruption
- **Load Balancing**: Automatic traffic distribution across scaled components

## Architecture

### Initial Configuration (1 Replica)
```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OTel Collector  │───▶│   Gateway (×1)       │───▶│   Storage       │
└─────────────────┘    └──────────────────────┘    │   (MinIO)       │
                                │                  └─────────────────┘
                                ▼
                       ┌──────────────────────┐
                       │ Distributor (×1)     │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Ingester (×1)        │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Querier (×1)         │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Query Frontend (×1)  │
                       └──────────────────────┘
```

### Scaled Configuration (2 Replicas)
```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OTel Collector  │───▶│   Gateway (×2)       │───▶│   Storage       │
└─────────────────┘    └──────────────────────┘    │   (MinIO)       │
                                │                  └─────────────────┘
                                ▼
                       ┌──────────────────────┐
                       │ Distributor (×2)     │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Ingester (×2)        │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Querier (×2)         │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │ Query Frontend (×2)  │
                       └──────────────────────┘
```

## Prerequisites

- OpenShift cluster with sufficient resources for scaled components
- Tempo Operator installed
- OpenTelemetry Operator installed
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Deploy MinIO Object Storage

Create the storage backend for trace persistence:

```bash
oc apply -f install-storage.yaml
```

**Reference**: [`install-storage.yaml`](./install-storage.yaml)

### Step 2: Deploy Initial TempoStack (Single Replica)

Create TempoStack with multi-tenant configuration and single replicas:

```bash
oc apply -f install-tempo.yaml
```

**Key Configuration from [`install-tempo.yaml`](./install-tempo.yaml)**:
- **Multi-Tenant Setup**: OpenShift RBAC mode with `dev` and `prod` tenants
- **Gateway Enabled**: Central authentication and routing gateway
- **Jaeger Query UI**: Enabled with external route access
- **RBAC Configuration**: ClusterRole and RoleBinding for tenant access

### Step 3: Scale Components to 2 Replicas

Apply the scaling configuration with parameterized replica counts:

```bash
# The test uses bindings to set tempo_replicas=2
oc apply -f scale-tempo.yaml
```

**Key Scaling Configuration from [`scale-tempo.yaml`](./scale-tempo.yaml)**:
- **Parameterized Replicas**: Uses `($tempo_replicas)` binding for all components
- **All Components Scaled**: Compactor, distributor, gateway, ingester, querier, query frontend
- **Consistent Scaling**: All components scaled uniformly to maintain balance

Components being scaled:
- `compactor.replicas: ($tempo_replicas)`
- `distributor.component.replicas: ($tempo_replicas)`
- `gateway.component.replicas: ($tempo_replicas)`
- `ingester.replicas: ($tempo_replicas)`
- `querier.replicas: ($tempo_replicas)`
- `queryFrontend.component.replicas: ($tempo_replicas)`

### Step 4: Deploy OpenTelemetry Collector

Install collector to generate traces for testing scaled components:

```bash
oc apply -f install-otelcol.yaml
```

**Reference**: [`install-otelcol.yaml`](./install-otelcol.yaml)

### Step 5: Generate Test Traces

Create traces to validate functionality with scaled components:

```bash
oc apply -f generate-traces.yaml
```

**Reference**: [`generate-traces.yaml`](./generate-traces.yaml)

### Step 6: Verify Traces with Scaled Components

Validate that traces are properly handled by multiple component replicas:

```bash
oc apply -f verify-traces.yaml
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

### Step 7: Scale Down to Single Replica

Test scaling down to verify graceful replica reduction:

```bash
# The test reapplies scale-tempo.yaml with tempo_replicas=1
oc apply -f scale-tempo.yaml
```

This demonstrates:
- **Graceful Scale-Down**: Reducing replicas without data loss
- **Service Continuity**: Maintaining service availability during scaling
- **Resource Optimization**: Reducing resource usage when load decreases

## Key Features Demonstrated

### 1. **Dynamic Component Scaling**
- **Runtime Scaling**: Scale components up and down without service restart
- **Uniform Scaling**: All components scaled together to maintain proper ratios
- **Parameterized Configuration**: Template-based replica configuration
- **Immediate Effect**: Scaling takes effect without manual intervention

### 2. **High Availability Architecture**
- **Load Distribution**: Traffic automatically distributed across replicas
- **Fault Tolerance**: Service continues if individual replicas fail
- **Rolling Updates**: Scaling operations use rolling update strategy
- **Service Discovery**: Kubernetes services automatically load balance

### 3. **Multi-Tenant Scaling**
- **Tenant Isolation**: Scaling maintains tenant separation and security
- **Gateway Scaling**: Authentication gateway scales with traffic demands
- **RBAC Preservation**: Tenant permissions maintained during scaling
- **Performance Isolation**: Scaled components handle tenant traffic independently

### 4. **Operational Excellence**
- **Zero-Downtime Scaling**: No service interruption during scaling operations
- **Resource Efficiency**: Optimal resource utilization with appropriate scaling
- **Monitoring Integration**: Scaled components automatically discovered
- **Configuration Management**: Consistent configuration across all replicas

## Scaling Patterns and Considerations

### Component Scaling Strategies

1. **Uniform Scaling** (demonstrated):
   - All components scaled by same factor
   - Maintains architectural balance
   - Simplifies configuration management

2. **Selective Scaling** (alternative):
   ```yaml
   template:
     distributor:
       component:
         replicas: 3  # High ingestion load
     querier:
       replicas: 2    # Moderate query load
     ingester:
       replicas: 4    # Data persistence intensive
   ```

3. **Load-Based Scaling** (production):
   - Use Horizontal Pod Autoscaler (HPA)
   - Scale based on CPU/memory metrics
   - Custom metrics scaling (trace volume, query latency)

### Resource Planning

Consider resource requirements for scaled components:

```yaml
spec:
  resources:
    total:
      limits:
        memory: 8Gi    # Increased for multiple replicas
        cpu: 4000m     # Distributed across components
  template:
    distributor:
      component:
        replicas: 2
        resources:
          requests:
            memory: 1Gi
            cpu: 500m
```

## Monitoring Scaled Components

### Verify Scaling Operations

```bash
# Check replica counts
oc get deployments -l app.kubernetes.io/managed-by=tempo-operator

# Verify all components are running
oc get pods -l app.kubernetes.io/managed-by=tempo-operator

# Check service endpoints
oc get endpoints tempo-cmpreps-distributor
```

### Monitor Load Distribution

```bash
# Check ingester ring status
oc port-forward svc/tempo-cmpreps-ingester 3200:3200
curl http://localhost:3200/ring

# Monitor distributor load balancing
oc logs -l app.kubernetes.io/component=distributor | grep -i "request\|connection"

# Verify gateway traffic distribution
oc logs -l app.kubernetes.io/component=gateway | grep -i "upstream\|backend"
```

### Performance Validation

```bash
# Test query performance with multiple queriers
time curl "http://tempo-cmpreps-query-frontend:3200/api/search?q={}"

# Monitor resource utilization
oc top pods -l app.kubernetes.io/managed-by=tempo-operator

# Check component health
oc get tempostack cmpreps -o jsonpath='{.status.components}'
```

## Troubleshooting Scaling Issues

### Common Scaling Problems

1. **Insufficient Resources**:
   ```bash
   # Check resource constraints
   oc describe nodes | grep -A 5 "Allocated resources"
   # Verify pod resource requests
   oc describe pod tempo-cmpreps-distributor-* | grep -A 10 "Requests"
   ```

2. **Service Discovery Issues**:
   ```bash
   # Verify service configuration
   oc describe svc tempo-cmpreps-distributor
   # Check endpoint updates
   oc describe endpoints tempo-cmpreps-distributor
   ```

3. **Configuration Drift**:
   ```bash
   # Validate TempoStack configuration
   oc get tempostack cmpreps -o yaml
   # Check deployment consistency
   oc get deployments -l app.kubernetes.io/managed-by=tempo-operator -o yaml
   ```

### Scaling Validation

```bash
# Verify replica counts match specification
EXPECTED_REPLICAS=2
for component in distributor ingester querier query-frontend gateway compactor; do
  ACTUAL=$(oc get deployment tempo-cmpreps-$component -o jsonpath='{.spec.replicas}')
  echo "$component: expected=$EXPECTED_REPLICAS, actual=$ACTUAL"
done
```

## Production Considerations

### 1. **Resource Planning**
- Calculate total resource requirements for scaled components
- Consider node capacity and resource allocation
- Plan for auto-scaling policies and limits
- Monitor resource utilization patterns

### 2. **Storage Scaling**
- Ingester scaling affects storage requirements
- Consider storage performance with multiple ingesters
- Plan for storage class and volume scaling
- Monitor storage I/O patterns

### 3. **Network and Load Balancing**
- Ensure adequate network capacity for scaled traffic
- Configure appropriate load balancing policies
- Monitor network latency and throughput
- Consider service mesh integration

### 4. **Operational Procedures**
- Document scaling procedures and runbooks
- Implement automated scaling based on metrics
- Establish monitoring and alerting for scaled components
- Plan for disaster recovery with scaled architecture

## Related Configurations

- [Multi-tenancy](../multitenancy/README.md) - Multi-tenant architecture patterns
- [Monitoring](../monitoring/README.md) - Monitoring scaled components
- [Basic TempoStack](../../e2e/compatibility/README.md) - Single replica baseline
- [Resource Management](../tempostack-resources/README.md) - Advanced resource configuration

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/component-replicas --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test demonstrates both scale-up and scale-down operations to validate complete scaling lifecycle without service disruption.