# Tempo Serverless with OpenTelemetry Collector

This test demonstrates deploying a TempoStack instance integrated with OpenShift Serverless workloads via an OpenTelemetry Collector. It covers the full trace lifecycle from Knative service app generation through an OTel Collector to Tempo for storage and querying.

## Test Overview

### Components
- **TempoStack**: Distributed Tempo deployment with S3 storage and Jaeger Query UI
- **MinIO**: S3-compatible object storage backend for trace data
- **OpenTelemetry Collector**: Intermediate trace collection layer using Zipkin receiver
- **Knative Serving & Eventing**: OpenShift Serverless platform components
- **Serverless Application**: Knative service that generates traces when invoked

### Trace Flow
1. HTTP requests to Knative service → Generate traces
2. Traces sent to OpenTelemetry Collector (Zipkin protocol)
3. OTel Collector forwards traces to TempoStack distributor (OTLP gRPC)
4. Traces stored in MinIO via TempoStack
5. Traces queryable through Jaeger Query UI

## Deployment Steps

### 1. Install MinIO Object Storage
```bash
kubectl apply -f install-minio.yaml
```

Key configuration from [`install-minio.yaml`](install-minio.yaml):
- Creates PersistentVolumeClaim with 2Gi storage for MinIO data
- Deploys MinIO server with tempo/supersecret credentials
- Creates Secret with S3 endpoint configuration for TempoStack

### 2. Deploy TempoStack
```bash
kubectl apply -f install-tempo.yaml
```

Key configuration from [`install-tempo.yaml`](install-tempo.yaml):
- Configures TempoStack to use MinIO S3 backend
- Enables Jaeger Query UI with OpenShift Route
- Allocates 2Gi memory and 2000m CPU for processing
- 200M storage size for traces

### 3. Create OpenTelemetry Collector
```bash
kubectl apply -f create-otel-collector.yaml
```

Key configuration from [`create-otel-collector.yaml`](create-otel-collector.yaml):
- Deploys OTel Collector in deployment mode
- Configures Zipkin receiver for trace ingestion
- Forwards traces to TempoStack distributor via OTLP gRPC
- Uses insecure TLS connection to Tempo distributor

### 4. Set Up Knative Infrastructure
```bash
kubectl apply -f create-knative-serving.yaml
kubectl apply -f create-knative-eventing.yaml
```

Creates:
- **KnativeServing**: Enables serverless application deployment
- **KnativeEventing**: Enables event-driven architecture capabilities

### 5. Deploy Serverless Application
```bash
kubectl apply -f create-knative-app.yaml
```

Key configuration from [`create-knative-app.yaml`](create-knative-app.yaml):
- Knative Service using `quay.io/openshift-knative/helloworld:v1.2`
- Minimum scale of 1 pod for consistent availability
- Target concurrency of 1 request per pod
- Auto-scaling based on request load

### 6. Generate Traces
```bash
kubectl apply -f generate-traces.yaml
```

This step:
- Sends HTTP requests to the Knative service
- Triggers trace generation in the serverless application
- Traces are sent to OTel Collector via Zipkin protocol

### 7. Verify Traces
```bash
kubectl apply -f verify-traces.yaml
```

This step:
- Queries TempoStack via Jaeger Query API
- Validates that traces were successfully ingested and stored
- Confirms end-to-end trace collection from serverless workloads

## Key Features Tested

### Serverless Integration
- ✅ Knative service deployment and scaling
- ✅ Trace generation from serverless workloads
- ✅ Auto-scaling behavior with trace collection

### OpenTelemetry Integration
- ✅ OTel Collector as intermediate trace processing layer
- ✅ Zipkin protocol trace ingestion
- ✅ OTLP gRPC forwarding to TempoStack

### Tempo Configuration
- ✅ Distributed TempoStack deployment on OpenShift
- ✅ S3-compatible storage backend integration
- ✅ Jaeger Query UI with OpenShift Route access
- ✅ Multi-component trace processing pipeline

### End-to-End Validation
- ✅ Serverless → OTel Collector → TempoStack trace flow
- ✅ Trace persistence and queryability
- ✅ OpenShift-native routing and access

## Architecture

```
[Knative Service] 
        ↓ (Zipkin traces)
[OpenTelemetry Collector]
        ↓ (OTLP gRPC)
[TempoStack Distributor]
        ↓
[MinIO S3 Storage]
        ↑
[Jaeger Query UI] ← [OpenShift Route]
```

This test validates that the Tempo Operator can successfully deploy and manage a TempoStack instance that integrates seamlessly with OpenShift Serverless workloads through an OpenTelemetry Collector intermediary, providing a complete observability solution for serverless applications.

