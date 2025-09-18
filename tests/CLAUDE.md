# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Test Framework and Commands

This test suite uses **Chainsaw** (Kubernetes native declarative end-to-end testing) as the primary test framework. Tests are organized as YAML-based test definitions with apply/assert patterns.

### Running Tests

```bash
# Run all e2e tests
chainsaw test --test-dir ./tests/e2e --config .chainsaw.yaml

# Run OpenShift-specific tests
chainsaw test --test-dir ./tests/e2e-openshift --config .chainsaw-openshift.yaml

# Run upgrade tests
chainsaw test --test-dir ./tests/e2e-upgrade --config .chainsaw-upgrade.yaml

# Run specific test
chainsaw test --test-dir ./tests/e2e/compatibility

# Run with custom values (for upgrade tests)
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-upgrade --values - <<EOF
upgrade_fbc_image: brew.registry.redhat.io/rh-osbs/iib:988093
upgrade_operator_version: 0.16.0
upgrade_tempo_version: 2.7.2
upgrade_operator_csv_name: tempo-operator.v0.16.0-1
EOF
```

### Docker Test Environment

Build and run tests in containerized environment:
```bash
# Build test image
docker build -f tests/Dockerfile -t tempo-operator-tests .

# Run tests in container
docker run --rm -v /path/to/kubeconfig:/kubeconfig tempo-operator-tests
```

## Test Architecture

### Test Categories

- **e2e/**: Core end-to-end tests (compatibility, custom-ca, gateway, monolithic variants, receivers)
- **e2e-openshift/**: OpenShift-specific tests (multitenancy, monitoring, RBAC, TLS, routes)
- **e2e-openshift-object-stores/**: Cloud provider object store integration (AWS STS, Azure WIF, GCP WIF)
- **e2e-openshift-ossm/**: OpenShift Service Mesh integration tests
- **e2e-openshift-serverless/**: Knative/Serverless integration tests
- **e2e-openshift-upgrade/**: OpenShift operator upgrade tests
- **e2e-upgrade/**: Generic upgrade tests
- **e2e-long-running/**: Long-duration tests (retention policies)
- **operator-metrics/**: Operator metrics validation

### Test Structure Pattern

Each test directory follows this pattern:
- `chainsaw-test.yaml`: Main test definition with steps
- `XX-install*.yaml`: Resource installation steps (numbered)
- `XX-assert*.yaml`: Assertion files to verify state
- `XX-generate-traces.yaml`: Trace generation jobs
- `XX-verify-traces*.yaml`: Trace verification steps
- `*.sh`: Shell scripts for complex setup/teardown

### Key Test Components

#### TempoStack vs TempoMonolithic
- **TempoStack**: Multi-component distributed deployment (querier, ingester, distributor, compactor)
- **TempoMonolithic**: Single-component deployment for simpler setups
- Both support multitenancy, RBAC, and various storage backends

#### Storage Backends Tested
- MinIO (local object storage)
- AWS S3 (with STS token authentication)
- Azure Blob Storage (with Workload Identity Federation)
- Google Cloud Storage (with Workload Identity Federation)
- IBM Cloud Object Storage

#### Authentication & Authorization
- RBAC tests validate namespace-level access control
- Static tenant configuration tests
- OpenShift RBAC integration
- Service account token authentication

## Common Development Tasks

### Adding New Tests

1. Create test directory under appropriate category (e2e/, e2e-openshift/, etc.)
2. Create `chainsaw-test.yaml` with test steps
3. Add installation YAML files (numbered sequence)
4. Add corresponding assertion files
5. Include trace generation and verification steps if applicable

### Test Debugging

```bash
# Run with verbose output
chainsaw test --test-dir ./tests/e2e/compatibility -v

# Skip cleanup for debugging
chainsaw test --test-dir ./tests/e2e/compatibility --skip-delete

# Run single step
chainsaw test --test-dir ./tests/e2e/compatibility --include-test-regex "step-01"
```

### Cloud Provider Setup

For object store tests, ensure cloud credentials are configured:
```bash
# AWS
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

# Azure
az login

# GCP
gcloud auth application-default login
```

## Configuration Blueprints

The tests directory includes comprehensive README files that serve as configuration blueprints for deploying Tempo observability stacks. These READMEs provide step-by-step instructions, architecture diagrams, and production considerations:

### Security & Authentication
- **[OIDC Authentication Gateway](./e2e/gateway/README.md)** - External OIDC provider integration, secure API access, token validation
- **[Mutual TLS (mTLS)](./e2e/receivers-mtls/README.md)** - Client certificate authentication, encrypted channels, PKI infrastructure
- **[TLS-Secured Receivers](./e2e/receivers-tls/README.md)** - Server-side TLS encryption, certificate management, secure ingestion
- **[OpenShift Multi-Tenancy with RBAC](./e2e-openshift/multitenancy-rbac/README.md)** - Namespace isolation, service account mapping, role-based access
- **[Single Tenant Authentication](./e2e-openshift/tempo-single-tenant-auth/README.md)** - OpenShift OAuth, SAR validation, simplified auth
- **[TLS Single Tenant](./e2e-openshift/tls-singletenant/README.md)** - OpenShift Service CA, automatic certificates, encrypted communications

### Deployment Models
- **[TempoStack with Object Storage](./e2e/compatibility/README.md)** - Distributed architecture, S3-compatible storage, component scaling
- **[TempoMonolithic with Memory Storage](./e2e/monolithic-memory/README.md)** - Single-pod deployment, embedded storage, development environments
- **[Monolithic Single Tenant Auth](./e2e-openshift/monolithic-single-tenant-auth/README.md)** - Simple deployment with OpenShift authentication

### Cloud Integrations & Advanced Features
- **[AWS STS Authentication](./e2e-openshift-object-stores/aws-sts-tempostack/README.md)** - IAM roles, STS tokens, secure AWS S3 access
- **[Azure Workload Identity](./e2e-openshift-object-stores/azure-wif-tempostack/README.md)** - Managed identity, federated credentials, Azure Blob storage
- **[OpenShift Service Mesh](./e2e-openshift-ossm/ossm-tempostack/README.md)** - Istio integration, service mesh observability, sidecar tracing
- **[Knative Serverless](./e2e-openshift-serverless/tempo-serverless/README.md)** - Serverless workload tracing, event-driven architectures

### Monitoring & Operations
- **[RED Metrics & Alerting](./e2e-openshift/red-metrics/README.md)** - Prometheus integration, custom metrics, alert rules
- **[Component Scaling](./e2e-openshift/component-replicas/README.md)** - Horizontal scaling, resource management, performance tuning
- **[Monitoring Integration](./e2e-openshift/monitoring/README.md)** - ServiceMonitors, operator metrics, workload monitoring

### Core Deployments
- **[TempoStack Compatibility](./e2e/compatibility/README.md)** - Multi-component distributed deployment, dual query APIs
- **[Custom CA Integration](./e2e/custom-ca/README.md)** - Custom certificate authorities, enterprise PKI integration
- **[Memory Storage Deployment](./e2e/monolithic-memory/README.md)** - Development and testing configurations

Each README includes:
- **Architecture diagrams** showing component relationships and data flow
- **Step-by-step deployment** instructions with kubectl/oc commands
- **Configuration details** with YAML examples and customization options
- **Security considerations** and best practices for production
- **Troubleshooting guides** with common issues and debug commands
- **Production considerations** including scaling, monitoring, and compliance
- **Related configurations** linking to complementary setups

## Test Environment Requirements

- Kubernetes cluster (or OpenShift for openshift-specific tests)
- kubectl/oc CLI tools
- Cloud provider CLIs (aws, az, gcloud) for object store tests
- OLM installed for upgrade tests
- Sufficient cluster resources for distributed deployments