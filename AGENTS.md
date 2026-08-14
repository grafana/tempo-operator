# AGENTS.md

This file provides guidance to AI assistants when working with code in this repository.

## Project Overview

The Tempo Operator is a Kubernetes operator for managing [Grafana Tempo](https://github.com/grafana/tempo), an open-source distributed tracing backend. It automates deployment, configuration, and management of Tempo clusters in Kubernetes environments including OpenShift.

## Architecture

**Core Components:**
- **TempoStack CR**: Primary custom resource defining a microservices Tempo deployment
- **TempoMonolithic CR**: Monolithic Tempo deployment for development/testing
- **Gateway**: Handles authentication, authorization, and multi-tenancy
- **Query Frontend**: Distributes query requests among the queriers. Jaeger UI is deployed in the same pod.
- **Distributor**: Receives and distributes traces to the ingesters
- **Ingester**: Batches trace data into blocks and writes them to storage
- **Querier**: Reads traces from storage and ingester cache
- **Compactor**: Compacts and deduplicates trace data in storage

**Directory Structure:**
- `cmd/`: Application entry point
- `api/`: CRD definitions (`tempo/v1alpha1`, `config/v1alpha1`)
- `internal/`: Core operator logic
- `config/`: Kubernetes deployment configurations and overlays
- `tests/`: End-to-end test suites
- `hack/`: Development and build automation scripts

## Changelog

Every PR that affects user-facing behavior must include a changelog entry. Create a new YAML file in the `.chloggen/` directory, using `.chloggen/TEMPLATE.yaml` as the template. Name the file after the change (e.g. `fix-query-timeout.yaml`). Do **not** edit `CHANGELOG.md` directly; it is generated from `.chloggen/` entries at release time.

## Development Commands

### Build and Test
```bash
make build              # Build operator binary
make test               # Run unit tests
make lint               # Run golangci-lint
make fmt                # Format Go code
```

### Code Generation
```bash
make reset              # Update all generated files
```

### Development Deployment
Use the `/deploy` skill, or invoke the script directly:
```bash
.claude/skills/deploy/deploy.sh
```

### Testing
```bash
# Unit tests
make test

# Run end-to-end tests. These run on Kubernetes and OpenShift.
make e2e

# Upgrade tests
make e2e-upgrade

# Run OpenShift-specific tests.
# This command may run up to 30 minutes, adjust the command timeout accordingly.
make e2e-openshift

# Run a single e2e test
chainsaw test --test-dir ./tests/e2e/reconcile

# Run a single e2e test and keep the objects to allow debugging
chainsaw test --test-dir ./tests/e2e/reconcile --skip-delete

# Single test execution example
go test ./internal/manifests/... -run TestManifests
```

## Configuration

**Custom Resources:**
- `TempoStack`: Production-ready distributed Tempo deployment
- `TempoMonolithic`: Monolithic deployment for development/testing

## Important Technical Details

**Versioning:** Component versions are managed in the Makefile (TEMPO_VERSION, JAEGER_QUERY_VERSION, etc.)

**Image Management:** All container images are configurable via environment variables in the manager deployment

**Webhook Configuration:** Admission webhooks for validation and mutation are automatically configured when deployed via manifests (disabled in `make run`)

**Dependencies:**
- Kubernetes
- OpenShift (for OpenShift deployments)
- cert-manager (for TLS certificate management)
- Object storage (S3-compatible)

**Testing Framework:**
- Unit tests: Ginkgo/Gomega
- E2E tests: Chainsaw test runner

**Bundle Variants:**
- `community`: Standard Kubernetes deployment
- `openshift`: OpenShift-specific features and configurations
