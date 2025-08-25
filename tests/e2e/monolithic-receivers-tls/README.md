# TempoMonolithic with TLS-Enabled Receivers

This configuration blueprint demonstrates how to secure trace ingestion in TempoMonolithic by enabling TLS encryption for all receiver protocols. This setup provides comprehensive transport security for trace data while supporting multiple ingestion protocols (OTLP gRPC and HTTP) with proper certificate management.

## Overview

This test validates comprehensive TLS receiver security features:
- **Multi-Protocol TLS**: TLS encryption for both OTLP gRPC and HTTP receivers
- **Certificate Management**: Custom certificates for server-side TLS termination
- **Jaeger UI Integration**: Secure trace visualization alongside encrypted ingestion
- **OpenTelemetry Collector Integration**: Secure trace forwarding with TLS validation

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ OpenTelemetry       │───▶│   TempoMonolithic        │───▶│ In-Memory Storage   │
│ Collector           │    │ ┌─────────────────────┐  │    │ - Traces            │
│ ┌─────────────────┐ │    │ │ TLS Receivers       │  │    │ - Jaeger UI         │
│ │ OTLP/gRPC       │◀┼────┤ │ • gRPC:4317 (TLS)   │  │    └─────────────────────┘
│ │ OTLP/HTTP       │◀┼────┤ │ • HTTP:4318 (TLS)   │  │
│ └─────────────────┘ │    │ │ • Custom Certs      │  │
└─────────────────────┘    │ └─────────────────────┘  │
                           │ Jaeger UI (HTTP:16686)   │
┌─────────────────────┐    └──────────────────────────┘
│ Custom Certificate  │
│ Authority           │    Certificate Validation:
│ - Server Certs      │    Client ←→ Server: TLS 1.2+ with custom CA
│ - CA Bundle         │
└─────────────────────┘
```

## Prerequisites

- Kubernetes cluster with certificate management capabilities
- Tempo Operator installed
- OpenTelemetry Operator installed (for collector deployment)
- `kubectl` CLI access
- Basic understanding of TLS certificates and transport security

## Step-by-Step Deployment

### Step 1: Create TLS Certificates and CA

Set up the certificate authority and server certificates for TLS termination:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-ca
data:
  service-ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIFZTCCA02gAwIBAgIUFDK4W5lEpkYZyOpFrKphNi6cu+0wDQYJKoZIhvcNAQEL
    BQAwQjELMAkGA1UEBhMCTVgxFTATBgNVBAcMDERlZmF1bHQgQ2l0eTEcMBoGA1UE
    CgwTRGVmYXVsdCBDb21wYW55IEx0ZDAeFw0yNDA3MTEwMDI2MjhaFw0yNzA1MDEw
    # ... (full certificate content)
    -----END CERTIFICATE-----
---
apiVersion: v1
kind: Secret
metadata:
  name: custom-cert
data:
  tls.crt: LS0tLS1CRUdJTi... # Base64 encoded server certificate
  tls.key: LS0tLS1CRUdJTi... # Base64 encoded server private key
EOF
```

**Certificate Components**:
- **CA ConfigMap**: Root certificate authority for client-side validation
- **Server Certificate Secret**: TLS certificate and private key for Tempo's receivers
- **Certificate Validity**: Proper Subject Alternative Names (SAN) for service names

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 2: Deploy TempoMonolithic with TLS Receivers

Create TempoMonolithic with TLS-enabled OTLP receivers:

```bash
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: simplest
spec:
  jaegerui:
    enabled: true
  ingestion:
    otlp:
      grpc:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
      http:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
EOF
```

**Key Configuration Details**:

#### OTLP gRPC Receiver with TLS
- `grpc.enabled: true`: Enables OTLP gRPC receiver on port 4317
- `grpc.tls.enabled: true`: Activates TLS encryption for gRPC connections
- `grpc.tls.certName: custom-cert`: References the server certificate secret

#### OTLP HTTP Receiver with TLS  
- `http.enabled: true`: Enables OTLP HTTP receiver on port 4318
- `http.tls.enabled: true`: Activates TLS encryption for HTTP connections
- `http.tls.certName: custom-cert`: Uses the same certificate for HTTP TLS

#### Jaeger UI Integration
- `jaegerui.enabled: true`: Provides trace visualization interface
- **Unencrypted Access**: Jaeger UI remains on HTTP for internal access

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Verify TLS Configuration

Validate that TempoMonolithic is properly configured with TLS receivers:

```bash
# Check TempoMonolithic readiness
kubectl get tempomonolithic simplest -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True

# Verify TLS-enabled services
kubectl get svc tempo-simplest -o yaml | grep -A10 ports

# Check certificate mounting in pod
kubectl describe pod tempo-simplest-0 | grep -A5 "Volumes:"

# Verify TLS configuration in StatefulSet
kubectl get statefulset tempo-simplest -o yaml | grep -A5 -B5 "tls\|cert"
```

Expected validation results:
- **Service Ports**: 4317 (OTLP gRPC), 4318 (OTLP HTTP), 16686 (Jaeger UI)
- **Certificate Mount**: Custom certificate available in pod
- **TLS Configuration**: Proper TLS termination configured

### Step 4: Deploy OpenTelemetry Collector with TLS Client

Create an OpenTelemetry Collector configured to connect to TLS-enabled receivers:

```bash
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: opentelemetry
spec:
  volumeMounts:
    - mountPath: /var/run/tls/receiver/ca
      name: custom-ca
      readOnly: true
  volumes:
    - configMap:
        defaultMode: 420
        name: custom-ca
      name: custom-ca
  config:
    exporters:
      otlp:
        endpoint: tempo-simplest:4317
        tls:
          insecure: false
          ca_file: "/var/run/tls/receiver/ca/service-ca.crt"
      otlphttp:
        endpoint: https://tempo-simplest:4318
        tls:
          insecure: false
          ca_file: "/var/run/tls/receiver/ca/service-ca.crt"
    receivers:
      otlp/grpc:
        protocols:
          grpc: {}
      otlp/http:
        protocols:
          http: {}
    extensions:
      health_check:
        endpoint: 0.0.0.0:13133
    service:
      extensions: [health_check]
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

**Collector TLS Configuration**:

#### gRPC Exporter (OTLP)
- `endpoint: tempo-simplest:4317`: Direct gRPC connection to TLS-enabled receiver
- `tls.insecure: false`: Enforces TLS certificate validation
- `ca_file`: Path to CA certificate for server validation

#### HTTP Exporter (OTLP HTTP)  
- `endpoint: https://tempo-simplest:4318`: HTTPS connection to TLS-enabled HTTP receiver
- **HTTPS Protocol**: Explicit HTTPS scheme for encrypted HTTP transport
- **Certificate Validation**: Same CA file used for server certificate validation

#### Dual Pipeline Setup
- **gRPC Pipeline**: `otlp/grpc` → `otlp` exporter
- **HTTP Pipeline**: `otlp/http` → `otlphttp` exporter
- **Health Check**: Monitoring endpoint for collector health

**Reference**: [`02-install-otel.yaml`](./02-install-otel.yaml)

### Step 5: Generate and Verify Secure Traces

Test both TLS-enabled receivers with trace generation:

```bash
# Generate traces via OpenTelemetry Collector (dual pipelines)
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

# Verify traces received through both pipelines
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
spec:
  template:
    spec:
      containers:
      - name: verify-traces-http
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Verify traces via HTTP pipeline
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          echo "Found \$num_traces traces via HTTP pipeline"
          
      - name: verify-traces-grpc
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          # Verify traces via gRPC pipeline
          curl -v -G \
            http://tempo-simplest:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ge 10 ]]; then
            echo "✓ Successfully received traces via TLS-enabled receivers"
          else
            echo "✗ Expected at least 10 traces, got \$num_traces"
            exit 1
          fi
      restartPolicy: Never
EOF
```

**Trace Flow Validation**:
1. **telemetrygen** → **OpenTelemetry Collector** (unencrypted, internal)
2. **OpenTelemetry Collector** → **TempoMonolithic** (TLS encrypted, dual protocols)
3. **Query API** validates trace storage (HTTP API, unencrypted internal)

**References**: [`03-generate-traces.yaml`](./03-generate-traces.yaml), [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## TLS Receiver Configuration

### 1. **Protocol-Specific TLS Settings**

#### OTLP gRPC with TLS
```yaml
spec:
  ingestion:
    otlp:
      grpc:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
          # Optional: min/max TLS versions
          minVersion: "1.2"
          maxVersion: "1.3"
```

#### OTLP HTTP with TLS
```yaml
spec:
  ingestion:
    otlp:
      http:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
          # Optional: cipher suite configuration
          cipherSuites:
            - "TLS_AES_256_GCM_SHA384"
            - "TLS_CHACHA20_POLY1305_SHA256"
```

### 2. **Jaeger Receivers with TLS**

#### gRPC Jaeger Receiver
```yaml
spec:
  ingestion:
    jaeger:
      grpc:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
      thriftHttp:
        enabled: true
        tls:
          enabled: true
          certName: custom-cert
```

#### Binary and Compact Protocols
```yaml
spec:
  ingestion:
    jaeger:
      thriftBinary:
        enabled: true
        # UDP protocol - no TLS support
      thriftCompact:
        enabled: true
        # UDP protocol - no TLS support
```

### 3. **Advanced TLS Configuration**

#### Certificate Rotation Support
```yaml
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          certName: custom-cert
          # Operator watches for certificate updates
          autoReload: true
```

#### Client Authentication (Optional)
```yaml
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          certName: custom-cert
          caName: custom-ca          # Enable client cert validation
          clientAuthRequired: true   # Require client certificates
```

## Security Validation and Testing

### 1. **TLS Connection Testing**

#### Test gRPC TLS Connection
```bash
# Test TLS handshake for gRPC
openssl s_client -connect tempo-simplest:4317 \
  -CAfile ca.crt \
  -verify_return_error

# Test using grpcurl with TLS
grpcurl -insecure \
  -cacert ca.crt \
  tempo-simplest:4317 \
  list
```

#### Test HTTP TLS Connection
```bash
# Test HTTPS connection
curl -v https://tempo-simplest:4318/v1/traces \
  --cacert ca.crt \
  -H "Content-Type: application/json" \
  -X POST \
  -d '{}'

# Test certificate validity
openssl s_client -connect tempo-simplest:4318 \
  -CAfile ca.crt \
  -servername tempo-simplest
```

### 2. **Certificate Validation**

#### Check Certificate Details
```bash
# Extract certificate from secret
kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d > server.crt

# Verify certificate details
openssl x509 -in server.crt -text -noout

# Check certificate chain
openssl verify -CAfile ca.crt server.crt

# Verify Subject Alternative Names
openssl x509 -in server.crt -noout -ext subjectAltName
```

#### Monitor Certificate Expiration
```bash
# Check certificate expiration
kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | \
  openssl x509 -noout -dates

# Set up certificate expiration monitoring
kubectl create job cert-monitor --image=alpine/openssl -- \
  sh -c "echo 'Certificate expires:'; openssl x509 -noout -enddate -in /etc/certs/tls.crt"
```

### 3. **Performance Impact Assessment**

#### TLS Overhead Monitoring
```bash
# Monitor TLS handshake performance
kubectl exec tempo-simplest-0 -- ss -i | grep :4317

# Check TLS-related metrics
kubectl port-forward svc/tempo-simplest 3200:3200 &
curl http://localhost:3200/metrics | grep -E "(tls|ssl|handshake)"

# Monitor CPU usage impact
kubectl top pod tempo-simplest-0
```

## Troubleshooting TLS Issues

### 1. **Certificate Problems**

#### Certificate Mounting Issues
```bash
# Check certificate availability in pod
kubectl exec tempo-simplest-0 -- ls -la /etc/tls/

# Verify certificate content
kubectl exec tempo-simplest-0 -- cat /etc/tls/tls.crt

# Check certificate permissions
kubectl exec tempo-simplest-0 -- stat /etc/tls/tls.crt
```

#### Certificate Validation Errors
```bash
# Check Tempo logs for TLS errors
kubectl logs tempo-simplest-0 | grep -i "tls\|certificate\|ssl"

# Verify certificate chain
kubectl exec tempo-simplest-0 -- openssl verify -CAfile /etc/ca/service-ca.crt /etc/tls/tls.crt
```

### 2. **TLS Connection Issues**

#### Connection Refused or Reset
```bash
# Check if TLS ports are listening
kubectl exec tempo-simplest-0 -- netstat -ln | grep -E "4317|4318"

# Test local TLS connectivity
kubectl exec tempo-simplest-0 -- openssl s_client -connect localhost:4317

# Check service endpoints
kubectl get endpoints tempo-simplest
```

#### TLS Handshake Failures
```bash
# Enable TLS debugging in collector
kubectl patch opentelemetrycollector opentelemetry --type='merge' -p='
spec:
  config:
    service:
      telemetry:
        logs:
          level: "DEBUG"'

# Check collector logs for TLS errors
kubectl logs deployment/opentelemetry-collector | grep -i "tls\|handshake"
```

### 3. **Configuration Issues**

#### Invalid TLS Configuration
```bash
# Validate TempoMonolithic TLS config
kubectl get tempomonolithic simplest -o yaml | yq '.spec.ingestion.otlp'

# Check generated Tempo configuration
kubectl get configmap tempo-simplest-config -o jsonpath='{.data.tempo\.yaml}' | yq '.server'

# Verify TLS certificate mounting
kubectl describe pod tempo-simplest-0 | grep -A10 "Volumes:"
```

## Production Considerations

### 1. **Certificate Management**
- Implement automated certificate renewal (cert-manager)
- Use proper certificate validation and monitoring
- Plan for certificate rotation procedures
- Maintain certificate backup and recovery

### 2. **Performance Optimization**
```yaml
# Optimized TLS configuration for production
spec:
  ingestion:
    otlp:
      grpc:
        tls:
          enabled: true
          certName: custom-cert
          minVersion: "1.2"
          cipherSuites:
            - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
            - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
          sessionTickets: false      # Disable for better security
```

### 3. **Security Hardening**
- Disable weak cipher suites and protocols
- Implement proper key management
- Use Hardware Security Modules (HSM) for certificate storage
- Regular security assessments and penetration testing

### 4. **Monitoring and Alerting**
- Set up alerts for TLS handshake failures
- Monitor certificate expiration dates
- Track TLS connection metrics
- Implement security incident response procedures

## Cipher Suite and Protocol Configuration

### 1. **TLS Version Control**
```yaml
# Restrict to modern TLS versions
spec:
  extraConfig:
    tempo:
      server:
        grpc_tls_config:
          min_version: VersionTLS12
          max_version: VersionTLS13
```

### 2. **Cipher Suite Selection**
```yaml
# Modern cipher suite configuration
spec:
  extraConfig:
    tempo:
      server:
        grpc_tls_config:
          cipher_suites:
            - TLS_AES_256_GCM_SHA384
            - TLS_CHACHA20_POLY1305_SHA256
            - TLS_AES_128_GCM_SHA256
```

## Related Configurations

- [mTLS Ingestion Security](../monolithic-ingestion-mtls/README.md) - Mutual TLS authentication
- [TempoStack TLS](../../e2e-openshift/tls-singletenant/README.md) - Distributed TLS setup
- [Basic TempoMonolithic](../monolithic-memory/README.md) - Non-TLS configuration

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/monolithic-receivers-tls
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test includes specific timing (20s wait) and detailed logging to ensure proper TLS handshake completion before trace generation begins.

