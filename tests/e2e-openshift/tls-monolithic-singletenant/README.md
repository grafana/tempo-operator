# TempoMonolithic with OpenShift Service CA TLS (Single Tenant)

This configuration blueprint demonstrates TempoMonolithic deployment with TLS-enabled ingestion using OpenShift's native service CA infrastructure for certificate management. This setup provides enterprise-grade security for single-tenant trace ingestion while leveraging OpenShift's automatic certificate provisioning and rotation capabilities.

## Overview

This test validates OpenShift-native TLS integration features:
- **OpenShift Service CA**: Automatic TLS certificate generation and rotation
- **Single-Tenant TLS**: Secure trace ingestion without multi-tenant complexity
- **Dual Protocol TLS**: Both OTLP gRPC and HTTP with TLS encryption
- **Native Certificate Management**: Zero-configuration TLS using OpenShift infrastructure
- **Complete Ingestion Flow**: End-to-end TLS validation with OpenTelemetry Collector

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ OpenTelemetry Collector │───▶│   TempoMonolithic        │───▶│ MinIO Storage           │
│ - OTLP gRPC/HTTP        │    │   TLS Receivers          │    │ - S3 Compatible         │
│ - TLS Client            │    │ ┌─────────────────────┐  │    │ - Persistent Volume     │
│ - Service CA Validation │    │ │ OTLP gRPC (4317)    │  │    │ - Bucket: tempo         │
└─────────────────────────┘    │ │ - TLS Enabled       │  │    └─────────────────────────┘
                               │ │ - Service CA Cert   │  │
┌─────────────────────────┐    │ └─────────────────────┘  │    ┌─────────────────────────┐
│ OpenShift Service CA    │───▶│ ┌─────────────────────┐  │───▶│ Jaeger UI               │
│ - Automatic Cert       │    │ │ OTLP HTTP (4318)    │  │    │ - OpenShift Route       │
│ - Certificate Rotation  │    │ │ - TLS Enabled       │  │    │ - External Access       │
│ - CA Bundle Injection  │    │ │ - HTTPS Protocol    │  │    │ - Single Tenant View    │
└─────────────────────────┘    │ └─────────────────────┘  │    └─────────────────────────┘
                               │ Jaeger UI (HTTP:16686)   │
┌─────────────────────────┐    └──────────────────────────┘    ┌─────────────────────────┐
│ Certificate Validation  │                                    │ Trace Generation        │
│ - Service CA Trust      │    TLS Communication:              │ - telemetrygen          │
│ - Automatic Rotation    │    Collector ←→ Tempo: TLS 1.2+   │ - 10 sample traces      │
│ - Zero Configuration    │    with OpenShift service CA      │ - OTLP Protocol         │
└─────────────────────────┘                                   └─────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Understanding of OpenShift service CA infrastructure
- Knowledge of TLS certificate management in OpenShift

## Step-by-Step Configuration

### Step 1: Deploy Persistent Storage Backend

Create MinIO with persistent storage for trace data:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    app.kubernetes.io/name: minio
  name: minio
  namespace: chainsaw-tls-mono-st
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
  namespace: chainsaw-tls-mono-st
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
  namespace: chainsaw-tls-mono-st
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
  namespace: chainsaw-tls-mono-st
stringData:
  endpoint: http://minio:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**Storage Configuration Details**:
- **Persistent Volume**: 2Gi PVC for durable trace storage
- **S3 Compatibility**: MinIO provides S3-compatible object storage
- **Namespace Isolation**: Deployed in dedicated test namespace

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Deploy TempoMonolithic with OpenShift Service CA TLS

Create TempoMonolithic with TLS-enabled receivers using OpenShift's service CA:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: mono
  namespace: chainsaw-tls-mono-st
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
      http:
        tls:
          enabled: true
EOF
```

**Key Configuration Elements**:

#### TLS-Enabled Ingestion
- `otlp.grpc.tls.enabled: true`: Enables TLS for OTLP gRPC receiver (port 4317)
- `otlp.http.tls.enabled: true`: Enables TLS for OTLP HTTP receiver (port 4318)
- **OpenShift Service CA**: Automatic certificate generation without explicit cert configuration

#### Single-Tenant Configuration
- **No Multi-Tenancy**: Simple single-tenant deployment for focused TLS testing
- **Standard Storage**: Uses default storage configuration with MinIO
- **Jaeger UI**: External access via OpenShift Route

#### Automatic Certificate Management
When TLS is enabled without explicit certificate configuration, the Tempo Operator:
1. **Requests Service CA**: Triggers OpenShift service CA certificate generation
2. **Certificate Injection**: Service CA automatically provisions TLS certificates
3. **CA Bundle Mount**: Injects service CA bundle for client validation
4. **Automatic Rotation**: Service CA handles certificate lifecycle

**Generated TLS Resources**:
- **Service**: Annotated for service CA certificate generation
- **Secret**: Contains TLS certificate and private key (automatically created)
- **ConfigMap**: Service CA bundle for client certificate validation
- **Pod Volumes**: Certificates automatically mounted in Tempo container

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Deploy OpenTelemetry Collector with Service CA Trust

Create an OpenTelemetry Collector configured to trust OpenShift service CA:

```bash
oc apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: dev
  namespace: chainsaw-tls-mono-st
spec:
  config: |
    receivers:
      otlp/grpc:
        protocols:
          grpc:
      otlp/http:
        protocols:
          http:
    exporters:
      otlp:
        endpoint: tempo-mono.chainsaw-tls-mono-st.svc.cluster.local:4317
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
      otlphttp:
        endpoint: https://tempo-mono.chainsaw-tls-mono-st.svc.cluster.local:4318
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
    service:
      telemetry:
        logs:
          level: "DEBUG"
          development: true
          encoding: "json"
      pipelines:
        traces/grpc:
          receivers: [otlp/grpc]
          exporters: [otlp]
        traces/http:
          receivers: [otlp/http]
          exporters: [otlphttp]
EOF
```

**OpenTelemetry Collector TLS Configuration**:

#### Service CA Trust Configuration
- `ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"`: Uses OpenShift service CA for certificate validation
- **Automatic Injection**: Service CA bundle automatically mounted by OpenShift
- **No Manual Certificate Management**: Zero-configuration TLS trust

#### Dual Protocol Exporters
```yaml
otlp:
  endpoint: tempo-mono.chainsaw-tls-mono-st.svc.cluster.local:4317
  tls: {insecure: false, ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"}

otlphttp:
  endpoint: https://tempo-mono.chainsaw-tls-mono-st.svc.cluster.local:4318
  tls: {insecure: false, ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"}
```

#### TLS Validation Features
- `insecure: false`: Enforces TLS certificate validation
- **FQDN Endpoints**: Uses fully qualified service names for proper certificate validation
- **Protocol Specificity**: gRPC uses default port, HTTP uses HTTPS scheme

**Reference**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)

### Step 4: Generate Traces via TLS-Enabled Collector

Create traces that flow through the secure TLS pipeline:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces
  namespace: chainsaw-tls-mono-st
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
        args:
        - traces
        - --otlp-endpoint=dev-collector:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Generation Flow**:
1. **telemetrygen** → **OpenTelemetry Collector** (unencrypted, internal)
2. **OpenTelemetry Collector** → **TempoMonolithic** (TLS encrypted with service CA)
3. **TempoMonolithic** → **MinIO** (internal storage, unencrypted)

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 5: Verify TLS-Secured Trace Ingestion

Validate that traces were successfully ingested through the TLS pipeline:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
  namespace: chainsaw-tls-mono-st
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
          curl -v -G \
            http://tempo-mono:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          echo "✓ Successfully verified \$num_traces traces via TLS ingestion"
      restartPolicy: Never
EOF
```

**Verification Process**:
- **Internal Query**: Uses internal HTTP API (non-TLS) for verification
- **Trace Count**: Validates exactly 10 traces were ingested
- **Pipeline Validation**: Confirms end-to-end TLS trace flow

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## OpenShift Service CA Features

### 1. **Automatic Certificate Generation**

#### Service Annotation for Certificate Request
When TLS is enabled, the operator annotates the service:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: tempo-mono
  annotations:
    service.beta.openshift.io/serving-cert-secret-name: tempo-mono-tls
spec:
  ports:
  - name: otlp-grpc
    port: 4317
    protocol: TCP
  - name: otlp-http
    port: 4318
    protocol: TCP
```

#### Automatic Secret Creation
OpenShift service CA creates a secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tempo-mono-tls
  annotations:
    service.beta.openshift.io/expiry: "2025-01-01T00:00:00Z"
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi... # Base64 encoded certificate
  tls.key: LS0tLS1CRUdJTi... # Base64 encoded private key
```

#### CA Bundle Injection
Service CA bundle is injected into pods:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: service-ca-bundle
  annotations:
    service.beta.openshift.io/inject-cabundle: "true"
data:
  service-ca.crt: |
    -----BEGIN CERTIFICATE-----
    # OpenShift service CA certificate
    -----END CERTIFICATE-----
```

### 2. **Certificate Lifecycle Management**

#### Automatic Rotation
- **Rotation Period**: Certificates automatically rotate before expiration
- **Grace Period**: Overlap period ensures no downtime during rotation
- **Pod Restart**: Automatic pod restart when certificates are renewed

#### Validation and Trust
- **Subject Alternative Names**: Automatically includes service DNS names
- **Trust Chain**: Service CA is trusted by all OpenShift pods by default
- **Certificate Validation**: Full certificate chain validation

### 3. **Integration with OpenShift Components**

#### Router and Route Integration
```yaml
# External access with service CA
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: tempo-mono-jaegerui
spec:
  tls:
    termination: edge
    # Router trusts service CA automatically
  to:
    kind: Service
    name: tempo-mono
    weight: 100
```

#### Service Mesh Integration
```yaml
# Service mesh with service CA
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: tempo-mono-tls
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: tempo-monolithic
  mtls:
    mode: STRICT
```

## Advanced OpenShift TLS Configuration

### 1. **Custom Certificate Configuration**

#### Using Custom CA with Service CA
```yaml
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: custom-ca-bundle  # Custom CA ConfigMap
          certName: custom-tls-cert # Custom certificate Secret
```

#### Certificate Rotation Policies
```yaml
# Custom certificate lifecycle
apiVersion: v1
kind: Secret
metadata:
  name: custom-tls-cert
  annotations:
    service.beta.openshift.io/expiry: "720h"  # 30 days
    cert-rotation.operator.openshift.io/rotation-interval: "168h"  # 7 days
```

### 2. **Multi-Protocol TLS Configuration**

#### Protocol-Specific TLS Settings
```yaml
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          minVersion: "1.2"
          cipherSuites:
          - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      http:
        tls:
          enabled: true
          minVersion: "1.3"
```

#### Jaeger Receiver TLS
```yaml
spec:
  ingestion:
    jaeger:
      grpc:
        tls:
          enabled: true
      thriftHttp:
        tls:
          enabled: true
```

### 3. **Security Policy Integration**

#### Pod Security Standards
```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: tempo
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop: [ALL]
      readOnlyRootFilesystem: true
```

#### Network Policies for TLS
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-tls-access
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: tempo-monolithic
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: opentelemetry-collector
    ports:
    - protocol: TCP
      port: 4317  # OTLP gRPC TLS
    - protocol: TCP
      port: 4318  # OTLP HTTP TLS
```

## Production Deployment Considerations

### 1. **Certificate Management Strategy**

#### Service CA vs External CA
```yaml
# Production: Use enterprise CA
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: enterprise-ca-bundle
          certName: enterprise-tls-cert

# Development: Use service CA
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          # Service CA used automatically
```

#### Certificate Monitoring
```yaml
# Alert on certificate expiration
alert: ServiceCACertExpiring
expr: (cert_expiry_timestamp - time()) / 86400 < 30
for: 1h
annotations:
  summary: "Service CA certificate expiring in {{ $value }} days"
```

### 2. **Performance Optimization**

#### TLS Performance Tuning
```yaml
spec:
  extraConfig:
    tempo:
      server:
        grpc_tls_config:
          min_version: VersionTLS12
          max_version: VersionTLS13
          cipher_suites:
          - TLS_AES_256_GCM_SHA384
          - TLS_CHACHA20_POLY1305_SHA256
        http_tls_config:
          min_version: VersionTLS12
          prefer_server_cipher_suites: true
```

#### Connection Pooling
```yaml
# OpenTelemetry Collector connection management
exporters:
  otlp:
    endpoint: tempo-mono:4317
    tls:
      insecure: false
      ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
    sending_queue:
      enabled: true
      num_consumers: 10
    retry_on_failure:
      enabled: true
      max_elapsed_time: 300s
```

### 3. **Security Hardening**

#### Disable Insecure Protocols
```yaml
spec:
  extraConfig:
    tempo:
      server:
        grpc_tls_config:
          min_version: VersionTLS12
        http_tls_config:
          min_version: VersionTLS12
        # Disable HTTP ingestion entirely for security
      ingestion:
        otlp:
          http:
            enabled: false
```

#### Mutual TLS (mTLS)
```yaml
# Require client certificates
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          clientAuth: RequireAndVerifyClientCert
          caName: client-ca-bundle
```

## Troubleshooting OpenShift Service CA TLS

### 1. **Certificate Generation Issues**

#### Check Service CA Operator
```bash
# Verify service CA operator is running
oc get pods -n openshift-service-ca-operator

# Check service CA operator logs
oc logs -n openshift-service-ca-operator deployment/service-ca-operator

# Verify service CA ConfigMap
oc get configmap service-ca-bundle -n openshift-service-ca-operator
```

#### Service Annotation Problems
```bash
# Check service annotations
oc get service tempo-mono -o yaml | grep -A5 annotations

# Verify certificate secret creation
oc get secret tempo-mono-tls

# Check certificate content
oc get secret tempo-mono-tls -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

### 2. **TLS Connection Issues**

#### Certificate Validation Failures
```bash
# Test TLS connection manually
oc run tls-test --image=curlimages/curl --rm -it -- \
  curl -v --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  https://tempo-mono:4318/

# Check certificate chain
openssl s_client -connect tempo-mono:4317 -servername tempo-mono \
  -CAfile /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
```

#### OpenTelemetry Collector TLS Issues
```bash
# Check collector logs for TLS errors
oc logs deployment/dev-collector | grep -i "tls\|cert\|handshake"

# Verify service CA bundle mount
oc exec deployment/dev-collector -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Test service CA connectivity
oc exec deployment/dev-collector -- \
  curl --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  https://tempo-mono:4318/
```

### 3. **Service CA Bundle Issues**

#### CA Bundle Injection Problems
```bash
# Check service CA bundle ConfigMap
oc get configmap service-ca-bundle -o yaml

# Verify CA bundle injection annotation
oc get configmap <configmap-name> -o yaml | grep inject-cabundle

# Check pod service CA mount
oc describe pod tempo-mono-0 | grep -A10 "service-ca"
```

#### Trust Store Validation
```bash
# Verify service CA is in trust store
oc exec tempo-mono-0 -- cat /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt

# Check certificate trust chain
oc exec tempo-mono-0 -- openssl verify \
  -CAfile /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  /etc/tls/tls.crt
```

## Related Configurations

- [TLS Receivers](../../e2e/monolithic-receivers-tls/README.md) - Generic TLS receiver configuration
- [mTLS Ingestion](../../e2e/monolithic-ingestion-mtls/README.md) - Mutual TLS authentication
- [TempoStack TLS](../tls-singletenant/README.md) - Distributed TLS setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/tls-monolithic-singletenant
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates OpenShift-native TLS integration using service CA for automatic certificate management. The test demonstrates zero-configuration TLS security suitable for production OpenShift environments with single-tenant trace ingestion requirements.

