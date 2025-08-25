# TempoStack with Mutual TLS (mTLS) for Receivers

This configuration blueprint demonstrates how to deploy TempoStack with mutual TLS (mTLS) authentication for secure trace ingestion. This setup showcases enterprise-grade security where both client and server authenticate each other using certificates, providing the highest level of transport security for observability data.

## Overview

This test validates a secure observability stack featuring:
- **Mutual TLS Authentication**: Both client and server present certificates for verification
- **Custom Certificate Authority**: Self-managed PKI infrastructure for certificate validation
- **Client Certificate Management**: OpenTelemetry collector configured with client certificates
- **Server Certificate Validation**: TempoStack distributor validates client certificates
- **End-to-End Encryption**: Complete protection of trace data in transit

## Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OTel Collector  │───▶│    TempoStack        │───▶│ MinIO Storage   │
│ ┌─────────────┐ │    │ ┌─────────────────┐  │    │ (S3 Compatible) │
│ │ Client Cert │ │    │ │ Server Cert +   │  │    └─────────────────┘
│ │ Client Key  │ │    │ │ Client CA       │  │
│ │ Server CA   │ │    │ │ Mutual Auth     │  │
│ └─────────────┘ │    │ └─────────────────┘  │
└─────────────────┘    └──────────────────────┘
          │                        │
          │ ┌──────────────────────┴─────────────────────┐
          │ │         Custom Certificate Authority       │
          └▶│ ┌─────────────────┐ ┌─────────────────┐    │
            │ │ Client Certs    │ │ Server Certs    │    │
            │ │ (Collector)     │ │ (Distributor)   │    │
            │ └─────────────────┘ └─────────────────┘    │
            └────────────────────────────────────────────┘
```

## Prerequisites

- Kubernetes cluster with sufficient resources
- Tempo Operator installed
- OpenTelemetry Operator installed
- Understanding of PKI and certificate management
- `kubectl` CLI access

## Step-by-Step Deployment

### Step 1: Deploy MinIO Object Storage

Create the storage backend:

```bash
# Apply storage configuration
kubectl apply -f - <<EOF
# Standard MinIO deployment with PVC, service, and secret
# Reference: 00-install-storage.yaml
EOF
```

**Reference**: [`00-install-storage.yaml`](./00-install-storage.yaml)

### Step 2: Create Custom Certificate Authority

Set up the custom CA and certificates for mTLS:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-ca
data:
  service-ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIDWzCCAkOgAwIBAgIUJ714jTtYBKKKYtVCysJY+DqMNgowDQYJKoZIhvcNAQEL
    BQAwPDELMAkGA1UEBhMCWFgxFTATBgNVBAcMDERlZmF1bHQgQ2l0eTEWMBQGA1UE
    CgwNb2JzZXJ2YWJpbGl0eTAgFw0yMzExMDIwMzE1MzdaGA8yMDUxMDMxOTAzMTUz
    N1owPDELMAkGA1UEBhMCWFgxFTATBgNVBAcMDERlZmF1bHQgQ2l0eTEWMBQGA1UE
    CgwNb2JzZXJ2YWJpbGl0eTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
    AKqdrAkgclDaho+NwCrdr4wuR1zgDZ71Gzmdjokkn0dBa8sUR69or25PfB3oAzs0
    J8i23lQB9Ny4jsDud8XoNkPpECTh1ddvqLj33Z3tacdZ82ESZ16HYdtDVEc2JnUZ
    GzmrR9WKWFJ/JFS3/Kp1CSiLVy8fmT6Xq3RShgv+cGJ7tTI+Y4g6It5gDCmT5sSA
    ZfubqGcbo9LLYeQuovEiSTUW4K+w0/3psBf5SmGH4srzICHejX4pSV3lMaB8rwOC
    zWex/vdCyinf3TfLlUb6euqRMFxiGZgaskgqG0xVshFqv3FoRZ3yAOM1YeIgDNyz
    gKObWefMzWmXLL8E+MLJMdUCAwEAAaNTMFEwHQYDVR0OBBYEFLFDtqWNRN2a32iX
    7Ralg/PfX32JMB8GA1UdIwQYMBaAFLFDtqWNRN2a32iX7Ralg/PfX32JMA8GA1Ud
    EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAHgbypkfsfucJMuyIG2xHnNr
    yJr14vbVETq7Rl4MlpWkrTMJieJb2egSkSxsFq25H8da6Rqkj+3me57zxYZTnrgG
    4xcdoVuX2Lm+pytX+SIMJkhY6J44uq43CBgJ0RPIwPAN1za2+VRnaIhc48m2cyxP
    xv0wyCCc0SRthqNLhUd9vSgp3NWIBT9Dl2d6RVORl87dV1sd5GG4BQMmvmBmM36/
    A1CVtDwDngOy1C9rjnWWHxQ/NzVYRKCzNUqiC+E5xtDpSfCCzNrWRvjwIdNveri9
    ga1x9xINK/3uKa5ZEcTg+PAVKI6dNsPvpolaH3zBNs4YaqXcQ21Ix29JuKIsNH8=
    -----END CERTIFICATE-----
---
apiVersion: v1
kind: Secret
metadata:
  name: custom-cert
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...  # Server certificate
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0t...  # Server private key
EOF
```

**Certificate Configuration Details**:
- `custom-ca`: ConfigMap containing the root CA certificate for validation
- `custom-cert`: Secret containing server certificate and private key for Tempo distributor
- **Certificate Subjects**: Must match expected DNS names and service endpoints

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Deploy TempoStack with mTLS

Create TempoStack configured for mutual TLS authentication:

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
  template:
    distributor:
      tls:
        enabled: true
        caName: custom-ca
        certName: custom-cert
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```

**Key mTLS Configuration Details**:

#### TLS Settings for Distributor
- `tls.enabled: true`: Enables TLS for the distributor component
- `caName: custom-ca`: References ConfigMap containing CA certificate for client validation
- `certName: custom-cert`: References Secret containing server certificate and key

#### Certificate Validation
- **Server Authentication**: Distributor presents server certificate to clients
- **Client Authentication**: Distributor validates client certificates against custom CA
- **Mutual Verification**: Both parties must present valid certificates

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 4: Create Client Certificates for Collector

Deploy OpenTelemetry collector with client certificates:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: opentelemetry-collector-cert
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...  # Client certificate
  tls.key: LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0t...  # Client private key
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
        endpoint: tempo-simplest-distributor:4317
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
          http:
            endpoint: 0.0.0.0:4318
    extensions:
      health_check:
        endpoint: 0.0.0.0:13133
    service:
      extensions: [health_check]
      pipelines:
        traces:
          exporters: [otlp, debug]
          receivers: [otlp]
EOF
```

**Key mTLS Configuration Details**:

#### Volume Mounts
- `/var/run/tls/receiver/ca/`: CA certificate for server validation
- `/var/run/tls/receiver/cert/`: Client certificate and private key

#### TLS Configuration
- `insecure: false`: Enforces TLS certificate validation
- `ca_file`: CA certificate to validate server certificate
- `cert_file`: Client certificate for mutual authentication
- `key_file`: Private key corresponding to client certificate

#### Certificate Management
- **Volume-based**: Certificates mounted as read-only volumes
- **File-based Access**: OTel collector reads certificates from filesystem
- **Secure Permissions**: Files mounted with appropriate read-only permissions

**Reference**: [`02-install-otel.yaml`](./02-install-otel.yaml)

### Step 5: Generate Sample Traces

Create traces using the mTLS-enabled collector:

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
        - --otlp-endpoint=opentelemetry-collector:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Configuration Notes**:
- Traces sent to collector (not directly to Tempo)
- Collector handles mTLS complexity with Tempo
- End-to-end security from collector to distributor

**Reference**: [`03-generate-traces.yaml`](./03-generate-traces.yaml)

### Step 6: Verify mTLS Trace Flow

Test that traces flow securely through the mTLS pipeline:

```bash
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
            http://tempo-simplest-query-frontend:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          
          echo "Successfully verified \$num_traces traces via mTLS pipeline"
      restartPolicy: Never
EOF
```

**Reference**: [`04-verify-traces.yaml`](./04-verify-traces.yaml)

## Key Features Demonstrated

### 1. **Mutual Authentication**
- **Bidirectional Verification**: Both client and server authenticate each other
- **Certificate-based Trust**: PKI infrastructure for identity verification
- **Zero-Trust Security**: No implicit trust between components
- **Identity Validation**: Cryptographic proof of component identity

### 2. **Custom PKI Management**
- **Self-Managed CA**: Custom certificate authority for organizational control
- **Certificate Lifecycle**: Creation, distribution, and rotation procedures
- **Trust Chain**: Hierarchical certificate validation
- **Security Isolation**: Separate PKI from external certificate authorities

### 3. **Transport Security**
- **Encryption in Transit**: All trace data encrypted during transmission
- **Perfect Forward Secrecy**: Session keys cannot compromise past sessions
- **Algorithm Selection**: Strong cryptographic algorithms and key sizes
- **Protocol Security**: TLS 1.2+ with secure cipher suites

### 4. **Enterprise Integration**
- **Certificate Management**: Integration with enterprise PKI systems
- **Compliance Ready**: Meets strict security compliance requirements
- **Audit Trail**: Comprehensive logging of certificate operations
- **Operational Security**: Secure key distribution and storage

## Certificate Management

### Certificate Generation

Example commands for generating mTLS certificates:

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -key ca.key -sha256 -subj "/C=XX/L=Default City/O=observability" -days 10000 -out ca.crt

# Generate server private key
openssl genrsa -out server.key 4096

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr -subj "/CN=tempo-simplest-distributor"

# Generate server certificate
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 10000 -sha256 -extensions v3_req

# Generate client private key
openssl genrsa -out client.key 4096

# Generate client certificate signing request
openssl req -new -key client.key -out client.csr -subj "/CN=opentelemetry-collector"

# Generate client certificate
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 10000 -sha256
```

### Certificate Rotation

Implement regular certificate rotation:

```bash
# Create new certificates
kubectl create secret tls new-custom-cert --cert=new-server.crt --key=new-server.key

# Update TempoStack configuration
kubectl patch tempostack simplest --type='merge' -p='{"spec":{"template":{"distributor":{"tls":{"certName":"new-custom-cert"}}}}}'

# Wait for rollout
kubectl rollout status deployment/tempo-simplest-distributor

# Update collector certificates
kubectl create secret tls new-collector-cert --cert=new-client.crt --key=new-client.key
kubectl patch opentelemetrycollector opentelemetry --type='merge' -p='{"spec":{"volumes":[{"name":"opentelemetry-collector-cert","secret":{"secretName":"new-collector-cert"}}]}}'
```

## Security Validation

### Certificate Verification

```bash
# Verify server certificate
openssl x509 -in server.crt -text -noout

# Verify certificate chain
openssl verify -CAfile ca.crt server.crt

# Check certificate expiration
openssl x509 -in server.crt -noout -dates

# Validate client certificate
openssl verify -CAfile ca.crt client.crt
```

### mTLS Connection Testing

```bash
# Test mTLS handshake
openssl s_client -connect tempo-simplest-distributor:4317 \
  -cert client.crt -key client.key -CAfile ca.crt

# Verify mutual authentication
kubectl exec deployment/opentelemetry-collector -- \
  openssl s_client -connect tempo-simplest-distributor:4317 \
  -cert /var/run/tls/receiver/cert/tls.crt \
  -key /var/run/tls/receiver/cert/tls.key \
  -CAfile /var/run/tls/receiver/ca/service-ca.crt
```

### Security Monitoring

```bash
# Monitor TLS handshake failures
kubectl logs -l app.kubernetes.io/component=distributor | grep -i "tls\|certificate\|handshake"

# Check certificate validation errors
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector | grep -i "certificate\|tls"

# Verify mTLS metrics
kubectl port-forward svc/tempo-simplest-distributor 3200:3200
curl http://localhost:3200/metrics | grep tls
```

## Troubleshooting

### Certificate Issues

```bash
# Check certificate validity
kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# Verify CA certificate
kubectl get configmap custom-ca -o jsonpath='{.data.service-ca\.crt}' | openssl x509 -text -noout

# Test certificate chain
kubectl exec deployment/opentelemetry-collector -- \
  openssl verify -CAfile /var/run/tls/receiver/ca/service-ca.crt /var/run/tls/receiver/cert/tls.crt
```

### TLS Handshake Failures

```bash
# Debug TLS connection
kubectl exec deployment/opentelemetry-collector -- \
  openssl s_client -connect tempo-simplest-distributor:4317 -debug

# Check certificate subject names
kubectl logs -l app.kubernetes.io/component=distributor | grep -i "certificate subject"

# Verify file permissions
kubectl exec deployment/opentelemetry-collector -- ls -la /var/run/tls/receiver/cert/
```

### Common mTLS Issues

1. **Certificate Subject Mismatch**:
   ```bash
   # Verify certificate CN matches service name
   kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -subject
   ```

2. **Expired Certificates**:
   ```bash
   # Check certificate expiration
   kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates
   ```

3. **CA Certificate Mismatch**:
   ```bash
   # Verify CA used to sign certificates
   kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -issuer
   ```

## Production Considerations

### 1. **Certificate Management**
- Implement automated certificate rotation
- Use enterprise PKI or certificate management systems
- Monitor certificate expiration dates
- Establish certificate revocation procedures

### 2. **Security Hardening**
- Use strong cryptographic algorithms (RSA 4096, ECDSA P-384)
- Implement certificate pinning for critical communications
- Regular security audits and penetration testing
- Secure storage of private keys

### 3. **Operational Excellence**
- Automate certificate deployment and rotation
- Monitor mTLS connection health and performance
- Implement comprehensive logging and alerting
- Document certificate management procedures

### 4. **Compliance and Governance**
- Meet regulatory requirements (SOX, PCI DSS, HIPAA)
- Implement certificate policy and procedures
- Regular compliance audits and assessments
- Document security controls and processes

## Related Configurations

- [TLS Single Tenant](../../e2e-openshift/tls-singletenant/README.md) - Basic TLS configuration
- [Basic TempoStack](../compatibility/README.md) - Non-TLS baseline
- [Gateway Authentication](../gateway/README.md) - Application-level authentication
- [Multi-tenant Security](../../e2e-openshift/multitenancy/README.md) - Tenant isolation patterns

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/receivers-mtls
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires custom PKI setup and demonstrates the highest level of transport security for Tempo deployments.