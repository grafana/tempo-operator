# OpenShift Serverless (Knative) with TempoStack Integration

This test demonstrates how to integrate TempoStack with OpenShift Serverless (Knative) to enable distributed tracing for serverless workloads. The configuration enables automatic trace collection from both Knative Serving and Eventing components.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Knative App    │    │  Knative Serving │    │   TempoStack    │
│  (Serverless)   │───▶│   & Eventing     │───▶│   Distributor   │
│                 │    │                  │    │   :9411/v2      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         │                        │                       ▼
         │                        │              ┌─────────────────┐
         │                        │              │   Jaeger UI     │
         │                        │              │   (Route)       │
         │                        │              └─────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐    ┌──────────────────┐
│   Trace Data    │    │  Zipkin Backend  │
│   Collection    │    │  Configuration   │
└─────────────────┘    └──────────────────┘
```

## Test Components

### TempoStack Configuration
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Features**: Jaeger UI with OpenShift Route, S3-compatible storage (MinIO)
- **Resources**: 2Gi memory, 2000m CPU, 200M storage
- **Access**: Jaeger UI available via OpenShift route with edge termination

### Knative Serving Setup
- **File**: [`create-knative-serving.yaml`](./create-knative-serving.yaml)
- **Tracing**: Zipkin backend configured to TempoStack distributor endpoint
- **Sample Rate**: 10% trace sampling
- **Endpoint**: `http://tempo-serverless-distributor.chainsaw-tempo-serverless.svc:9411/api/v2/spans`

### Knative Eventing Setup
- **File**: [`create-knative-eventing.yaml`](./create-knative-eventing.yaml)
- **Tracing**: Same Zipkin configuration as Serving
- **Integration**: Events and triggers automatically traced

### Serverless Application
- **File**: [`create-knative-app.yaml`](./create-knative-app.yaml)
- **Image**: `quay.io/openshift-knative/helloworld:v1.2`
- **Scaling**: Min scale 1, target 1 instance
- **Resources**: 200m CPU request

## Quick Start

### Prerequisites
- OpenShift cluster with Serverless Operator installed
- Tempo Operator deployed
- MinIO or S3-compatible storage available

### Step-by-Step Deployment

1. **Install Dependencies and Storage**
   ```bash
   # Deploy MinIO storage backend
   kubectl apply -f install-minio.yaml
   kubectl wait --for=condition=ready pod -l app=minio -n chainsaw-tempo-serverless --timeout=300s
   ```

2. **Deploy TempoStack**
   ```bash
   # Create TempoStack instance with Jaeger UI
   kubectl apply -f install-tempo.yaml
   kubectl wait --for=condition=ready tempostack serverless -n chainsaw-tempo-serverless --timeout=300s
   ```

3. **Configure Knative Serving**
   ```bash
   # Deploy Knative Serving with tracing enabled
   kubectl apply -f create-knative-serving.yaml
   kubectl wait --for=condition=ready knativeserving serverless -n knative-serving --timeout=300s
   ```

4. **Configure Knative Eventing**
   ```bash
   # Deploy Knative Eventing with tracing enabled
   kubectl apply -f create-knative-eventing.yaml
   kubectl wait --for=condition=ready knativeeventing serverless -n knative-eventing --timeout=300s
   ```

5. **Deploy Serverless Application**
   ```bash
   # Create sample Go serverless application
   kubectl apply -f create-knative-app.yaml
   kubectl wait --for=condition=ready ksvc serverless-app -n chainsaw-tempo-serverless --timeout=300s
   ```

6. **Generate and Verify Traces**
   ```bash
   # Generate traces by calling the serverless app
   kubectl apply -f generate-traces.yaml
   
   # Verify traces are collected in Tempo
   kubectl apply -f verify-traces.yaml
   ```

## Testing Procedure

The complete test is defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml) and executes these steps:

1. **Storage Setup**: Install MinIO object store
2. **Tempo Deployment**: Create TempoStack with Jaeger UI
3. **Knative Configuration**: Setup Serving and Eventing with tracing
4. **App Deployment**: Deploy sample serverless application
5. **Trace Generation**: Generate traffic to create traces
6. **Verification**: Confirm traces are collected and queryable

## Trace Configuration Details

### Zipkin Integration
- **Backend**: Zipkin protocol for trace ingestion
- **Endpoint**: TempoStack distributor service on port 9411
- **Format**: OpenTelemetry traces converted to Zipkin format
- **Sampling**: 10% of requests traced (configurable)

### Automatic Instrumentation
- **Knative Serving**: Automatically instruments HTTP requests/responses
- **Knative Eventing**: Traces event flows and trigger executions
- **Application**: Traces internal application spans (if instrumented)

## Accessing Traces

1. **Get Jaeger UI Route**
   ```bash
   oc get route -n chainsaw-tempo-serverless
   ```

2. **Query Traces**
   - Service: `serverless-app`
   - Operation: HTTP requests to serverless endpoints
   - Tags: Knative-specific metadata (revision, service, etc.)

## Production Considerations

### Performance
- **Cold Start Impact**: Tracing adds minimal overhead to cold starts
- **Sampling**: Adjust sample rate based on traffic volume and retention needs
- **Resource Allocation**: Scale TempoStack components based on trace volume

### Security
- **Network Policies**: Ensure Knative components can reach TempoStack distributor
- **RBAC**: Configure appropriate permissions for trace collection
- **TLS**: Consider enabling TLS for trace transmission in production

### Scaling
- **Knative Scaling**: Configure appropriate autoscaling parameters
- **Tempo Scaling**: Scale distributors and ingesters based on trace throughput
- **Storage**: Plan retention and compaction strategies for trace data

## Troubleshooting

### Common Issues

1. **No Traces Appearing**
   ```bash
   # Check Knative configuration
   oc get knativeserving serverless -n knative-serving -o yaml
   oc get knativeeventing serverless -n knative-eventing -o yaml
   
   # Verify distributor endpoint
   oc get svc tempo-serverless-distributor -n chainsaw-tempo-serverless
   ```

2. **Connectivity Issues**
   ```bash
   # Test distributor accessibility
   oc exec -n knative-serving deployment/controller -- curl -v http://tempo-serverless-distributor.chainsaw-tempo-serverless.svc:9411/api/v2/spans
   ```

3. **Serverless App Not Scaling**
   ```bash
   # Check Knative service status
   oc get ksvc serverless-app -n chainsaw-tempo-serverless
   oc describe ksvc serverless-app -n chainsaw-tempo-serverless
   ```

### Debug Commands
```bash
# Check trace ingestion
oc logs -n chainsaw-tempo-serverless deployment/tempo-serverless-distributor

# Monitor serverless application
oc logs -n chainsaw-tempo-serverless -l app=helloworld-go

# Verify Knative configuration
oc get cm config-tracing -n knative-serving -o yaml
oc get cm config-tracing -n knative-eventing -o yaml
```

## Related Resources

- [OpenShift Serverless Documentation](https://docs.openshift.com/serverless/)
- [Knative Tracing Documentation](https://knative.dev/docs/serving/observability/tracing/)
- [TempoStack Configuration Guide](../../../docs/tempo-configuration.md)
- [Jaeger Query API Reference](https://www.jaegertracing.io/docs/apis/)