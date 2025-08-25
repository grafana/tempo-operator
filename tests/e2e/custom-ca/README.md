# TempoStack with Custom Certificate Authority (CA) for Storage TLS

This test validates TempoStack deployment with a custom Certificate Authority (CA) for secure TLS communication to object storage. It demonstrates how to configure TempoStack to trust a custom CA certificate when connecting to TLS-enabled storage backends like MinIO.

## Test Overview

### Purpose
- **Custom CA Integration**: Tests TempoStack's ability to use custom Certificate Authorities for storage connections
- **Storage TLS Security**: Validates secure communication between TempoStack and TLS-enabled storage backends
- **Certificate Management**: Demonstrates proper handling of custom certificates in Kubernetes environments
- **Enterprise Security**: Shows how to integrate with enterprise PKI infrastructures

### Components
- **TempoStack**: Distributed Tempo deployment configured with custom CA for storage TLS
- **MinIO with TLS**: S3-compatible storage server secured with custom certificates
- **Custom CA Certificate**: Self-signed Certificate Authority for test environment
- **TLS Certificate**: Server certificate signed by the custom CA for MinIO

## Security Architecture

```
[TempoStack Components]
        ↓ (TLS with Custom CA validation)
[MinIO Storage Server]
        ↓ (Encrypted Storage)
[Trace Data Persistence]
```

## Certificate Generation

The test uses pre-generated certificates created with these commands:

### 1. Create Custom Certificate Authority
```bash
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout ca.key -out ca.crt -subj '/CN=MyDemoCA'
```

### 2. Generate Server Certificate Signed by CA
```bash
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout cert.key -out cert.crt -CA ca.crt -CAkey ca.key \
  -subj "/CN=minio" -addext "subjectAltName=DNS:minio"
```

## Deployment Steps

### 1. Deploy MinIO with TLS Certificates
```bash
kubectl apply -f 00-install-storage.yaml
```

The [`00-install-storage.yaml`](00-install-storage.yaml) file creates:
- **minio-cert Secret**: Contains TLS private key and certificate for MinIO server
- **MinIO Deployment**: Configured to use TLS certificates from mounted secret
- **MinIO Service**: Exposes MinIO on port 9000 for TempoStack access
- **minio Secret**: S3 credentials with HTTPS endpoint for TempoStack

Key MinIO TLS configuration:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: minio-cert
data:
  private.key: <base64-encoded-private-key>
  public.crt: <base64-encoded-certificate>
---
# MinIO deployment with TLS certificate mounting
spec:
  template:
    spec:
      containers:
      - name: minio
        command:
        - /bin/sh
        - -c
        - |
          mkdir -p /storage/tempo && \
          minio server --certs-dir=/root/.minio/certs /storage
        volumeMounts:
        - name: cert
          mountPath: /root/.minio/certs
      volumes:
      - name: cert
        secret:
          secretName: minio-cert
```

### 2. Deploy TempoStack with Custom CA
```bash
kubectl apply -f 01-install-tempo.yaml
```

Key configuration from [`01-install-tempo.yaml`](01-install-tempo.yaml):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-ca
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    <custom-ca-certificate-content>
    -----END CERTIFICATE-----
---
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: minio
      type: s3
    tls:
      enabled: true
      caName: custom-ca  # References the ConfigMap with CA certificate
  storageSize: 200M
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
```

### 3. Generate Test Traces
```bash
kubectl apply -f 02-generate-traces.yaml
```

The trace generation from [`02-generate-traces.yaml`](02-generate-traces.yaml):
```yaml
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
```

### 4. Verify Traces
```bash
kubectl apply -f 03-verify-traces.yaml
```

## Key Features Tested

### Custom CA Certificate Management
- ✅ ConfigMap-based CA certificate storage
- ✅ TempoStack CA certificate reference configuration
- ✅ TLS certificate validation against custom CA
- ✅ Secure storage communication with custom PKI

### TLS Storage Configuration
- ✅ MinIO server with TLS enabled using custom certificates
- ✅ S3-compatible storage with HTTPS endpoints
- ✅ Certificate mounting and MinIO TLS configuration
- ✅ TempoStack TLS client configuration

### End-to-End Security Validation
- ✅ Encrypted communication between TempoStack and storage
- ✅ Custom CA trust chain validation
- ✅ Trace ingestion and storage with TLS security
- ✅ Query functionality through encrypted storage connections

### Certificate Authority Integration
- ✅ Self-signed CA certificate creation and deployment
- ✅ Server certificate generation signed by custom CA
- ✅ Subject Alternative Name (SAN) configuration for DNS validation
- ✅ Long-term certificate validity (10 years for testing)

## TLS Configuration Details

### MinIO TLS Setup
- **Certificate Location**: `/root/.minio/certs` in MinIO container
- **Private Key**: `private.key` file mounted from Kubernetes secret
- **Public Certificate**: `public.crt` file mounted from Kubernetes secret
- **CA Validation**: MinIO validates client certificates against system CA store

### TempoStack TLS Configuration
- **CA Certificate Source**: ConfigMap named `custom-ca`
- **TLS Enabled**: `storage.tls.enabled: true`
- **CA Reference**: `storage.tls.caName: custom-ca`
- **Endpoint**: HTTPS URL `https://minio:9000` in storage secret

## Environment Requirements

### Kubernetes Prerequisites
- Kubernetes cluster with ConfigMap and Secret support
- Ability to mount volumes for certificate storage
- Network connectivity between TempoStack and MinIO pods

### Security Prerequisites
- Understanding of PKI and certificate management
- Knowledge of TLS/SSL certificate validation
- Familiarity with Kubernetes secret management

## Production Considerations

### Certificate Management
- **Expiration Monitoring**: Implement certificate expiration monitoring
- **Automated Renewal**: Set up certificate rotation processes
- **CA Security**: Protect CA private keys with appropriate access controls
- **Certificate Validation**: Ensure proper SAN and CN configuration

### Enterprise Integration
- **Corporate PKI**: Replace self-signed CA with enterprise Certificate Authority
- **Certificate Stores**: Integrate with enterprise certificate management systems
- **Compliance**: Ensure certificate practices meet regulatory requirements
- **Monitoring**: Implement TLS connection monitoring and alerting

## Troubleshooting

### Common Issues

**TLS Certificate Validation Failures**:
- Verify CA certificate is correctly formatted in ConfigMap
- Check that server certificate is signed by the specified CA
- Ensure SAN includes the correct DNS name (minio)
- Validate certificate expiration dates

**MinIO TLS Configuration Issues**:
- Confirm certificates are properly mounted in MinIO container
- Check MinIO logs for TLS initialization errors
- Verify certificate file permissions and accessibility
- Ensure private key and certificate match

**TempoStack Storage Connection Problems**:
- Validate storage secret contains HTTPS endpoint URL
- Check TempoStack pod logs for TLS handshake errors
- Verify CA certificate is readable from ConfigMap
- Ensure network connectivity to MinIO service

**Certificate Generation Problems**:
- Verify OpenSSL commands execute without errors
- Check that CA private key has proper permissions
- Ensure SAN extension includes correct DNS names
- Validate certificate chain completeness

## Security Best Practices

### Certificate Security
- **Private Key Protection**: Secure storage of private keys
- **Limited Validity**: Use shorter certificate lifetimes in production
- **Strong Algorithms**: Use RSA 4096-bit or ECDSA P-384 keys
- **Proper SAN Configuration**: Include all required DNS names and IPs

### Kubernetes Security
- **Secret Access**: Limit access to certificate secrets
- **RBAC**: Implement proper role-based access controls
- **Network Policies**: Restrict network access to storage services
- **Pod Security**: Use security contexts and non-root containers

## Alternative Approaches

### cert-manager Integration
For production environments, consider using cert-manager:
```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: minio-tls
spec:
  secretName: minio-cert
  issuerRef:
    name: ca-issuer
    kind: ClusterIssuer
  dnsNames:
  - minio
```

### External Certificate Sources
- **Vault Integration**: Use HashiCorp Vault for certificate management
- **Cloud PKI**: Leverage cloud provider certificate services
- **External Secrets**: Integrate with external secret management systems

This test demonstrates the complete workflow for securing TempoStack storage communications using custom Certificate Authorities, providing a foundation for enterprise PKI integration and secure observability deployments.