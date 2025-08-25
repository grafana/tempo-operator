# TempoStack Configuration Generation Tool

This configuration blueprint demonstrates how to use the Tempo Operator's built-in configuration generation capability to create complete Kubernetes manifests from minimal TempoStack specifications. This feature is essential for GitOps workflows, configuration templating, and CI/CD automation.

## Overview

This test validates the operator's configuration generation functionality:
- **Manifest Generation**: Generate complete Kubernetes manifests from minimal TempoStack config
- **Template Expansion**: Automatic expansion of operator-controlled defaults
- **Configuration Validation**: Ensure generated configs are deployable and functional
- **Development Workflow**: Support for offline configuration development and testing

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Minimal Config      │───▶│    Tempo Operator        │───▶│ Complete Manifests  │
│ - TempoStack CR     │    │    Manager Generate      │    │ - StatefulSets      │
│ - ProjectConfig     │    │    Command               │    │ - Deployments       │
│ - Storage Secret    │    └──────────────────────────┘    │ - Services          │
└─────────────────────┘                                    │ - ConfigMaps        │
                                                           │ - ServiceAccounts   │
┌─────────────────────┐    ┌──────────────────────────┐    └─────────────────────┘
│ Deployment Test     │◀───│    Generated Resources   │
│ - Apply manifests   │    │    Validation           │
│ - Verify readiness  │    │                         │
└─────────────────────┘    └──────────────────────────┘
```

## Prerequisites

- Kubernetes cluster with basic resources
- Tempo Operator binary (`bin/manager`) available
- `kubectl` CLI access
- Storage backend (MinIO) for testing generated config

## Step-by-Step Generation Process

### Step 1: Prepare Storage Configuration

Create the storage secret for the TempoStack:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
   name: minio-test
stringData:
  endpoint: http://minio.minio.svc:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF
```

**Reference**: [`00-storage-secret.yaml`](./00-storage-secret.yaml)

### Step 2: Define Project Configuration

Create the operator configuration file:

```yaml
apiVersion: config.tempo.grafana.com/v1alpha1
kind: ProjectConfig
distribution: community
health:
  healthProbeBindAddress: :8081
metrics:
  bindAddress: 127.0.0.1:8080
webhook:
  port: 9443
leaderElection:
  leaderElect: true
  resourceName: 8b886b0f.grafana.com
featureGates:
  openshift:
    openshiftRoute: false
    servingCertsService: false
  prometheusOperator: false
  httpEncryption: false
  grpcEncryption: false
  tlsProfile: Modern
  builtInCertManagement:
    enabled: false
    caValidity: 43830h
    caRefresh: 35064h
    certValidity: 2160h
    certRefresh: 1728h
  observability:
    metrics:
      createServiceMonitors: false
      createPrometheusRules: false
```

**Key Configuration Elements**:
- `distribution: community`: Target community distribution features
- `featureGates`: Disable OpenShift-specific features for community deployment
- `observability.metrics`: Disable Prometheus integration for basic setup
- `builtInCertManagement.enabled: false`: Use simple HTTP configuration

**Reference**: [`config.yaml`](./config.yaml)

### Step 3: Define Minimal TempoStack

Create a minimal TempoStack Custom Resource:

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: generated-tempo
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 1Gi
```

**Configuration Details**:
- **Minimal Specification**: Only essential fields defined
- **Storage Reference**: Links to external storage secret
- **Size Allocation**: 1Gi storage for demonstration
- **Default Values**: All other configurations use operator defaults

**Reference**: [`cr.yaml`](./cr.yaml)

### Step 4: Generate Complete Manifests

Use the Tempo Operator's generate command to create full Kubernetes manifests:

```bash
# Set required environment variables for image references
export RELATED_IMAGE_TEMPO=docker.io/grafana/tempo:2.7.0
export RELATED_IMAGE_TEMPO_QUERY=docker.io/grafana/tempo-query:2.7.0  
export RELATED_IMAGE_TEMPO_GATEWAY=quay.io/observatorium/api:main-2024-11-05-28e4c83
export RELATED_IMAGE_TEMPO_GATEWAY_OPA=quay.io/observatorium/opa-openshift:main-2024-10-09-7237863

# Generate complete manifests
../../../bin/manager generate \
  --config config.yaml \
  --cr cr.yaml \
  --output generated.yaml
```

**Command Parameters**:
- `--config`: ProjectConfig file defining operator behavior
- `--cr`: TempoStack Custom Resource with minimal specification
- `--output`: Output file for generated manifests

**Environment Variables**:
- Container image references for all Tempo components
- Override default images for specific deployment requirements

### Step 5: Deploy Generated Configuration

Apply the generated manifests to validate functionality:

```bash
kubectl apply -n $NAMESPACE -f generated.yaml
```

### Step 6: Verify Generated Resources

Validate that all generated components are ready:

```bash
# Check StatefulSet readiness (Ingester)
kubectl get statefulset tempo-generated-tempo-ingester -o jsonpath='{.status.readyReplicas}'
# Should return: 1

# Check Deployment readiness (all components)
kubectl get deployment -l app.kubernetes.io/instance=generated-tempo -o jsonpath='{.items[*].status.readyReplicas}'

# Verify Services are created
kubectl get services -l app.kubernetes.io/instance=generated-tempo
```

**Expected Resources**:
- **ConfigMap**: `tempo-generated-tempo` (Tempo configuration)
- **ServiceAccount**: `tempo-generated-tempo`
- **StatefulSet**: `tempo-generated-tempo-ingester` (data persistence)
- **Deployments**: distributor, query-frontend, querier, compactor
- **Services**: Component services + gossip ring + query frontend discovery

**Reference**: [`01-assert.yaml`](./01-assert.yaml)

## Generated Manifest Analysis

### ConfigMap Structure

The generated configuration includes:

#### Main Tempo Configuration (`tempo.yaml`)
```yaml
# Distributor configuration with all receiver protocols
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268
        thrift_binary:
          endpoint: 0.0.0.0:6832
        thrift_compact:
          endpoint: 0.0.0.0:6831
        grpc:
          endpoint: 0.0.0.0:14250
    zipkin:
      endpoint: 0.0.0.0:9411
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318

# Storage configuration with S3 backend
storage:
  trace:
    backend: s3
    blocklist_poll: 5m
```

#### Query Frontend Configuration (`tempo-query-frontend.yaml`)
- Specialized configuration for query frontend component
- Search and query optimization settings
- Component-specific performance tuning

### Component Deployments

#### Ingester (StatefulSet)
- **Persistent Storage**: 1Gi PVC per replica using `volumeClaimTemplates`
- **WAL Storage**: Write-ahead log for data durability  
- **Memberlist Integration**: Gossip protocol for service discovery
- **Environment Variables**: S3 credentials injection

#### Distributor (Deployment)
- **Multi-Protocol Support**: OTLP (gRPC/HTTP), Jaeger, Zipkin
- **Load Balancing**: Handles trace ingestion distribution
- **Port Configuration**: All standard trace receiver ports

#### Query Frontend (Deployment)
- **Query Coordination**: Manages query distribution to queriers
- **Cache Integration**: Query result caching capability
- **API Exposure**: TraceQL and Jaeger API endpoints

#### Querier (Deployment)
- **Trace Retrieval**: Reads traces from storage backend
- **Search Capabilities**: Supports complex trace queries
- **Parallel Processing**: Concurrent query execution

#### Compactor (Deployment)
- **Block Management**: Compacts and optimizes stored trace blocks
- **Retention Handling**: Implements configured retention policies
- **Background Processing**: Continuous optimization of storage

### Service Discovery

#### Gossip Ring Service (`tempo-generated-tempo-gossip-ring`)
- **Headless Service**: `ClusterIP: None` for direct pod communication
- **Memberlist Protocol**: Enables ring membership and failure detection
- **publishNotReadyAddresses**: Includes all pods in service discovery

#### Query Frontend Discovery (`tempo-generated-tempo-query-frontend-discovery`)
- **Load Balancing**: Distributes queries across querier instances
- **Health-aware Routing**: Only routes to ready queriers
- **gRPC Load Balancing**: Supports client-side load balancing

## Key Features Demonstrated

### 1. **Automatic Configuration Expansion**
- Minimal input configuration expands to complete deployment manifests
- Operator applies sensible defaults for all unspecified configurations
- Environment-specific optimizations automatically applied

### 2. **Multi-Protocol Trace Ingestion**
- **OTLP**: OpenTelemetry standard (gRPC/HTTP)
- **Jaeger**: Legacy Jaeger formats (thrift variants + gRPC)
- **Zipkin**: Zipkin HTTP format support
- **Port Management**: Automatic service configuration for all protocols

### 3. **Distributed Architecture Generation**
- **Component Separation**: Each Tempo component gets dedicated deployment
- **Service Mesh Ready**: Proper service discovery and load balancing
- **Scalability**: Each component can be independently scaled

### 4. **Storage Integration**
- **Secret Management**: Secure credential injection via environment variables
- **Backend Flexibility**: Generated config supports various storage types
- **Persistence**: StatefulSet for ingester with persistent volumes

## Use Cases

### 1. **GitOps Workflows**
- Generate manifests for version control
- Enable declarative infrastructure management
- Support automated deployment pipelines

### 2. **Configuration Templating**
```bash
# Generate configs for different environments
manager generate --config config-dev.yaml --cr cr-dev.yaml --output dev.yaml
manager generate --config config-prod.yaml --cr cr-prod.yaml --output prod.yaml
```

### 3. **Offline Development**
- Generate manifests without running operator
- Test configuration changes before deployment
- Validate resource requirements

### 4. **CI/CD Integration**
```bash
# CI pipeline example
./manager generate --config $CONFIG_FILE --cr $CR_FILE --output deployment.yaml
kubectl apply -f deployment.yaml --dry-run=client  # Validate syntax
kubectl apply -f deployment.yaml
```

## Configuration Customization

### Environment-Specific Images

```bash
# Production images
export RELATED_IMAGE_TEMPO=registry.company.com/tempo:2.7.0
export RELATED_IMAGE_TEMPO_QUERY=registry.company.com/tempo-query:2.7.0

# Generate with custom images
./manager generate --config config.yaml --cr cr.yaml --output custom.yaml
```

### Feature Gate Customization

```yaml
# config.yaml with OpenShift features
featureGates:
  openshift:
    openshiftRoute: true
    servingCertsService: true
  prometheusOperator: true
  httpEncryption: true
  grpcEncryption: true
```

### Advanced TempoStack Configuration

```yaml
# cr.yaml with additional specifications
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: generated-tempo
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 10Gi
  resources:
    total:
      limits:
        memory: 4Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
        ingress:
          type: ingress
```

## Troubleshooting

### Generation Issues

1. **Missing Images Error**:
   ```bash
   # Ensure all RELATED_IMAGE_* variables are set
   env | grep RELATED_IMAGE
   ```

2. **Invalid Configuration**:
   ```bash
   # Validate YAML syntax
   yamllint config.yaml cr.yaml
   ```

3. **Permission Issues**:
   ```bash
   # Check binary permissions
   ls -la ../../../bin/manager
   chmod +x ../../../bin/manager
   ```

### Deployment Issues

1. **Resource Not Ready**:
   ```bash
   # Check pod status
   kubectl describe pod -l app.kubernetes.io/instance=generated-tempo
   
   # View component logs
   kubectl logs -l app.kubernetes.io/component=distributor
   ```

2. **Storage Connection Issues**:
   ```bash
   # Verify secret exists
   kubectl get secret minio-test -o yaml
   
   # Test storage connectivity
   kubectl exec deployment/tempo-generated-tempo-distributor -- \
     curl -v http://minio.minio.svc:9000/
   ```

## Production Considerations

### 1. **Image Management**
- Use specific image tags, not `latest`
- Maintain consistent image versions across components
- Implement image scanning and security policies

### 2. **Configuration Management**
- Version control all configuration files
- Use configuration validation in CI pipelines
- Implement change approval processes

### 3. **Resource Planning**
- Review generated resource requests/limits
- Plan for storage growth and retention
- Monitor component resource utilization

### 4. **Security**
- Review generated RBAC permissions
- Validate network security policies
- Implement proper secret management

## Related Configurations

- [TempoStack Compatibility](../compatibility/README.md) - Full deployment with persistence
- [Monolithic Memory](../monolithic-memory/README.md) - Simple single-pod deployment
- [TLS Configuration](../tls-singletenant/README.md) - Secure communications

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/generate
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

