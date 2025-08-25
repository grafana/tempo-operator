# TempoMonolithic with OpenShift Service Mesh and OpenTelemetry

This configuration blueprint demonstrates integration of TempoMonolithic with OpenShift Service Mesh (OSSM) and OpenTelemetry Collector for comprehensive distributed tracing in a service mesh environment. This setup provides end-to-end observability with service mesh telemetry, OpenTelemetry instrumentation, and Tempo trace storage.

## Overview

This test validates Service Mesh integration featuring:
- **OpenShift Service Mesh (OSSM)**: Istio-based service mesh with tracing integration
- **TempoMonolithic Backend**: Centralized trace storage and query interface
- **OpenTelemetry Collector**: Advanced telemetry collection and processing
- **Bookinfo Sample App**: Microservices application for trace generation
- **Kiali Integration**: Service mesh observability with distributed tracing
- **Dual Telemetry Paths**: Both service mesh and OpenTelemetry trace collection

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ Bookinfo Application    │───▶│   OpenShift Service      │───▶│ TempoMonolithic         │
│ ┌─────────────────────┐ │    │   Mesh (OSSM)            │    │ ┌─────────────────────┐ │
│ │ productpage         │ │    │ ┌─────────────────────┐  │    │ │ Trace Storage       │ │
│ │ details             │ │    │ │ Istio Proxy         │  │    │ │ - Jaeger UI         │ │
│ │ reviews             │ │    │ │ - Zipkin Exporter   │  │    │ │ - Search API        │ │
│ │ ratings             │ │    │ │ - Trace Generation  │  │    │ │ - OpenShift Route   │ │
│ └─────────────────────┘ │    │ └─────────────────────┘  │    │ └─────────────────────┘ │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘
                                                               
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ OpenTelemetry Collector │───▶│   Telemetry Processing   │───▶│ Trace Aggregation       │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │    │ ┌─────────────────────┐ │
│ │ OTLP Receivers      │ │    │ │ Batch Processing    │  │    │ │ Service Map         │ │
│ │ - gRPC/HTTP         │ │    │ │ - Memory Limiter    │  │    │ │ - Dependency Graph  │ │
│ │ - Jaeger Receiver   │ │    │ │ - Resource Detection│  │    │ │ - Performance       │ │
│ └─────────────────────┘ │    │ └─────────────────────┘  │    │ └─────────────────────┘ │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘

┌─────────────────────────┐    ┌──────────────────────────┐
│ Kiali Observability     │◀───│   Service Mesh Control   │
│ ┌─────────────────────┐ │    │   Plane                  │
│ │ Service Graph       │ │    │ ┌─────────────────────┐  │
│ │ - Traffic Flow      │ │    │ │ ServiceMeshControl  │  │
│ │ - Trace Links       │ │    │ │ Plane (SMCP)        │  │
│ │ - Health Status     │ │    │ │ - v2.5              │  │
│ └─────────────────────┘ │    │ └─────────────────────┘  │
└─────────────────────────┘    └──────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- OpenShift Service Mesh Operator installed
- OpenTelemetry Operator installed  
- Tempo Operator installed
- Sufficient cluster resources for mesh components
- Understanding of Istio/Service Mesh concepts

## Test Execution Overview

This test runs through a complete service mesh integration scenario:

1. **OSSM Installation**: Deploy OpenShift Service Mesh control plane
2. **TempoMonolithic Deployment**: Set up trace storage backend
3. **OpenTelemetry Setup**: Configure advanced telemetry collection  
4. **Kiali Configuration**: Enable observability dashboard
5. **Telemetry Integration**: Connect mesh tracing to Tempo
6. **Bookinfo Deployment**: Deploy sample microservices application
7. **Trace Generation**: Generate traffic and traces through mesh
8. **Verification**: Validate traces in both OpenTelemetry and mesh paths

## Key Configuration Elements

### Service Mesh Control Plane
```yaml
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: istio-system
  namespace: istio-system
spec:
  version: v2.5
  tracing:
    type: None  # Custom tracing configuration
    sampling: 10000
  meshConfig:
    extensionProviders:
      - name: tempo
        zipkin:
          service: tempo-simplest.tracing-system.svc.cluster.local
          port: 9411
```

### TempoMonolithic Integration
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
  namespace: tracing-system
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
```

### OpenTelemetry Collector
```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel
  namespace: tracing-system
spec:
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:
      jaeger:
        protocols:
          grpc:
          thrift_http:
    
    processors:
      batch:
      memory_limiter:
        limit_mib: 512
    
    exporters:
      otlp:
        endpoint: tempo-simplest:4317
        tls:
          insecure: true
    
    service:
      pipelines:
        traces:
          receivers: [otlp, jaeger]
          processors: [memory_limiter, batch]
          exporters: [otlp]
```

### Telemetry Configuration
```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: mesh-default
  namespace: istio-system
spec:
  tracing:
  - providers:
    - name: tempo
    randomSamplingPercentage: 100
```

## Service Mesh Integration Features

### 1. **Automatic Service Discovery**
- Service mesh automatically discovers all services in member namespaces
- Envoy proxies inject tracing headers into all HTTP requests
- Trace correlation across service boundaries without code changes

### 2. **Distributed Tracing**
- Complete request tracing across microservice boundaries  
- Automatic trace context propagation through service mesh
- Service topology and dependency mapping via traces

### 3. **Performance Monitoring**
- Request latency tracking across service calls
- Error rate monitoring and alerting
- Traffic flow analysis and capacity planning

### 4. **Security and Compliance**
- mTLS communication between all mesh services
- Certificate management and rotation
- Policy enforcement and audit logging

## OpenTelemetry Integration Benefits

### 1. **Advanced Telemetry Processing**
- Batch processing for improved performance
- Memory limiting to prevent resource exhaustion
- Resource detection for enhanced metadata

### 2. **Multi-Protocol Support**
- OTLP gRPC and HTTP receivers for modern applications
- Jaeger receivers for legacy application compatibility  
- Flexible telemetry collection strategies

### 3. **Telemetry Enhancement**
- Automatic resource attribute detection
- Span processing and enrichment
- Custom processors for business logic

### 4. **Vendor Neutrality**
- Standards-based telemetry collection
- Future-proof observability investment
- Easy migration between backend systems

## Kiali Observability Features

### 1. **Service Graph Visualization**
- Real-time service topology with traffic flow
- Service health and performance indicators
- Interactive exploration of service dependencies

### 2. **Distributed Trace Integration**
- Direct links from service graph to distributed traces
- Trace timeline visualization within Kiali
- Correlation between metrics and traces

### 3. **Traffic Management**
- Circuit breaker and retry policy visualization
- Load balancing and routing rule monitoring
- Security policy enforcement status

## Production Deployment Considerations

### 1. **Resource Planning**
```yaml
# TempoMonolithic resource allocation for mesh workloads
spec:
  resources:
    limits:
      memory: 4Gi
      cpu: 2000m
  # Scale based on mesh service count and traffic volume
```

### 2. **Mesh Configuration**
```yaml
# Production mesh settings
spec:
  version: v2.5
  tracing:
    sampling: 1000  # 10% sampling for production
  security:
    dataPlane:
      mtls: true
  policy:
    mixer:
      adapters:
        useAdapterCRDs: false
```

### 3. **High Availability**
```yaml
# Multi-replica control plane for HA
spec:
  runtime:
    components:
      pilot:
        deployment:
          replicas: 3
      galley:
        deployment:
          replicas: 2
```

## Troubleshooting Service Mesh Integration

### 1. **Service Mesh Connectivity**
```bash
# Check control plane status
oc get smcp -n istio-system

# Verify member namespaces
oc get smmr -n istio-system

# Check sidecar injection
oc get pods -n bookinfo -o jsonpath='{.items[*].spec.containers[*].name}'
```

### 2. **Tracing Configuration**
```bash
# Verify Tempo connectivity from mesh
oc exec -n istio-system deploy/istiod -- curl -v tempo-simplest.tracing-system:9411/api/v1/spans

# Check telemetry configuration
oc get telemetry -n istio-system

# Verify trace generation
oc logs -n istio-system deploy/istiod | grep -i tempo
```

### 3. **OpenTelemetry Issues**
```bash
# Check collector status
oc get opentelemetrycollector -n tracing-system

# Monitor collector logs
oc logs -n tracing-system deploy/otel-collector

# Verify trace pipeline
oc port-forward -n tracing-system svc/otel-collector 8889:8889 &
curl http://localhost:8889/metrics
```

## Related Configurations

- [OSSM TempoStack](../ossm-tempostack/README.md) - Distributed service mesh deployment
- [OSSM with Pure OpenTelemetry](../ossm-tempostack-otel/README.md) - OpenTelemetry-only mesh integration
- [Basic Service Mesh](../../e2e/compatibility/README.md) - Baseline distributed tracing

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-ossm/ossm-monolithic-otel
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test runs sequentially (`concurrent: false`) due to high resource requirements and static namespace usage. The test validates complete integration between service mesh, OpenTelemetry, and Tempo for enterprise observability requirements.

**References**:
- [`install-ossm.yaml`](./install-ossm.yaml) - Service mesh control plane
- [`install-tempo.yaml`](./install-tempo.yaml) - TempoMonolithic configuration
- [`install-otel-collector.yaml`](./install-otel-collector.yaml) - OpenTelemetry setup
- [`apply-telemetry-cr.yaml`](./apply-telemetry-cr.yaml) - Mesh telemetry configuration
- [`install-bookinfo.yaml`](./install-bookinfo.yaml) - Sample application

