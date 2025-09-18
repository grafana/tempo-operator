# TempoStack Integration with OpenShift Service Mesh (OSSM)

This configuration blueprint demonstrates how to integrate TempoStack with OpenShift Service Mesh for comprehensive distributed tracing in microservices architectures. This setup showcases production-ready observability with automatic trace collection, service mesh integration, and unified trace visualization through Kiali.

## Overview

This test validates a complete service mesh observability stack featuring:
- **OpenShift Service Mesh (OSSM)**: Istio-based service mesh with automatic trace generation
- **TempoStack Integration**: Distributed trace collection and storage via Zipkin protocol
- **Kiali Integration**: Service topology visualization with trace correlation
- **Bookinfo Demo Application**: Complete microservices application with distributed tracing
- **Automatic Instrumentation**: Zero-code tracing via service mesh sidecar injection

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ Bookinfo App    │───▶│ OpenShift Service    │───▶│   TempoStack    │
│ ┌─────────────┐ │    │ Mesh (OSSM)          │    │ ┌─────────────┐ │
│ │ productpage │ │    │ ┌─────────────────┐  │    │ │ Distributors│ │
│ │ details     │ │    │ │ Istio Proxies   │  │    │ │ Ingesters   │ │
│ │ reviews     │ │    │ │ Trace Gen       │  │    │ │ Queriers    │ │
│ │ ratings     │ │    │ │ (Zipkin)        │  │    │ └─────────────┘ │
│ └─────────────┘ │    │ └─────────────────┘  │    └─────────────────┘
└─────────────────┘    └──────────────────────┘              │
                                │                             ▼
┌─────────────────┐             ▼                    ┌─────────────────┐
│ External Users  │    ┌──────────────────────┐       │ MinIO Storage   │
│ HTTP Requests   │───▶│ Istio Gateway        │       │ (S3 Compatible) │
└─────────────────┘    │ (Traffic Entry)      │       └─────────────────┘
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │      Kiali UI        │
                       │ - Service Topology   │
                       │ - Trace Correlation  │
                       │ - Performance Metrics│
                       └──────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+)
- OpenShift Service Mesh Operator installed
- Tempo Operator installed
- Kiali Operator installed
- Sufficient cluster resources for service mesh and tracing
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Install OpenShift Service Mesh

Deploy the service mesh control plane with TempoStack integration:

```bash
oc apply -f install-ossm.yaml
```

**Key OSSM Configuration from [`install-ossm.yaml`](./install-ossm.yaml)**:

#### Service Mesh Control Plane
- **Version**: Maistra v2.5 (Istio-based service mesh)
- **Tracing Configuration**: Disabled built-in Jaeger, configured for external Tempo
- **Sampling Rate**: 10000 (100% sampling for comprehensive tracing)

#### Extension Provider Configuration
```yaml
extensionProviders:
  - name: tempo
    zipkin:
      service: tempo-simplest-distributor.tracing-system.svc.cluster.local
      port: 9411
```

#### Namespace Membership
- `istio-system`: Service mesh control plane
- `tracing-system`: TempoStack deployment namespace
- `bookinfo`: Demo application namespace

### Step 2: Deploy MinIO Object Storage

Create storage backend for TempoStack:

```bash
oc apply -f install-minio.yaml
```

**Reference**: [`install-minio.yaml`](./install-minio.yaml)

### Step 3: Deploy TempoStack

Install TempoStack for trace collection and storage:

```bash
oc apply -f install-tempo.yaml
```

**Key TempoStack Configuration from [`install-tempo.yaml`](./install-tempo.yaml)**:
- **Namespace**: `tracing-system` (included in service mesh)
- **Jaeger Query UI**: Enabled with OpenShift route for external access
- **Storage**: MinIO S3-compatible backend
- **Resource Allocation**: 3Gi memory for service mesh trace volumes

### Step 4: Configure Kiali for TempoStack Integration

Update Kiali configuration to integrate with TempoStack:

```bash
oc patch -f update-kiali.yaml
```

**Kiali Integration from [`update-kiali.yaml`](./update-kiali.yaml)**:
- **Tracing Enabled**: Connects Kiali to TempoStack query frontend
- **In-Cluster URL**: Direct service-to-service communication
- **Query Timeout**: 30 seconds for trace retrieval
- **Jaeger Compatible**: Uses Jaeger API compatibility in TempoStack

### Step 5: Enable Service Mesh Tracing

Apply telemetry configuration to enable distributed tracing:

```bash
oc apply -f apply-telemetry-cr.yaml
```

**Telemetry Configuration from [`apply-telemetry-cr.yaml`](./apply-telemetry-cr.yaml)**:
- **Mesh-wide Tracing**: Applied to all services in the mesh
- **Provider**: Uses TempoStack as trace backend
- **Sampling**: 100% sampling for complete trace visibility

### Step 6: Deploy Bookinfo Demo Application

Install the microservices demo application:

```bash
oc apply -f install-bookinfo.yaml
```

**Bookinfo Application Components**:
- **productpage**: Frontend service (Python)
- **details**: Book details service (Ruby)
- **reviews**: Book reviews service (Java) - 3 versions
- **ratings**: Star ratings service (Node.js)

**Reference**: [`install-bookinfo.yaml`](./install-bookinfo.yaml)

### Step 7: Generate Traffic and Traces

Create realistic traffic patterns to generate distributed traces:

```bash
# Generate load through Istio gateway
for i in {1..20}; do
  curl http://$(oc -n istio-system get route istio-ingressgateway -o jsonpath='{.spec.host}')/productpage
  sleep 1
done
```

This generates traces showing:
- Request flow through multiple microservices
- Service dependencies and call patterns
- Latency distribution across service boundaries
- Error propagation and fault isolation

### Step 8: Verify Traces in Kiali

Validate trace collection and visualization:

```bash
oc apply -f verify-traces.yaml
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

## Key Features Demonstrated

### 1. **Service Mesh Automatic Instrumentation**
- **Zero-Code Tracing**: Automatic trace generation via Envoy sidecars
- **Distributed Context Propagation**: Headers automatically injected and propagated
- **Service Communication Visibility**: Complete inter-service call tracking
- **Performance Monitoring**: Latency and error rate measurement

### 2. **TempoStack Service Mesh Integration**
- **Zipkin Protocol Support**: Compatible with Istio's default trace format
- **High-Volume Ingestion**: Handles service mesh trace volumes efficiently
- **Distributed Storage**: Scalable trace storage for microservices architectures
- **Query Performance**: Fast trace retrieval for service mesh topologies

### 3. **Kiali Observability Integration**
- **Service Topology Visualization**: Real-time service dependency graphs
- **Trace Correlation**: Direct links from service map to distributed traces
- **Performance Analytics**: Service-level metrics correlated with traces
- **Unified Experience**: Single pane of glass for service mesh observability

### 4. **Production Service Mesh Patterns**
- **Traffic Management**: Istio gateway for ingress traffic control
- **Security Policies**: mTLS between services with trace visibility
- **Canary Deployments**: Multiple service versions with trace differentiation
- **Fault Injection**: Error scenarios with complete trace visibility

## Service Mesh Tracing Flow

### Automatic Trace Generation

1. **Request Ingress**: Traffic enters via Istio Gateway with trace headers
2. **Sidecar Interception**: Envoy proxies automatically generate trace spans
3. **Context Propagation**: Trace context headers forwarded between services
4. **Span Collection**: Individual service spans collected by Istio infrastructure
5. **Zipkin Export**: Spans exported to TempoStack via Zipkin protocol
6. **Trace Assembly**: TempoStack assembles complete distributed traces
7. **Kiali Integration**: Traces made available for service topology correlation

### Trace Enrichment

Service mesh automatically adds:
- **Service Metadata**: Service names, versions, and namespaces
- **HTTP Semantics**: Request methods, status codes, and URLs
- **Network Metrics**: Connection timing and data transfer rates
- **Security Context**: mTLS certificate information and identity

## Monitoring and Troubleshooting

### Verify Service Mesh Configuration

```bash
# Check service mesh control plane status
oc get servicemeshcontrolplane -n istio-system

# Verify member roll configuration
oc get servicemeshmemberroll -n istio-system

# Check sidecar injection
oc get pods -n bookinfo -o jsonpath='{.items[*].spec.containers[*].name}'
```

### Validate TempoStack Integration

```bash
# Check TempoStack readiness
oc get tempostack simplest -n tracing-system

# Verify Zipkin endpoint accessibility
oc port-forward -n tracing-system svc/tempo-simplest-distributor 9411:9411
curl http://localhost:9411/api/v2/services

# Monitor trace ingestion
oc logs -n tracing-system -l app.kubernetes.io/component=distributor | grep zipkin
```

### Kiali Tracing Integration

```bash
# Check Kiali configuration
oc get kiali kiali -n istio-system -o yaml

# Verify trace query connectivity
oc port-forward -n istio-system svc/kiali 20001:20001
# Access http://localhost:20001/kiali and check tracing integration

# Test trace queries
curl "http://tempo-simplest-query-frontend.tracing-system.svc.cluster.local:16686/api/traces?service=productpage"
```

### Debug Common Issues

1. **Missing Traces**:
   ```bash
   # Check telemetry configuration
   oc get telemetry mesh-default -n istio-system -o yaml
   
   # Verify extension provider
   oc get servicemeshcontrolplane istio-system -n istio-system -o yaml | grep -A 10 extensionProviders
   ```

2. **Sidecar Injection Problems**:
   ```bash
   # Verify namespace membership
   oc get servicemeshmemberroll default -n istio-system -o yaml
   
   # Check pod annotations
   oc describe pod -n bookinfo | grep sidecar
   ```

3. **Kiali Trace Visualization Issues**:
   ```bash
   # Check Kiali external services configuration
   oc get kiali kiali -n istio-system -o jsonpath='{.spec.external_services.tracing}'
   
   # Verify route accessibility
   oc get route tempo-simplest-query-frontend -n tracing-system
   ```

## Production Considerations

### 1. **Performance and Scaling**
- Monitor trace volume impact on service mesh performance
- Configure appropriate sampling rates for production traffic
- Scale TempoStack components based on service mesh trace load
- Optimize Zipkin exporter configurations for high throughput

### 2. **Security Integration**
- Leverage service mesh mTLS with trace security context
- Implement trace-aware security policies
- Monitor security events correlated with distributed traces
- Configure proper RBAC for trace data access

### 3. **Operational Excellence**
- Establish SLI/SLO monitoring correlated with traces
- Implement trace-driven alerting for service degradation
- Configure trace retention policies based on compliance requirements
- Automate trace analysis for anomaly detection

### 4. **Service Mesh Governance**
- Implement consistent tracing across all mesh services
- Establish trace data quality standards
- Monitor service dependency changes via trace analysis
- Document service communication patterns from trace data

## Related Configurations

- [Basic TempoStack](../../e2e/compatibility/README.md) - TempoStack without service mesh
- [Monitoring Integration](../../e2e-openshift/monitoring/README.md) - Prometheus metrics with tracing
- [Multi-tenancy](../../e2e-openshift/multitenancy/README.md) - Multi-tenant service mesh tracing
- [Serverless Integration](../../e2e-openshift-serverless/tempo-serverless/README.md) - Knative with service mesh

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-ossm/ossm-tempostack --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs with `concurrent: false` due to high resource requirements and static namespace usage. The test demonstrates complete service mesh integration with automatic trace generation and Kiali visualization.