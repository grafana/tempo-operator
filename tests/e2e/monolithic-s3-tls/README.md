# TempoMonolithic with S3 TLS Storage Backend

This configuration blueprint demonstrates how to deploy TempoMonolithic with a secure S3-compatible storage backend using TLS encryption. This setup provides enterprise-grade security for trace data storage, ensuring encrypted communication between Tempo and the object storage service while maintaining high performance and scalability.

## Overview

This test validates secure object storage integration features:
- **TLS-Enabled S3 Storage**: Encrypted communication with S3-compatible object storage
- **Certificate Authority Validation**: Custom CA for validating storage service certificates
- **MinIO with TLS**: Self-hosted S3-compatible storage with TLS termination
- **Secure Trace Persistence**: End-to-end encryption for stored trace data

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Trace Ingestion     │───▶│   TempoMonolithic        │───▶│ MinIO S3 Storage    │
│ - OTLP gRPC         │    │ ┌─────────────────────┐  │    │ ┌─────────────────┐ │
│ - OTLP HTTP         │    │ │ Tempo Components    │  │    │ │ TLS Encryption  │ │
│ - Jaeger formats    │    │ │ - Storage Client    │  │◀───┤ │ - Custom Certs  │ │
└─────────────────────┘    │ │ - TLS Validation    │  │    │ │ - Port 9000     │ │
                           │ └─────────────────────┘  │    │ └─────────────────┘ │
┌─────────────────────┐    │ S3 Client (HTTPS)       │    │ Bucket: tempo       │
│ Custom Certificate  │    └──────────────────────────┘    └─────────────────────┘
│ Authority           │
│ - Storage CA        │    TLS Communication:
│ - Server Certs      │    Tempo ←→ MinIO: HTTPS with certificate validation
└─────────────────────┘
```

## Prerequisites

- Kubernetes cluster with persistent volume support
- Tempo Operator installed
- `kubectl` CLI access
- Understanding of S3/object storage concepts
- Basic knowledge of TLS certificates

## Step-by-Step Deployment

### Step 1: Deploy TLS-Enabled MinIO Storage

Create the MinIO S3-compatible storage service with TLS encryption:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: minio-cert
data:
  private.key: LS0tLS1CRUdJTi... # Base64 encoded MinIO private key
  public.crt: LS0tLS1CRUdJTi...  # Base64 encoded MinIO certificate
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
              minio server --certs-dir=/root/.minio/certs /storage
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
            - name: cert
              mountPath: /root/.minio/certs
      volumes:
        - name: storage
          emptyDir: {}
        - name: cert
          secret:
            secretName: minio-cert
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
  endpoint: https://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**Key MinIO Configuration Elements**:

#### TLS Certificate Setup
- **Certificate Secret**: Contains MinIO's TLS certificate and private key
- **Certificate Mount**: Certificates mounted at `/root/.minio/certs`
- **TLS Server**: MinIO automatically enables HTTPS when certificates are present

#### MinIO Server Configuration
- `--certs-dir=/root/.minio/certs`: Configures TLS certificate directory
- **HTTPS Endpoint**: Service accessible via `https://minio:9000`
- **Bucket Creation**: Automatic creation of the `tempo` bucket
- **Access Credentials**: Simple authentication with access key/secret

#### Storage Secret for Tempo
- `endpoint: https://minio:9000`: HTTPS endpoint for secure communication
- **S3 Credentials**: Access key and secret for S3 API authentication
- **Bucket Configuration**: Dedicated `tempo` bucket for trace storage

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Create Certificate Authority for Storage Validation

Set up the CA certificate for validating MinIO's TLS certificate:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: storage-ca
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIFBzCCAu+gAwIBAgIUDwTaPC/j59gRvcgVyvWn0Qd8WaUwDQYJKoZIhvcNAQEL
    BQAwEzERMA8GA1UEAwwITXlEZW1vQ0EwHhcNMjMwODI0MTYxMTAyWhcNMzMwODIx
    # ... (full CA certificate content)
    -----END CERTIFICATE-----
EOF
```

**Certificate Authority Purpose**:
- **Server Validation**: Tempo validates MinIO's certificate against this CA
- **Trust Chain**: Establishes trusted communication channel
- **Security**: Prevents man-in-the-middle attacks on storage communications

### Step 3: Deploy TempoMonolithic with S3 TLS Backend

Create TempoMonolithic configured for secure S3 storage:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: minio
        tls:
          enabled: true
          caName: storage-ca
EOF
```

**Key Configuration Details**:

#### S3 Storage Backend
- `backend: s3`: Specifies S3-compatible object storage
- `s3.secret: minio`: References the MinIO connection secret
- **Automatic Configuration**: Operator reads endpoint, credentials, and bucket from secret

#### TLS Configuration
- `tls.enabled: true`: Enables TLS for S3 communication
- `tls.caName: storage-ca`: References CA ConfigMap for certificate validation
- **Certificate Validation**: Tempo validates MinIO certificate against custom CA

#### Default Storage Features
- **Block Storage**: Compressed trace blocks stored in S3
- **Metadata Management**: Efficient block indexing and retrieval
- **Compaction**: Automatic background compaction of trace blocks

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 4: Verify TLS Storage Configuration

Validate that TempoMonolithic is properly configured with TLS S3 storage:

```bash
# Check TempoMonolithic readiness
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify MinIO TLS service
kubectl get svc minio

# Test MinIO TLS connectivity
kubectl run test-tls --rm -i --tty --image=curlimages/curl -- \
  curl -k https://minio:9000/minio/health/live

# Check Tempo storage configuration
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | grep -A10 storage

# Verify CA certificate mounting
kubectl describe pod tempo-simplest-0 | grep -A5 "Volumes:"
```

Expected validation results:
- **TempoMonolithic Ready**: Successful S3 TLS connection established
- **MinIO Service**: HTTPS endpoint accessible on port 9000
- **TLS Handshake**: Successful certificate validation
- **Storage Configuration**: S3 backend with TLS parameters configured

### Step 5: Generate and Verify Traces

Test the complete TLS S3 storage pipeline:

```bash
# Generate traces
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
        - --otlp-endpoint=tempo-simplest:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF

# Verify traces are stored in S3
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
spec:
  template:
    spec:
      containers:
      - name: verify-traces
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Query traces via Tempo API
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          
          echo "✓ Successfully stored and retrieved \$num_traces traces via TLS S3"
      restartPolicy: Never
EOF
```

**References**: [`03-generate-traces.yaml`](./03-generate-traces.yaml), [`04-verify-traces.yaml`](./04-verify-traces.yaml)

### Step 6: Verify S3 Storage Contents (Optional)

Check that traces are actually stored in MinIO:

```bash
# Access MinIO to verify trace blocks
kubectl run minio-client --rm -i --tty --image=minio/mc -- \
  sh -c "
  mc config host add local https://minio:9000 tempo supersecret --insecure && \
  mc ls local/tempo/
  "

# Check trace block structure
kubectl exec tempo-simplest-0 -- \
  find /var/tempo -name "*.gz" | head -5
```

## S3 TLS Configuration Features

### 1. **Storage Backend Configuration**

#### Basic S3 TLS Setup
```yaml
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: minio-secret
        tls:
          enabled: true
          caName: storage-ca
```

#### Advanced S3 Configuration
```yaml
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: minio-secret
        tls:
          enabled: true
          caName: storage-ca
          certName: client-cert     # Optional: client certificate
          serverName: minio.local   # Optional: SNI override
          insecureSkipVerify: false # Optional: disable cert validation
```

### 2. **Certificate Management Options**

#### Self-Signed Certificates
```yaml
# For development/testing environments
spec:
  storage:
    traces:
      backend: s3
      s3:
        tls:
          enabled: true
          insecureSkipVerify: true  # Skip certificate validation
```

#### Production Certificate Setup
```yaml
# For production with proper CA hierarchy
spec:
  storage:
    traces:
      backend: s3
      s3:
        tls:
          enabled: true
          caName: production-ca
          certName: tempo-client-cert
```

### 3. **Performance Optimization**

#### Connection Pool Configuration
```yaml
spec:
  extraConfig:
    tempo:
      storage:
        trace:
          s3:
            http_config:
              max_idle_conns: 100
              max_idle_conns_per_host: 10
              idle_conn_timeout: 90s
              tls_handshake_timeout: 10s
```

#### Compression and Transfer
```yaml
spec:
  extraConfig:
    tempo:
      storage:
        trace:
          s3:
            part_size: 67108864      # 64MB parts
            sse_config:
              type: SSE-S3           # Server-side encryption
```

## Security Considerations

### 1. **Certificate Management**

#### Certificate Generation (Example)
```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -key ca.key -sha256 -subj "/CN=MyDemoCA" -days 3650 -out ca.crt

# Generate MinIO private key
openssl genrsa -out minio.key 4096

# Generate MinIO certificate signing request
openssl req -new -key minio.key -out minio.csr -config <(
cat <<EOF
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = req_ext

[req_distinguished_name]
CN = minio

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = minio
DNS.2 = minio.default.svc.cluster.local
EOF
)

# Sign MinIO certificate with CA
openssl x509 -req -in minio.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out minio.crt -days 365 -sha256
```

#### Kubernetes Secret Creation
```bash
# Create MinIO certificate secret
kubectl create secret generic minio-cert \
  --from-file=private.key=minio.key \
  --from-file=public.crt=minio.crt

# Create CA ConfigMap
kubectl create configmap storage-ca \
  --from-file=ca.crt=ca.crt
```

### 2. **Access Control and Authentication**

#### MinIO IAM Integration
```yaml
# Enhanced MinIO deployment with IAM
spec:
  template:
    spec:
      containers:
      - name: minio
        env:
        - name: MINIO_ROOT_USER
          value: admin
        - name: MINIO_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: minio-admin
              key: password
```

#### S3 Bucket Policies
```bash
# Create bucket policy for Tempo access
kubectl exec deployment/minio -- mc admin policy add local tempo-policy /tmp/tempo-policy.json
kubectl exec deployment/minio -- mc admin user add local tempo supersecret
kubectl exec deployment/minio -- mc admin policy set local tempo-policy user=tempo
```

### 3. **Network Security**

#### TLS Configuration Validation
```bash
# Test TLS connection manually
kubectl run test-tls --rm -i --tty --image=alpine/curl -- \
  curl -v --cacert /tmp/ca.crt https://minio:9000/minio/health/live

# Verify certificate chain
kubectl exec tempo-simplest-0 -- \
  openssl s_client -connect minio:9000 -CAfile /etc/ssl/certs/storage-ca.crt -verify_return_error
```

#### Network Policies (Optional)
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-storage-access
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: tempo-monolithic
  policyTypes:
  - Egress
  egress:
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: minio
    ports:
    - protocol: TCP
      port: 9000
```

## Troubleshooting TLS Storage Issues

### 1. **Certificate Problems**

#### Certificate Validation Errors
```bash
# Check Tempo logs for TLS errors
kubectl logs tempo-simplest-0 | grep -i "tls\|certificate\|x509"

# Verify MinIO certificate
kubectl get secret minio-cert -o jsonpath='{.data.public\.crt}' | base64 -d | openssl x509 -text -noout

# Test certificate chain
kubectl exec tempo-simplest-0 -- openssl verify -CAfile /etc/ssl/certs/storage-ca.crt /tmp/minio.crt
```

#### Certificate Mounting Issues
```bash
# Check MinIO certificate mounting
kubectl exec deployment/minio -- ls -la /root/.minio/certs/

# Verify CA availability in Tempo
kubectl exec tempo-simplest-0 -- ls -la /etc/ssl/certs/ | grep storage

# Check certificate permissions
kubectl exec deployment/minio -- stat /root/.minio/certs/public.crt
```

### 2. **S3 Connection Issues**

#### TLS Handshake Failures
```bash
# Test MinIO TLS endpoint
kubectl run debug --rm -i --tty --image=curlimages/curl -- \
  curl -v -k https://minio:9000/minio/health/live

# Check MinIO TLS configuration
kubectl exec deployment/minio -- cat /root/.minio/certs/public.crt | openssl x509 -noout -dates

# Verify DNS resolution
kubectl exec tempo-simplest-0 -- nslookup minio
```

#### S3 API Errors
```bash
# Check S3 credentials
kubectl get secret minio -o yaml

# Test S3 API access
kubectl run s3-test --rm -i --tty --image=amazon/aws-cli -- \
  sh -c "
  export AWS_ACCESS_KEY_ID=tempo
  export AWS_SECRET_ACCESS_KEY=supersecret
  export AWS_ENDPOINT_URL=https://minio:9000
  aws s3 ls s3://tempo/ --no-verify-ssl
  "

# Monitor S3 operations in Tempo logs
kubectl logs tempo-simplest-0 | grep -i "s3\|storage"
```

### 3. **Performance Issues**

#### TLS Overhead Monitoring
```bash
# Monitor TLS handshake performance
kubectl exec tempo-simplest-0 -- ss -i | grep :9000

# Check S3 operation latency
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep tempo_storage

# Monitor MinIO performance
kubectl port-forward svc/minio 9000:9000 &
curl -k https://localhost:9000/minio/prometheus/metrics
```

## Production Deployment Considerations

### 1. **High Availability MinIO**
```yaml
# MinIO cluster deployment for HA
apiVersion: minio.min.io/v2
kind: Tenant
metadata:
  name: minio-cluster
spec:
  image: quay.io/minio/minio:latest
  pools:
  - servers: 4
    volumesPerServer: 2
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 100Gi
  requestAutoCert: true  # Automatic TLS certificate management
```

### 2. **Certificate Management**
- Use cert-manager for automated certificate lifecycle
- Implement certificate rotation procedures
- Monitor certificate expiration
- Maintain certificate backup and recovery

### 3. **Monitoring and Alerting**
```bash
# Key metrics to monitor
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep -E "(storage|s3|tls)"

# Important metrics:
# - tempo_storage_backend_operations_total
# - tempo_storage_backend_operation_duration_seconds
# - tempo_storage_backend_errors_total
```

### 4. **Security Hardening**
- Use dedicated service accounts with minimal permissions
- Implement proper IAM policies for S3 access
- Enable S3 server-side encryption
- Regular security assessments and penetration testing

## Related Configurations

- [Basic S3 Storage](../compatibility/README.md) - TempoStack with MinIO (non-TLS)
- [TempoMonolithic Memory](../monolithic-memory/README.md) - In-memory storage option
- [mTLS Configuration](../monolithic-ingestion-mtls/README.md) - Client-side TLS security

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-s3-tls
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates both TLS connectivity and trace storage functionality, ensuring end-to-end security for the complete trace storage pipeline.

