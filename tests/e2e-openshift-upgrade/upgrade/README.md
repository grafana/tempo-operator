# OpenShift Tempo Operator Upgrade Test with Multitenancy and RBAC

This comprehensive test validates the Tempo Operator upgrade process while maintaining complex production-like configurations including multitenancy, RBAC, and OpenTelemetry Collector integration. It ensures that both TempoStack and TempoMonolithic deployments continue to function correctly through operator upgrades.

## Test Overview

### Purpose
- **Operator Upgrade Validation**: Tests seamless upgrade of Tempo Operator using File-Based Catalogs (FBC)
- **Dual Architecture Support**: Validates both TempoStack (distributed) and TempoMonolithic (single-component) deployments
- **Multitenancy Preservation**: Ensures OpenShift-native multitenancy continues to work after upgrades
- **RBAC Continuity**: Validates namespace-level role-based access control across upgrades
- **OpenTelemetry Integration**: Tests OTel Collector functionality with tenant isolation

### Components
- **TempoStack**: Distributed Tempo deployment with gateway, RBAC, and multitenancy
- **TempoMonolithic**: Single-component Tempo deployment with RBAC and multitenancy
- **OpenTelemetry Collectors**: Separate collectors for each deployment with tenant-specific configuration
- **MinIO Storage**: S3-compatible backend for TempoStack
- **File-Based Catalog**: Custom operator catalog for upgrade testing
- **Service Accounts**: Namespace-scoped service accounts for RBAC testing

## Architecture Overview

```
[TempoStack + TempoMonolithic] → [Operator Upgrade] → [Validation]
     ↓                               ↓                    ↓
[Multi-tenant RBAC]            [FBC Catalog]        [RBAC + Traces]
[OTel Collectors]              [Subscription]       [Post-upgrade]
[Trace Generation]             [CSV Update]         [Functionality]
```

## Environment Requirements

### OpenShift Prerequisites
- OpenShift cluster version 4.12 or higher
- Operator Lifecycle Manager (OLM) installed
- Tempo Operator must NOT be pre-installed
- Sufficient cluster resources for dual deployments

### Required Configuration Values
- **`upgrade_fbc_image`**: File-Based Catalog image containing the target operator version
- **`upgrade_operator_version`**: Target operator version for upgrade (e.g., `0.16.0`)
- **`upgrade_tempo_version`**: Target Tempo version after upgrade (e.g., `2.7.2`)
- **`upgrade_operator_csv_name`**: CSV name of the target operator version

## Deployment Steps

### Phase 1: Initial Deployment

#### 1. Install Base Operators
```bash
kubectl apply -f install-operators-from-marketplace.yaml
```

#### 2. Setup Storage Infrastructure
```bash
kubectl apply -f install-storage.yaml
```

#### 3. Deploy Multitenant TempoStack
```bash
kubectl apply -f install-tempostack.yaml
```

Key configuration from [`install-tempostack.yaml`](install-tempostack.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplst
  namespace: chainsaw-rbac
spec:
  storage:
    secret:
      name: minio
      type: s3
  tenants:
    mode: openshift
    authentication:
      - tenantName: dev
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
      - tenantName: prod
        tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
  template:
    gateway:
      enabled: true
      rbac:
        enabled: true
```

#### 4. Deploy Multitenant TempoMonolithic
```bash
kubectl apply -f install-tempo-monolithic.yaml
```

Key configuration from [`install-tempo-monolithic.yaml`](install-tempo-monolithic.yaml):
```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: mmo-rbac
  namespace: chainsaw-mmo-rbac
spec:
  query:
    rbac:
      enabled: true
  multitenancy:
    enabled: true
    mode: openshift
    authentication:
    - tenantName: dev
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfa"
    - tenantName: prod
      tenantId: "1610b0c3-c509-4592-a256-a1871353dbfb"
```

#### 5. Deploy OpenTelemetry Collectors
```bash
kubectl apply -f install-otelcol-tempostack.yaml
kubectl apply -f install-otelcol-tempomonolithic.yaml
```

Key OTel configuration for TempoStack from [`install-otelcol-tempostack.yaml`](install-otelcol-tempostack.yaml):
```yaml
exporters:
  otlp:
    endpoint: tempo-simplst-gateway.chainsaw-rbac.svc.cluster.local:8090
    tls:
      insecure: false
      ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
    auth:
      authenticator: bearertokenauth
    headers:
      X-Scope-OrgID: "dev"
```

#### 6. Create Service Accounts with Namespace Access
```bash
kubectl apply -f create-SAs-with-namespace-access-tempostack.yaml
kubectl apply -f create-SAs-with-namespace-access-tempomonolithic.yaml
```

### Phase 2: Pre-Upgrade Validation

#### 7. Generate Traces from Multiple Namespaces
```bash
kubectl apply -f tempostack-rbac-sa-1-traces-gen.yaml
kubectl apply -f tempostack-rbac-sa-2-traces-gen.yaml
kubectl apply -f tempo-mono-rbac-sa-1-traces-gen.yaml
kubectl apply -f tempo-mono-rbac-sa-2-traces-gen.yaml
```

#### 8. Verify RBAC Functionality
```bash
kubectl apply -f tempostack-rbac-sa-1-traces-verify.yaml
kubectl apply -f tempo-mono-rbac-sa-1-traces-verify.yaml
kubectl apply -f kubeadmin-tempostack-traces-verify.yaml
kubectl apply -f kubeadmin-tempo-mono-traces-verify.yaml
```

### Phase 3: Operator Upgrade

#### 9. Create Upgrade Catalog
```bash
kubectl apply -f create-upgrade-catalog.yaml
```

Key configuration from [`create-upgrade-catalog.yaml`](create-upgrade-catalog.yaml):
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: tempo-registry
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ($upgrade_fbc_image)
  publisher: Openshift QE
```

#### 10. Execute Operator Upgrade
```bash
kubectl apply -f upgrade-operator.yaml
```

### Phase 4: Post-Upgrade Validation

The test automatically verifies that both TempoStack and TempoMonolithic instances return to `Ready` state and validates RBAC functionality continues to work after upgrade.

## Key Features Tested

### Operator Upgrade Process
- ✅ File-Based Catalog (FBC) integration for custom operator upgrades
- ✅ ImageDigestMirrorSet configuration for registry mirroring
- ✅ Subscription-based upgrade triggering via OLM
- ✅ Operator CSV installation and validation

### Multitenancy Preservation
- ✅ OpenShift-native tenant isolation across upgrades
- ✅ Tenant-specific trace routing and storage
- ✅ X-Scope-OrgID header processing and validation
- ✅ Multiple tenant authentication configuration

### RBAC Continuity
- ✅ Namespace-level access control preservation
- ✅ Service account-based authentication
- ✅ ClusterRole and ClusterRoleBinding validation
- ✅ Admin vs. user permission differentiation

### OpenTelemetry Integration
- ✅ Dual-protocol OTLP support (gRPC and HTTP)
- ✅ Bearer token authentication with service accounts
- ✅ Kubernetes metadata enrichment
- ✅ Tenant-aware trace routing

### Data Persistence
- ✅ Trace data survival through operator upgrades
- ✅ Storage backend connectivity preservation
- ✅ Query functionality across upgrade boundaries

## Running the Test

### Using Heredoc for Parameter Passing
```bash
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-upgrade --values - <<EOF
upgrade_fbc_image: registry.example.com/your-org/tempo-fbc:latest
upgrade_operator_version: 0.16.0
upgrade_tempo_version: 2.7.2
upgrade_operator_csv_name: tempo-operator.v0.16.0-1
EOF
```

### Using Values File
Create a `values.yaml` file:
```yaml
upgrade_fbc_image: registry.example.com/your-org/tempo-fbc:latest
upgrade_operator_version: 0.16.0
upgrade_tempo_version: 2.7.2
upgrade_operator_csv_name: tempo-operator.v0.16.0-1
```

Then run:
```bash
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-upgrade --values values.yaml
```

## Namespace Organization

### Core Deployment Namespaces
- **`chainsaw-rbac`**: TempoStack deployment with multitenancy and RBAC
- **`chainsaw-mmo-rbac`**: TempoMonolithic deployment with multitenancy and RBAC
- **`openshift-tempo-operator`**: Tempo Operator installation namespace

### RBAC Testing Namespaces
- **`chainsaw-test-rbac-1`** & **`chainsaw-test-rbac-2`**: TempoStack RBAC validation namespaces
- **`chainsaw-mono-rbac-1`** & **`chainsaw-mono-rbac-2`**: TempoMonolithic RBAC validation namespaces

## Troubleshooting

### Common Issues

**Operator Upgrade Failures**:
- Verify FBC image is accessible and contains correct operator bundles
- Check ImageDigestMirrorSet configuration for registry access
- Monitor operator pod logs during upgrade transition

**RBAC Validation Failures**:
- Confirm service accounts have proper ClusterRole bindings
- Verify tenant IDs match between authentication and RBAC rules
- Check X-Scope-OrgID headers in trace requests

**OpenTelemetry Collector Issues**:
- Verify bearer token authentication configuration
- Check service account token mounting in collector pods
- Monitor collector logs for authentication and routing errors

### Upgrade-Specific Debugging

**Pre-Upgrade Validation**:
```bash
oc get csv -n openshift-tempo-operator
oc get tempo,tempomonolithic --all-namespaces
```

**During Upgrade Monitoring**:
```bash
oc get subscription -n openshift-tempo-operator -w
oc logs -n openshift-tempo-operator -l app.kubernetes.io/name=tempo-operator
```

**Post-Upgrade Validation**:
```bash
oc get tempo simplst -n chainsaw-rbac -o yaml
oc get tempomonolithic mmo-rbac -n chainsaw-mmo-rbac -o yaml
```

## Production Considerations

### Upgrade Planning
- **Backup Strategy**: Ensure trace data and configuration backups before upgrades
- **Maintenance Windows**: Plan upgrades during low-traffic periods
- **Version Compatibility**: Verify Tempo version compatibility with workloads

### RBAC Management
- **Principle of Least Privilege**: Grant minimal required permissions
- **Tenant Isolation**: Ensure proper tenant separation in multitenant deployments
- **Regular Audits**: Periodically review service account permissions

This comprehensive upgrade test ensures that complex production-like Tempo deployments with multitenancy and RBAC can be safely upgraded while maintaining full functionality and data integrity.