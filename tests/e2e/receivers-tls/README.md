# TLS-Secured Receivers Configuration

This test demonstrates how to configure TempoStack with TLS-secured receivers for secure trace ingestion. Unlike the mutual TLS (mTLS) configuration, this setup focuses on server-side TLS encryption for the distributor endpoints while still maintaining secure communication.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  OpenTelemetry  │───▶│   TLS Channel    │───▶│   TempoStack    │
│  Collector      │    │   (Port 4317)    │    │   Distributor   │
│  (Client)       │    │                  │    │   (Server)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         │                        │                       ▼
         ▼                        ▼              ┌─────────────────┐
┌─────────────────┐    ┌──────────────────┐    │   Trace Storage │
│  Client Cert    │    │  Server CA Cert  │    │   (MinIO S3)    │
│  (Optional)     │    │  (Validation)    │    └─────────────────┘
└─────────────────┘    └──────────────────┘
```

## Security Features

### Server-Side TLS
- **Encryption**: All trace data encrypted in transit
- **Certificate Authority**: Custom CA certificate for server validation
- **Protocol**: TLS 1.2+ for OTLP gRPC and HTTP endpoints
- **Port Security**: Secured ports 4317 (gRPC) and 4318 (HTTP)

### Certificate Management
- **Server Certificate**: TempoStack distributor presents TLS certificate
- **CA Validation**: Clients validate server certificate against custom CA
- **Certificate Storage**: Certificates stored as Kubernetes secrets
- **Rotation**: Supports certificate rotation without service interruption

## Test Components

### TempoStack with TLS Configuration
- **File**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)
- **TLS Config**: Distributor TLS enabled with custom certificate
- **Certificate**: Custom server certificate named `custom-cert`
- **CA Bundle**: Custom Certificate Authority for client validation

### OpenTelemetry Collector Configuration
- **File**: [`02-install-otel.yaml`](./02-install-otel.yaml)
- **Client Setup**: Configured to connect to TLS-secured TempoStack
- **CA Validation**: Uses custom CA certificate to validate server
- **Protocols**: Both gRPC (4317) and HTTP (4318) receivers enabled

### Certificate Components
The configuration includes several certificate-related components:

1. **Custom CA Certificate** (ConfigMap)
   - Provides root certificate for client validation
   - Mounted at `/var/run/tls/receiver/ca/service-ca.crt`

2. **Server Certificate** (Secret)
   - Contains TLS certificate and private key for TempoStack
   - Base64-encoded certificate and key data
   - Referenced by `certName: custom-cert` in TempoStack

3. **Client Certificate** (Secret) 
   - Optional client certificate for the OpenTelemetry Collector
   - Mounted at `/var/run/tls/receiver/cert/`

## Quick Start

### Prerequisites
- Kubernetes cluster with Tempo Operator
- OpenTelemetry Operator installed
- MinIO or S3-compatible storage

### Step-by-Step Deployment

1. **Install Storage Backend**
   ```bash
   # Deploy MinIO for trace storage
   kubectl apply -f 00-install-storage.yaml
   kubectl wait --for=condition=ready pod -l app=minio --timeout=300s
   ```

2. **Deploy TempoStack with TLS**
   ```bash
   # Create TempoStack with TLS-enabled distributor
   kubectl apply -f 01-install-tempo.yaml
   kubectl wait --for=condition=ready tempostack simplest --timeout=300s
   ```

3. **Deploy OpenTelemetry Collector**
   ```bash
   # Deploy collector configured for TLS connection
   kubectl apply -f 02-install-otel.yaml
   kubectl wait --for=condition=ready opentelemetrycollector opentelemetry --timeout=300s
   ```

4. **Generate Test Traces**
   ```bash
   # Send traces through the TLS-secured pipeline
   kubectl apply -f 03-generate-traces.yaml
   ```

5. **Verify Trace Collection**
   ```bash
   # Confirm traces are successfully stored
   kubectl apply -f 04-verify-traces.yaml
   ```

## TLS Configuration Details

### TempoStack TLS Settings
```yaml
template:
  distributor:
    tls:
      enabled: true
      certName: custom-cert  # References the TLS secret
```

### OpenTelemetry Collector TLS Settings
```yaml
exporters:
  otlp:
    endpoint: tempo-simplest-distributor:4317
    tls:
      insecure: false  # Enforce TLS validation
      ca_file: /var/run/tls/receiver/ca/service-ca.crt  # CA certificate path
```

### Certificate Volume Mounts
- **CA Certificate**: `/var/run/tls/receiver/ca/service-ca.crt`
- **Client Cert**: `/var/run/tls/receiver/cert/tls.crt`
- **Client Key**: `/var/run/tls/receiver/cert/tls.key`

## Testing Procedure

The complete test is defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml) and follows these steps:

1. **Storage Setup**: Deploy MinIO object storage
2. **TLS TempoStack**: Create TempoStack with TLS-enabled distributor
3. **TLS Collector**: Deploy OpenTelemetry Collector with TLS client configuration
4. **Trace Generation**: Send test traces through the secured pipeline
5. **Verification**: Confirm traces are encrypted in transit and stored correctly

## Security Validation

### Connection Security
```bash
# Verify TLS is enabled on distributor
kubectl get svc tempo-simplest-distributor -o yaml

# Check certificate configuration
kubectl get secret custom-cert -o yaml

# Validate CA configmap
kubectl get configmap custom-ca -o yaml
```

### OpenTelemetry Collector Status
```bash
# Check collector configuration
kubectl get opentelemetrycollector opentelemetry -o yaml

# View collector logs for TLS connection
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector
```

## Production Considerations

### Certificate Management
- **Expiration**: Monitor certificate expiration dates
- **Rotation**: Plan regular certificate rotation strategy
- **Storage**: Secure certificate storage and access controls
- **Backup**: Maintain secure backups of private keys

### Performance Impact
- **CPU Overhead**: TLS encryption adds ~2-5% CPU overhead
- **Latency**: Minimal latency increase (<1ms) for TLS handshake
- **Throughput**: Negligible impact on trace throughput

### Security Best Practices
- **Strong Ciphers**: Use modern TLS cipher suites
- **Certificate Validation**: Always validate server certificates
- **Key Management**: Protect private keys with appropriate RBAC
- **Network Policies**: Restrict network access to TLS endpoints

## Troubleshooting

### Common TLS Issues

1. **Certificate Validation Failures**
   ```bash
   # Check certificate details
   kubectl get secret custom-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
   
   # Verify CA certificate
   kubectl get configmap custom-ca -o jsonpath='{.data.service-ca\.crt}'
   ```

2. **Connection Refused Errors**
   ```bash
   # Test TLS connectivity
   kubectl exec -it deployment/opentelemetry-collector -- openssl s_client -connect tempo-simplest-distributor:4317
   
   # Check distributor TLS configuration
   kubectl describe tempostack simplest
   ```

3. **Collector Not Sending Traces**
   ```bash
   # Check collector logs for TLS errors
   kubectl logs -l app.kubernetes.io/name=opentelemetry-collector | grep -i tls
   
   # Verify collector configuration
   kubectl get opentelemetrycollector opentelemetry -o jsonpath='{.spec.config}'
   ```

### Debug Commands
```bash
# Check TempoStack TLS status
kubectl get tempostack simplest -o jsonpath='{.status.conditions}'

# Verify certificate mounting
kubectl describe pod -l app.kubernetes.io/name=opentelemetry-collector

# Test trace pipeline
kubectl port-forward svc/opentelemetry-collector 4318:4318
curl -X POST http://localhost:4318/v1/traces -H "Content-Type: application/json" -d '{...}'
```

## Comparison with mTLS

| Feature | TLS (This Test) | Mutual TLS |
|---------|----------------|-------------|
| Server Auth | ✅ Server certificate | ✅ Server certificate |
| Client Auth | ❌ No client certificate required | ✅ Client certificate required |
| Complexity | Lower - simpler setup | Higher - requires client certs |
| Security | Good - encrypted transport | Excellent - mutual authentication |
| Use Case | Internal trusted networks | Zero-trust environments |

## Related Resources

- [Mutual TLS Configuration](../receivers-mtls/README.md)
- [TempoStack Security Guide](../../../docs/security-configuration.md)
- [OpenTelemetry TLS Documentation](https://opentelemetry.io/docs/specs/otel/configuration/tls/)
- [Kubernetes TLS Secret Management](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets)