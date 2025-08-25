# OpenShift Service Mesh with TempoStack and OpenTelemetry Collector Integration

This test validates the comprehensive integration between OpenShift Service Mesh (OSSM), TempoStack, and OpenTelemetry Collector for distributed tracing in microservices environments. It demonstrates how to configure OSSM to send traces through an OpenTelemetry Collector to TempoStack, providing multiple ingestion paths and enhanced observability capabilities.

## Test Overview

### Purpose
- **Multi-Protocol Trace Ingestion**: Tests both Zipkin and OTLP trace collection through OpenTelemetry Collector
- **Service Mesh Integration**: Validates OSSM automatic trace generation and routing to TempoStack
- **OpenTelemetry Collector Pipeline**: Demonstrates OTel Collector as an intermediary for trace processing
- **Dual Ingestion Paths**: Tests both direct OSSM→TempoStack and OTel→TempoStack trace flows

### Components
- **OpenShift Service Mesh (OSSM)**: Based on Istio, provides automatic service-to-service tracing
- **TempoStack**: Distributed Tempo deployment for trace storage and querying
- **OpenTelemetry Collector**: Trace collection, processing, and forwarding middleware
- **Kiali**: Service mesh observability and tracing visualization
- **Bookinfo Application**: Sample microservices application for trace generation
- **MinIO**: S3-compatible storage backend for TempoStack

## Architecture Overview

```
[Bookinfo Microservices] 
        ↓ (Istio Envoy Sidecars)
[OSSM Service Mesh] 
        ↓ (Zipkin Protocol)
[OpenTelemetry Collector]
        ↓ (OTLP Protocol)
[TempoStack Distributor]
        ↓
[MinIO Storage]

[External OTLP Applications]
        ↓ (OTLP HTTP/gRPC)
[OpenTelemetry Collector]
        ↓ (OTLP Protocol)
[TempoStack Distributor]
```

## Deployment Steps

### 1. Install OpenShift Service Mesh
```bash
kubectl apply -f install-ossm.yaml
```

Key configuration from [`install-ossm.yaml`](install-ossm.yaml):
```yaml
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: istio-system
  namespace: istio-system
spec:
  version: v2.5
  tracing:
    type: None  # Disable built-in Jaeger, use TempoStack instead
    sampling: 10000
  meshConfig:
    extensionProviders:
      - name: tempo
        zipkin:
          service: simplest-collector.tracing-system.svc.cluster.local
          port: 9411  # Zipkin receiver on OTel Collector
  addons:
    kiali:
      enabled: true
    grafana:
      enabled: true
---
kind: ServiceMeshMemberRoll
spec:
  members:
    - tracing-system
    - bookinfo
    - otlp-app
```

### 2. Deploy MinIO Storage Backend
```bash
kubectl apply -f install-minio.yaml
```

### 3. Install TempoStack with Route Access
```bash
kubectl apply -f install-tempo.yaml
```

Key configuration from [`install-tempo.yaml`](install-tempo.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
  namespace: tracing-system
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 3Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: route
```

### 4. Configure Kiali for TempoStack Integration
```bash
kubectl patch -f update-kiali.yaml
```

This updates Kiali to use TempoStack as the tracing backend instead of the default Jaeger.

### 5. Deploy OpenTelemetry Collector
```bash
kubectl apply -f install-otel-collector.yaml
```

Key configuration from [`install-otel-collector.yaml`](install-otel-collector.yaml):
```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: simplest
  namespace: tracing-system
spec:
  config: |
    receivers:
      zipkin: {}  # Receive traces from OSSM
      otlp:       # Receive traces from external applications
        protocols:
          grpc:
          http:
    
    exporters:
      otlp:
        endpoint: tempo-simplest-distributor.tracing-system.svc.cluster.local:4317
        tls:
          insecure: true
    
    service:
      pipelines:
        traces:
          receivers: [zipkin, otlp]
          processors: []
          exporters: [otlp]
```

### 6. Enable OSSM Tempo Provider
```bash
kubectl apply -f apply-telemetry-cr.yaml
```

This configures the service mesh to send traces to the Tempo provider:
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

### 7. Deploy Bookinfo Sample Application
```bash
kubectl apply -f install-bookinfo.yaml
```

### 8. Generate Service Mesh Traces
```bash
# Generate traces through Bookinfo application
for i in {1..20}; do
  curl http://$(oc -n istio-system get route istio-ingressgateway -o jsonpath='{.spec.host}')/productpage
  sleep 1
done
```

### 9. Generate Direct OTLP Traces
```bash
kubectl apply -f generate-traces-otel.yaml
```

This creates jobs that send traces directly to the OpenTelemetry Collector via both HTTP and gRPC:
```yaml
# HTTP OTLP traces
args:
- traces
- --otlp-endpoint=simplest-collector.tracing-system.svc.cluster.local:4318
- --otlp-http
- --service=telemetrygen-http

# gRPC OTLP traces  
args:
- traces
- --otlp-endpoint=simplest-collector.tracing-system.svc.cluster.local:4317
- --service=telemetrygen-grpc
```

### 10. Verify Traces in Multiple Locations
```bash
kubectl apply -f verify-traces.yaml          # Check traces in Kiali
kubectl apply -f verify-traces-otel.yaml     # Check OTLP traces in TempoStack
```

## Key Features Tested

### OpenTelemetry Collector Integration
- ✅ Multi-protocol trace ingestion (Zipkin + OTLP HTTP/gRPC)
- ✅ Trace processing and forwarding pipeline
- ✅ OSSM to OTel Collector to TempoStack trace flow
- ✅ Direct application to OTel Collector ingestion

### Service Mesh Tracing
- ✅ Automatic trace generation for service-to-service communication
- ✅ Envoy sidecar trace collection and forwarding
- ✅ Custom Tempo extension provider configuration
- ✅ 100% sampling rate for comprehensive trace capture

### TempoStack Configuration
- ✅ Distributed Tempo deployment with OpenShift Route access
- ✅ Multiple ingestion endpoints (direct OTLP + via OTel Collector)
- ✅ Jaeger Query UI integration for trace visualization
- ✅ S3-compatible storage backend (MinIO)

### Observability Integration
- ✅ Kiali integration with TempoStack tracing backend
- ✅ Service topology visualization with trace correlation
- ✅ Multiple trace visualization interfaces (Kiali + Jaeger UI)
- ✅ End-to-end trace flow validation

## Trace Flow Validation

### Service Mesh Traces (via Zipkin)
1. **Bookinfo microservices** generate service calls
2. **Istio Envoy sidecars** capture trace spans
3. **Service Mesh Control Plane** forwards traces via Zipkin protocol
4. **OpenTelemetry Collector** receives Zipkin traces on port 9411
5. **OTel Collector** converts and forwards to TempoStack via OTLP
6. **TempoStack** stores traces and makes them available for querying

### Direct OTLP Traces
1. **External applications** send traces via OTLP HTTP (port 4318) or gRPC (port 4317)
2. **OpenTelemetry Collector** receives OTLP traces directly
3. **OTel Collector** forwards traces to TempoStack via OTLP
4. **TempoStack** stores traces with different service identifiers

## Environment Requirements

### OpenShift Prerequisites
- OpenShift Service Mesh Operator installed
- OpenTelemetry Operator installed
- Tempo Operator installed
- Sufficient cluster resources for distributed deployment

### Namespace Configuration
- **istio-system**: Service mesh control plane and components
- **tracing-system**: TempoStack and OpenTelemetry Collector
- **bookinfo**: Sample application for service mesh tracing
- **otlp-app**: Direct OTLP trace generation applications

## Comparison with Other OSSM Tests

### vs. ossm-tempostack (Direct Integration)
- **Additional Layer**: Uses OpenTelemetry Collector as intermediary
- **Enhanced Processing**: Enables trace processing, filtering, and enrichment
- **Multi-Protocol**: Supports both Zipkin (OSSM) and OTLP (direct apps)
- **Unified Pipeline**: Single collector handles multiple trace sources

### vs. ossm-monolithic-otel (Monolithic Alternative)
- **Distributed Architecture**: Better scalability and fault tolerance
- **Resource Allocation**: Higher resource requirements but better performance
- **Production Ready**: More suitable for production service mesh deployments
- **Component Isolation**: Independent scaling of trace ingestion and storage

## Troubleshooting

### Common Issues

**OpenTelemetry Collector Problems**:
- Verify OTel Collector pod is running in `tracing-system` namespace
- Check collector logs for OTLP endpoint connectivity issues
- Ensure Zipkin receiver is properly configured for OSSM integration

**Service Mesh Integration Issues**:
- Confirm ServiceMeshMemberRoll includes all required namespaces
- Verify Tempo extension provider configuration in ServiceMeshControlPlane
- Check that Telemetry CR is applied and active in istio-system

**Trace Flow Validation Failures**:
- Ensure bookinfo application is properly deployed with sidecar injection
- Verify OSSM ingress gateway route is accessible for trace generation
- Check TempoStack distributor logs for incoming trace processing

**Kiali Integration Problems**:
- Confirm Kiali configuration patch was successful
- Verify Kiali can reach TempoStack query frontend
- Check that traces are visible in both Kiali and Jaeger UI

## Advanced Configuration

### Custom OTel Collector Processing
Enhance the OpenTelemetry Collector configuration with processors:
```yaml
processors:
  batch:
    timeout: 1s
    send_batch_size: 1024
  resource:
    attributes:
    - key: environment
      value: production
      action: upsert
  probabilistic_sampler:
    sampling_percentage: 50
```

### Service Mesh Sampling Configuration
Adjust sampling rates for different services:
```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: custom-sampling
  namespace: bookinfo
spec:
  tracing:
  - providers:
    - name: tempo
    randomSamplingPercentage: 50  # 50% sampling for bookinfo services
```

This test demonstrates a comprehensive observability stack that combines the automatic tracing capabilities of OpenShift Service Mesh with the flexible trace processing of OpenTelemetry Collector and the scalable storage of TempoStack, providing multiple ingestion paths and enhanced trace management capabilities.
