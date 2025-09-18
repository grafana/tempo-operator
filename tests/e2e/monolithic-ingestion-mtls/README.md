# TempoMonolithic with mTLS Ingestion Security

This configuration blueprint demonstrates how to secure trace ingestion in TempoMonolithic using mutual TLS (mTLS) authentication. This setup provides enterprise-grade security for trace data transmission, ensuring both client and server authentication through certificate-based validation.

## Overview

This test validates comprehensive mTLS security features:
- **Mutual TLS Authentication**: Two-way certificate validation between clients and Tempo
- **Custom Certificate Authority**: User-managed CA for certificate validation
- **OpenTelemetry Collector Integration**: Secure trace forwarding with certificate authentication
- **End-to-End Security**: Complete trace pipeline protection from generation to storage

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Trace Generator     │───▶│ OpenTelemetry Collector  │───▶│ TempoMonolithic     │
│ (telemetrygen)      │    │ ┌─────────────────────┐  │    │ ┌─────────────────┐ │
└─────────────────────┘    │ │ Client Certificate  │  │    │ │ Server Cert     │ │
                           │ │ + CA Bundle         │  │◀──▶│ │ + Client Auth   │ │
┌─────────────────────┐    │ └─────────────────────┘  │    │ └─────────────────┘ │
│ Custom CA           │    │ OTLP/gRPC + mTLS        │    │ OTLP Receiver       │
│ - Root Certificate  │───▶│ Port: 4317               │    │ Port: 4317          │
│ - Certificate Chain │    └──────────────────────────┘    └─────────────────────┘
└─────────────────────┘

Certificate Validation Flow:
Client ←→ Server: Mutual authentication with custom CA validation
```

## Prerequisites

- Kubernetes cluster with certificate management capabilities
- Tempo Operator installed
- OpenTelemetry Operator installed (for collector deployment)
- `kubectl` CLI access
- Basic understanding of TLS/SSL certificates

## Step-by-Step Deployment

### Step 1: Create Custom Certificate Authority

Set up the custom CA and certificates for mTLS authentication:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-ca
data:
  service-ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIFCTCCAvGgAwIBAgIUDbKo/R2ZknSoFDKMre3MmCrJkTQwDQYJKoZIhvcNAQEL
    BQAwEzERMA8GA1UEAwwITXlEZW1vQ0EwIBcNMjQwMTE5MTYyMzUyWhgPMjEyMzEy
    MjYxNjIzNTJaMBMxETAPBgNVBAMMCE15RGVtb0NBMIICIjANBgkqhkiG9w0BAQEF
    # ... (full certificate content)
    -----END CERTIFICATE-----
---
apiVersion: v1
kind: Secret
metadata:
  name: tempo-cert
data:
  tls.crt: LS0tLS1CRUdJTi... # Base64 encoded server certificate
  tls.key: LS0tLS1CRUdJTi... # Base64 encoded server private key
EOF
```

**Certificate Components**:
- **Custom CA (ConfigMap)**: Root certificate authority for validating client certificates
- **Server Certificate (Secret)**: Tempo's server certificate and private key for TLS termination
- **Certificate Chain**: Proper certificate hierarchy for trust validation

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 2: Deploy TempoMonolithic with mTLS Configuration

Create TempoMonolithic with TLS-enabled OTLP ingestion:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: custom-ca
          certName: tempo-cert
EOF
```

**Key Configuration Details**:

#### TLS Configuration
- `ingestion.otlp.grpc.tls.enabled: true`: Enables TLS for OTLP gRPC receiver
- `caName: custom-ca`: References the ConfigMap containing the CA certificate
- `certName: tempo-cert`: References the Secret containing server certificate and key

#### Security Features
- **Certificate Validation**: Clients must present valid certificates signed by the custom CA
- **Encrypted Transport**: All trace data encrypted in transit using TLS 1.2+
- **Authentication**: Mutual authentication prevents unauthorized access

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Verify TempoMonolithic TLS Configuration

Validate that Tempo is properly configured with mTLS:

```bash
# Check TempoMonolithic readiness
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify TLS configuration in StatefulSet
kubectl get statefulset tempo-simplest -o yaml | grep -A10 -B5 "tls\|cert\|ca"

# Check that certificates are mounted
kubectl describe pod tempo-simplest-0 | grep -A5 "Mounts:"

# Verify TLS port is configured
kubectl get svc tempo-simplest -o jsonpath='{.spec.ports[?(@.name=="otlp-grpc")].port}'
# Should return: 4317
```

### Step 4: Deploy OpenTelemetry Collector with mTLS Client

Create an OpenTelemetry Collector configured for mTLS communication:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: opentelemetry-collector-cert
data:
  tls.crt: LS0tLS1CRUdJTi... # Base64 encoded client certificate
  tls.key: LS0tLS1CRUdJTi... # Base64 encoded client private key
---
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: opentelemetry
spec:
  volumeMounts:
  - mountPath: /var/run/tls/receiver/ca
    name: custom-ca
    readOnly: true
  - mountPath: /var/run/tls/receiver/cert
    name: opentelemetry-collector-cert
    readOnly: true
  volumes:
  - configMap:
      defaultMode: 420
      name: custom-ca
    name: custom-ca
  - name: opentelemetry-collector-cert
    secret:
      defaultMode: 420
      secretName: opentelemetry-collector-cert
  config:
    exporters:
      debug: {}
      otlp:
        endpoint: tempo-simplest:4317
        tls:
          insecure: false
          ca_file: /var/run/tls/receiver/ca/service-ca.crt
          cert_file: /var/run/tls/receiver/cert/tls.crt
          key_file: /var/run/tls/receiver/cert/tls.key
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
    service:
      pipelines:
        traces:
          exporters: [otlp,debug]
          receivers: [otlp]
EOF
```

**Collector mTLS Configuration**:

#### Certificate Management
- **Volume Mounts**: CA and client certificates mounted as files
- **File Paths**: Certificates accessible at `/var/run/tls/receiver/`
- **Read-Only Access**: Certificates mounted as read-only for security

#### OTLP Exporter TLS Settings
- `insecure: false`: Enforces TLS connection validation
- `ca_file`: Path to CA certificate for server validation
- `cert_file`: Client certificate for mutual authentication
- `key_file`: Client private key for certificate authentication

**Reference**: [`02-install-otel.yaml`](./02-install-otel.yaml)

### Step 5: Generate and Verify Secure Traces

Test the complete mTLS pipeline with trace generation:

```bash
# Generate traces via the OpenTelemetry Collector
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
        - --otlp-endpoint=opentelemetry-collector:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF

# Verify traces were received by Tempo
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
          # Query Tempo's search API
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          
          echo "✓ Successfully received \$num_traces traces via mTLS"
      restartPolicy: Never
EOF
```

**Trace Flow Validation**:
1. **telemetrygen** → **OpenTelemetry Collector** (unencrypted, internal)
2. **OpenTelemetry Collector** → **TempoMonolithic** (mTLS encrypted)
3. **Query API** validates trace storage (HTTP API, unencrypted internal)

**References**: [`03-generate-traces.yaml`](./03-generate-traces.yaml), [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## mTLS Security Features

### 1. **Certificate-Based Authentication**

#### Client Certificate Validation
```yaml
# Tempo validates client certificates against the custom CA
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: custom-ca      # CA for client cert validation
          certName: tempo-cert   # Server certificate
```

#### Server Certificate Validation
```yaml
# Collector validates Tempo's server certificate
exporters:
  otlp:
    tls:
      insecure: false
      ca_file: /var/run/tls/receiver/ca/service-ca.crt  # Server cert validation
```

### 2. **Certificate Management Best Practices**

#### Certificate Generation (Example)
```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -key ca.key -sha256 -subj "/CN=MyDemoCA" -days 3650 -out ca.crt

# Generate server private key
openssl genrsa -out server.key 4096

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr -config <(
cat <<EOF
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = req_ext

[req_distinguished_name]
CN = tempo-simplest

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = tempo-simplest
EOF
)

# Sign server certificate with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extensions req_ext -extfile <(cat server.csr.config)
```

#### Kubernetes Secret Creation
```bash
# Create server certificate secret
kubectl create secret tls tempo-cert \
  --cert=server.crt \
  --key=server.key

# Create CA ConfigMap
kubectl create configmap custom-ca \
  --from-file=service-ca.crt=ca.crt
```

### 3. **Advanced TLS Configuration**

#### TLS Version and Cipher Control
```yaml
# Enhanced TLS configuration (via extraConfig)
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
```

#### Certificate Rotation Support
```yaml
# Automated certificate rotation setup
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: custom-ca
          certName: tempo-cert
          # Operator watches for certificate updates
```

## Security Validation and Testing

### 1. **Certificate Validation Testing**

```bash
# Test connection with valid certificate
openssl s_client -connect tempo-simplest:4317 \
  -cert client.crt -key client.key -CAfile ca.crt

# Test connection rejection without client certificate
openssl s_client -connect tempo-simplest:4317 \
  -CAfile ca.crt
# Should fail with certificate required error
```

### 2. **mTLS Connection Verification**

```bash
# Check TLS handshake details
kubectl exec deployment/opentelemetry-collector -- \
  openssl s_client -connect tempo-simplest:4317 \
  -cert /var/run/tls/receiver/cert/tls.crt \
  -key /var/run/tls/receiver/cert/tls.key \
  -CAfile /var/run/tls/receiver/ca/service-ca.crt \
  -verify_return_error
```

### 3. **Security Monitoring**

```bash
# Monitor TLS-related metrics
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep -E "(tls|ssl|cert)"

# Check TLS connection logs
kubectl logs tempo-simplest-0 | grep -i tls

# Monitor certificate expiration
kubectl get secret tempo-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates
```

## Troubleshooting mTLS Issues

### 1. **Certificate Problems**

#### Certificate Validation Errors
```bash
# Check certificate details
kubectl get secret tempo-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# Verify certificate chain
openssl verify -CAfile ca.crt server.crt

# Check certificate expiration
openssl x509 -in server.crt -noout -dates
```

#### Certificate Mount Issues
```bash
# Verify certificate mounts in pods
kubectl exec tempo-simplest-0 -- ls -la /etc/tls/
kubectl exec opentelemetry-collector-... -- ls -la /var/run/tls/receiver/

# Check certificate permissions
kubectl exec tempo-simplest-0 -- cat /etc/tls/server.crt
```

### 2. **TLS Connection Issues**

#### Connection Refused
```bash
# Check if TLS port is open
kubectl exec tempo-simplest-0 -- netstat -ln | grep 4317

# Test TLS connectivity
kubectl exec opentelemetry-collector-... -- \
  openssl s_client -connect tempo-simplest:4317 \
  -cert /var/run/tls/receiver/cert/tls.crt \
  -key /var/run/tls/receiver/cert/tls.key
```

#### Certificate Verification Failures
```bash
# Check Tempo logs for TLS errors
kubectl logs tempo-simplest-0 | grep -i "tls\|certificate\|handshake"

# Check collector logs for TLS errors
kubectl logs deployment/opentelemetry-collector | grep -i "tls\|certificate"
```

### 3. **Configuration Issues**

#### Invalid TLS Configuration
```bash
# Validate TempoMonolithic TLS config
kubectl get tempomonolithic simplest -o yaml | yq '.spec.ingestion.otlp.grpc.tls'

# Check generated Tempo configuration
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | yq '.server'
```

## Production Considerations

### 1. **Certificate Management**
- Use automated certificate management (cert-manager)
- Implement certificate rotation procedures
- Monitor certificate expiration dates
- Maintain certificate backup and recovery procedures

### 2. **Security Hardening**
```yaml
# Enhanced security configuration
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          caName: custom-ca
          certName: tempo-cert
          minVersion: "1.2"              # Minimum TLS version
          maxVersion: "1.3"              # Maximum TLS version
          cipherSuites:                  # Allowed cipher suites
            - "TLS_AES_256_GCM_SHA384"
            - "TLS_CHACHA20_POLY1305_SHA256"
```

### 3. **Monitoring and Alerting**
- Set up alerts for certificate expiration
- Monitor TLS handshake failures
- Track client authentication errors
- Monitor cipher suite usage and security

### 4. **Compliance and Auditing**
- Document certificate management procedures
- Implement certificate audit trails
- Ensure compliance with security policies
- Regular security assessments

## Related Configurations

- [TempoStack with TLS](../../e2e-openshift/tls-singletenant/README.md) - Distributed TLS setup
- [Basic TempoMonolithic](../monolithic-memory/README.md) - Non-TLS configuration
- [Receiver TLS Configuration](../receivers-tls/README.md) - TLS for all receiver protocols

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-ingestion-mtls
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires the OpenTelemetry Operator to be installed in the cluster for the OpenTelemetryCollector resource to function properly.

