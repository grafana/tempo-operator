# TempoStack with Object Storage - Compatibility Test

This configuration blueprint demonstrates how to deploy a distributed Tempo observability stack using TempoStack with object storage backend. This setup provides compatibility between Tempo's native API and Jaeger's query API, enabling seamless integration with existing observability tools.

## Overview

This test validates a complete observability stack featuring:
- **TempoStack**: Distributed Tempo deployment with separate components (querier, distributor, ingester, compactor)
- **MinIO Object Storage**: Local S3-compatible storage backend
- **Dual Query APIs**: Both Tempo and Jaeger query interfaces
- **Trace Generation & Verification**: End-to-end trace flow validation

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Trace Generator │───▶│   TempoStack     │───▶│ MinIO Storage   │
│ (telemetrygen)  │    │ - Distributor    │    │ (S3 Compatible) │
└─────────────────┘    │ - Ingester       │    └─────────────────┘
                       │ - Querier        │
┌─────────────────┐    │ - Compactor      │
│ Query Clients   │◀───│ - Query Frontend │
│ - Tempo API     │    └──────────────────┘
│ - Jaeger API    │
└─────────────────┘
```

## Prerequisites

- Kubernetes cluster with sufficient resources (2Gi memory, 2000m CPU)
- Tempo Operator installed
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Deploy MinIO Object Storage

Create the storage backend with PVC, deployment, service, and secret:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    app.kubernetes.io/name: minio
  name: minio
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
---
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
          persistentVolumeClaim:
            claimName: minio
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
  name: minio
stringData:
  endpoint: http://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Deploy TempoStack

Create the distributed Tempo deployment with dual query APIs:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 200M
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: ingress
EOF
```

**Key Configuration Details**:
- `storage.secret`: References the MinIO credentials
- `storageSize`: Allocates 200MB for trace storage
- `resources.total`: Sets resource limits for all components
- `jaegerQuery.enabled`: Enables Jaeger-compatible API endpoint
- `ingress.type`: Configures ingress for external access

**Reference**: [`01-install.yaml`](./01-install.yaml)

### Step 3: Verify Deployment

Wait for TempoStack to be ready:

```bash
kubectl get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

Check all components are running:

```bash
kubectl get pods -l app.kubernetes.io/managed-by=tempo-operator
```

### Step 4: Generate Sample Traces

Create traces using OpenTelemetry's telemetrygen tool:

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
        - --otlp-endpoint=tempo-simplest-distributor:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Configuration Notes**:
- `--otlp-endpoint`: Points to TempoStack distributor service
- `--otlp-insecure`: Uses unencrypted connection (suitable for testing)
- `--traces=10`: Generates exactly 10 traces for verification

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 5: Verify Traces via Jaeger API

Test the Jaeger-compatible query interface:

```bash
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces-jaeger
spec:
  template:
    spec:
      containers:
      - name: verify-traces-jaeger
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Query Tempo's native search API
          curl -v -G \
            http://tempo-simplest-query-frontend:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Tempo API returned \$num_traces instead of 10 traces."
            exit 1
          fi

          # Query Jaeger-compatible API
          curl -v -G \
            http://tempo-simplest-query-frontend:16686/api/traces \
            --data-urlencode "service=telemetrygen" | tee /tmp/jaeger.out
          
          num_traces=\$(jq ".data | length" /tmp/jaeger.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Jaeger API returned \$num_traces instead of 10 traces."
            exit 1
          fi
      restartPolicy: Never
EOF
```

**API Endpoints Tested**:
- **Tempo Native API**: `http://tempo-simplest-query-frontend:3200/api/search`
- **Jaeger Compatible API**: `http://tempo-simplest-query-frontend:16686/api/traces`

**Reference**: [`04-verify-traces-jaeger.yaml`](./04-verify-traces-jaeger.yaml)

### Step 6: Verify Traces via Grafana API

Test the Grafana-compatible query interface:

```bash
kubectl apply -f - <<EOF
# Similar verification job for Grafana API endpoints
# See 05-verify-traces-grafana.yaml for complete configuration
EOF
```

**Reference**: [`05-verify-traces-grafana.yaml`](./05-verify-traces-grafana.yaml)

## Key Features Demonstrated

### 1. **Dual Query Interface Compatibility**
- Native Tempo API for advanced TraceQL queries
- Jaeger API for existing Jaeger UI integrations
- Grafana API for dashboard integration

### 2. **Object Storage Integration**
- S3-compatible storage (MinIO) for trace persistence
- Configurable retention and compaction policies
- Scalable storage backend

### 3. **Resource Management**
- Centralized resource allocation across all components
- Memory and CPU limits for predictable performance
- Storage size configuration

### 4. **Service Discovery**
- Automatic service creation for all components
- Consistent naming convention: `tempo-{name}-{component}`
- Load balancing across component replicas

## Troubleshooting

### Check TempoStack Status
```bash
kubectl describe tempostack simplest
```

### View Component Logs
```bash
# Distributor logs
kubectl logs -l app.kubernetes.io/component=distributor

# Query frontend logs  
kubectl logs -l app.kubernetes.io/component=query-frontend

# Ingester logs
kubectl logs -l app.kubernetes.io/component=ingester
```

### Test Storage Connectivity
```bash
# Port-forward to MinIO
kubectl port-forward svc/minio 9000:9000

# Access MinIO UI at http://localhost:9000
# Credentials: tempo / supersecret
```

### Verify Trace Ingestion
```bash
# Check distributor metrics
kubectl port-forward svc/tempo-simplest-distributor 3200:3200
curl http://localhost:3200/metrics | grep tempo_distributor
```

## Production Considerations

### 1. **Storage Backend**
- Use managed object storage (AWS S3, GCS, Azure Blob) for production
- Configure appropriate retention policies
- Enable backup and disaster recovery

### 2. **Resource Scaling**
- Adjust component replicas based on traffic volume
- Monitor resource utilization and scale accordingly
- Use Horizontal Pod Autoscaler for dynamic scaling

### 3. **Security**
- Enable TLS for all component communications
- Use proper authentication for storage access
- Configure network policies for traffic isolation

### 4. **Monitoring**
- Set up monitoring for all Tempo components
- Configure alerts for storage and query performance
- Monitor trace ingestion rates and query latency

## Related Configurations

- [Monolithic Memory Storage](../monolithic-memory/README.md) - In-memory storage setup
- [TLS Single Tenant](../tls-singletenant/README.md) - TLS-enabled configuration  
- [Multi-tenancy Setup](../../e2e-openshift/multitenancy/README.md) - Multi-tenant deployment

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/compatibility
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)