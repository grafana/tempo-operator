# TempoStack with TLS Security - Single Tenant

This configuration blueprint demonstrates how to deploy TempoStack with TLS encryption enabled for secure trace ingestion and query operations. This setup showcases production-ready security configuration using OpenShift's built-in certificate management for enterprise environments requiring encrypted communications.

## Overview

This test validates a secure observability stack featuring:
- **TLS-Encrypted Ingestion**: Secure OTLP trace ingestion with certificate-based encryption
- **OpenShift Service CA**: Automatic certificate generation and management
- **Secure Communication**: End-to-end encryption between collector and Tempo components
- **Single-Tenant Mode**: Simplified security model for single-organization deployments
- **Route Integration**: Secure external access via OpenShift routes

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OTel Collector  │───▶│    TempoStack        │───▶│ MinIO Storage   │
│ ┌─────────────┐ │    │ ┌─────────────────┐  │    │ (S3 Compatible) │
│ │ TLS Client  │ │    │ │ TLS Distributor │  │    └─────────────────┘
│ │ - CA Cert   │ │    │ │ - Server Cert   │  │
│ │ - Secure    │ │    │ │ - Port 4317     │  │
│ │   Endpoints │ │    │ │ - Port 4318     │  │
│ └─────────────┘ │    │ └─────────────────┘  │
└─────────────────┘    └──────────────────────┘
          │                        │
          │ ┌──────────────────────┴─────────────────────┐
          │ │         OpenShift Service CA               │
          └▶│ ┌─────────────────┐ ┌─────────────────┐    │
            │ │ Client Certs    │ │ Server Certs    │    │
            │ │ (Pod Volumes)   │ │ (TLS Secrets)   │    │
            │ └─────────────────┘ └─────────────────┘    │
            └────────────────────────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.10+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- OpenShift Service CA enabled (default)
- `oc` CLI access

## Step-by-Step Deployment

### Step 1: Deploy MinIO Object Storage

Create the storage backend with standard configuration:

```bash
# Apply storage configuration
oc apply -f - <<EOF
# Standard MinIO deployment
# Reference: 00-install-storage.yaml
EOF
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Deploy TempoStack with TLS

Create TempoStack with TLS enabled for the distributor:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
  namespace: chainsaw-tls-singletenant
spec:
  storage:
    secret:
      name: minio
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 4Gi
        cpu: 2000m
  template:
    distributor:
      tls:
        enabled: true
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: route
EOF
```

**Key TLS Configuration Details**:

#### Distributor TLS Settings
- `distributor.tls.enabled: true`: Enables TLS encryption for trace ingestion
- Automatically generates server certificates via OpenShift Service CA
- Exposes secure endpoints on ports 4317 (gRPC) and 4318 (HTTP)

#### Certificate Management
- **Automatic Generation**: OpenShift Service CA creates and manages certificates
- **Certificate Rotation**: Automatic certificate renewal before expiration
- **CA Distribution**: CA certificate available in all pods via projected volumes

#### External Access
- `ingress.type: route`: Creates OpenShift route for Jaeger UI access
- Route automatically configured with edge TLS termination

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Verify TLS Configuration

Check that TLS is properly configured:

```bash
# Check TempoStack status
oc get tempostack simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Verify TLS secret creation
oc get secrets | grep tempo-simplest-distributor

# Check certificate details
oc get secret tempo-simplest-distributor-tls -o yaml
```

### Step 4: Deploy Secure OpenTelemetry Collector

Create collector configured for TLS communication:

```bash
oc apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: dev
  namespace: chainsaw-tls-singletenant
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
        endpoint: tempo-simplest-distributor.chainsaw-tls-singletenant.svc.cluster.local:4317
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
      otlphttp:
        endpoint: https://tempo-simplest-distributor.chainsaw-tls-singletenant.svc.cluster.local:4318
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

**Key TLS Configuration Details**:

#### Secure Endpoints
- **gRPC Endpoint**: `tempo-simplest-distributor:4317` with TLS
- **HTTP Endpoint**: `https://tempo-simplest-distributor:4318` with TLS

#### Certificate Validation
- `tls.insecure: false`: Enforces certificate validation
- `ca_file`: Uses OpenShift Service CA for certificate verification
- **Automatic CA Mount**: Service CA certificate automatically mounted in all pods

#### Dual Protocol Support
- Separate pipelines for gRPC and HTTP trace ingestion
- Both protocols secured with TLS encryption

**Reference**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)

### Step 5: Generate Sample Traces

Create traces using the secure collector:

```bash
oc apply -f - <<EOF
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
        - --otlp-endpoint=dev-collector:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Configuration Notes**:
- Traces sent to collector, which forwards via secure TLS to Tempo
- End-to-end encryption from collector to Tempo distributor
- Collector handles TLS complexity, keeping trace generation simple

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 6: Verify Secure Trace Flow

Test that traces flow securely through the TLS-enabled pipeline:

```bash
oc apply -f - <<EOF
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
            http://tempo-simplest-query-frontend:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          
          echo "Successfully verified \$num_traces traces via TLS pipeline"
      restartPolicy: Never
EOF
```

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## Key Features Demonstrated

### 1. **Transport Layer Security**
- **Encryption in Transit**: All trace data encrypted during transmission
- **Certificate-based Authentication**: Mutual trust via PKI infrastructure
- **Protocol Security**: Secure gRPC and HTTPS protocols

### 2. **OpenShift Certificate Management**
- **Service CA Integration**: Automatic certificate lifecycle management
- **Certificate Distribution**: CA certificates available cluster-wide
- **Rotation Handling**: Seamless certificate renewal without downtime

### 3. **Production Security**
- **Zero-Trust Network**: Encrypted communications by default
- **Identity Verification**: Certificate-based component authentication
- **Compliance Ready**: Meets enterprise security requirements

### 4. **Operational Simplicity**
- **Automatic Configuration**: OpenShift handles certificate complexity
- **Transparent Operation**: Applications work without certificate management
- **Monitoring Integration**: TLS status visible in component metrics

## TLS Configuration Options

### Custom Certificate Authority

For environments requiring custom CA:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca-secret
type: Opaque
data:
  ca.crt: <base64-encoded-ca-certificate>
---
spec:
  template:
    distributor:
      tls:
        enabled: true
        caName: "custom-ca-secret"
```

### Mutual TLS (mTLS)

For enhanced security with client certificates:

```yaml
spec:
  template:
    distributor:
      tls:
        enabled: true
        clientAuth: "RequireAndVerifyClientCert"
        minVersion: "1.3"
        cipherSuites:
          - "TLS_AES_128_GCM_SHA256"
          - "TLS_AES_256_GCM_SHA384"
```

### Advanced TLS Settings

```yaml
spec:
  template:
    distributor:
      tls:
        enabled: true
        # Custom certificate configuration
        certName: "tempo-distributor-cert"
        keyName: "tempo-distributor-key"
        # TLS version constraints
        minVersion: "1.2"
        maxVersion: "1.3"
        # Custom cipher suites
        preferServerCipherSuites: true
```

## Security Validation

### Certificate Verification

```bash
# Check certificate details
oc get secret tempo-simplest-distributor-tls -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# Verify certificate chain
oc exec deployment/dev-collector -- \
  openssl s_client -connect tempo-simplest-distributor:4317 -CAfile /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
```

### TLS Handshake Testing

```bash
# Test TLS connectivity
oc run tls-test --image=curlimages/curl --rm -it -- \
  curl -v --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  https://tempo-simplest-distributor:4318/debug/ready
```

### Certificate Rotation Verification

```bash
# Monitor certificate expiration
oc get secret tempo-simplest-distributor-tls -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -dates

# Check Service CA status
oc get configmap -n openshift-service-ca service-ca-bundle -o yaml
```

## Troubleshooting

### TLS Connection Issues

```bash
# Check distributor logs for TLS errors
oc logs -l app.kubernetes.io/component=distributor | grep -i tls

# Verify service CA is working
oc get pods -n openshift-service-ca-operator

# Test certificate trust
oc exec deployment/dev-collector -- \
  curl -v --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
  https://tempo-simplest-distributor:4318/metrics
```

### Certificate Problems

```bash
# Check certificate generation
oc describe secret tempo-simplest-distributor-tls

# Verify Service CA injection
oc get configmap -o yaml | grep service-ca-bundle

# Force certificate regeneration
oc delete secret tempo-simplest-distributor-tls
# Certificate will be automatically regenerated
```

### Collector Configuration Issues

```bash
# Check collector configuration
oc get opentelemetrycollector dev -o yaml

# Test collector connectivity
oc logs -l app.kubernetes.io/component=opentelemetry-collector | grep -i error

# Verify CA file presence
oc exec deployment/dev-collector -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/
```

### Common TLS Errors

1. **Certificate Verification Failed**:
   ```bash
   # Check if Service CA is properly configured
   oc get csr | grep service-ca
   ```

2. **TLS Handshake Timeout**:
   ```bash
   # Verify network connectivity
   oc exec deployment/dev-collector -- telnet tempo-simplest-distributor 4317
   ```

3. **CA Certificate Not Found**:
   ```bash
   # Ensure service-ca-bundle annotation is present
   oc get configmap -o yaml | grep service.beta.openshift.io/inject-cabundle
   ```

## Performance Considerations

### TLS Overhead

TLS adds computational overhead:
- **CPU Usage**: 5-10% increase for encryption/decryption
- **Latency**: 1-2ms additional latency per request
- **Memory**: Additional memory for TLS buffers

### Optimization Strategies

```yaml
# Optimize TLS performance
spec:
  template:
    distributor:
      tls:
        enabled: true
        # Use modern, efficient cipher suites
        cipherSuites:
          - "TLS_AES_256_GCM_SHA384"
          - "TLS_CHACHA20_POLY1305_SHA256"
        # Disable older, slower protocols
        minVersion: "1.3"
      # Increase resources for TLS processing
      resources:
        requests:
          cpu: "200m"
          memory: "512Mi"
        limits:
          cpu: "1000m"
          memory: "1Gi"
```

## Production Considerations

### 1. **Certificate Management**
- Monitor certificate expiration dates
- Implement automated certificate rotation testing
- Backup certificate authority keys securely
- Document certificate recovery procedures

### 2. **Security Hardening**
- Use TLS 1.3 for maximum security
- Disable weak cipher suites
- Implement certificate pinning where appropriate
- Regular security audits and penetration testing

### 3. **Monitoring and Alerting**
- Monitor TLS handshake failures
- Alert on certificate expiration
- Track TLS performance metrics
- Log security events for audit trails

### 4. **Compliance**
- Document TLS configuration for audits
- Ensure compliance with industry standards (PCI DSS, SOX)
- Implement data classification and encryption policies
- Regular compliance assessments

## Related Configurations

- [Basic TempoStack](../../e2e/compatibility/README.md) - Non-TLS baseline configuration
- [Multi-tenant TLS](../multitenancy-rbac/README.md) - TLS with multi-tenancy
- [mTLS Configuration](../receivers-mtls/README.md) - Mutual TLS setup
- [Monitoring with TLS](../monitoring/README.md) - Secure monitoring configuration

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/tls-singletenant --config .chainsaw-openshift.yaml
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates end-to-end TLS encryption from collector to Tempo components.