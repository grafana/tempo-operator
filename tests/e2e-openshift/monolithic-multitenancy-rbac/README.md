# TempoMonolithic with RBAC-Based Multitenancy

This configuration blueprint demonstrates TempoMonolithic's RBAC-based multitenancy approach, which provides namespace-level trace isolation using OpenShift's native Role-Based Access Control system. This setup allows multiple teams or projects to share a single Tempo instance while maintaining secure data separation based on Kubernetes RBAC permissions and namespace boundaries.

## Overview

This test validates RBAC-driven multitenancy features:
- **Namespace-Level Isolation**: Trace data segregated by OpenShift project/namespace
- **RBAC Integration**: Fine-grained access control using Kubernetes RBAC
- **ServiceAccount-Based Authentication**: Automatic tenant identification via namespace context
- **Admin Override**: Cluster admin access to all tenant data
- **Cross-Namespace Security**: Validation of access control enforcement

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ Project: mono-rbac-1    │───▶│   TempoMonolithic        │───▶│ Tenant Isolation       │
│ - ServiceAccount: sa-1  │    │   with RBAC Query        │    │ - Namespace-based       │
│ - Admin permissions     │    │ ┌─────────────────────┐  │    │ - RBAC-controlled       │
│ - Generate traces       │    │ │ RBAC Query Layer    │  │    └─────────────────────────┘
└─────────────────────────┘    │ │ - Namespace check   │  │
                               │ │ - Permission verify │  │    ┌─────────────────────────┐
┌─────────────────────────┐    │ │ - Access control    │  │───▶│ Traces: mono-rbac-1     │
│ Project: mono-rbac-2    │───▶│ └─────────────────────┘  │    │ - Only sa-1 can access  │
│ - ServiceAccount: sa-2  │    │ Gateway Component        │    │ - Isolated from rbac-2  │
│ - Admin permissions     │    └──────────────────────────┘    └─────────────────────────┘
│ - Generate traces       │
└─────────────────────────┘                                    ┌─────────────────────────┐
                                                               │ Traces: mono-rbac-2     │
┌─────────────────────────┐    ┌──────────────────────────┐    │ - Only sa-2 can access  │
│ Cluster Admin           │───▶│   Admin Access          │───▶│ - Isolated from rbac-1  │
│ - cluster-admin role    │    │   - All namespaces      │    └─────────────────────────┘
│ - View all traces       │    │   - Override RBAC       │
└─────────────────────────┘    └──────────────────────────┘    ┌─────────────────────────┐
                                                               │ Admin View              │
                                                               │ - All traces visible    │
                                                               │ - Cross-tenant access   │
                                                               └─────────────────────────┘
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed
- OpenTelemetry Operator installed
- Cluster administrator privileges for project and RBAC creation
- Understanding of OpenShift projects and RBAC concepts

## Step-by-Step Configuration

### Step 1: Deploy TempoMonolithic with RBAC Query Support

Create TempoMonolithic with RBAC-based query authorization:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: mmo-rbac
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
EOF
```

**Key Configuration Elements**:

#### RBAC Query Layer
- `query.rbac.enabled: true`: Activates RBAC-based query authorization
- **Namespace Validation**: Queries validated against user's namespace permissions
- **Permission Checks**: Every query checked against RBAC rules

#### Multitenancy Setup
- `mode: openshift`: Uses OpenShift-native multitenancy
- **Tenant Mapping**: Maps internal tenant IDs to namespace-based access
- **Authentication**: ServiceAccount-based tenant identification

### Step 2: Configure RBAC Permissions for Tenant Access

Set up global RBAC rules for trace access:

```bash
# Create ClusterRole for dev tenant trace access
oc apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: allow-read-traces-dev-tenant-rbac
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev]
  resourceNames: [traces]
  verbs: [get]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: allow-read-traces-dev-tenant-rbac
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: allow-read-traces-dev-tenant-rbac
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: system:authenticated
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: view
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
- kind: ServiceAccount
  name: default
  namespace: chainsaw-mmo-rbac
EOF
```

**RBAC Configuration Details**:

#### Global Trace Access
- **ClusterRole**: Defines permissions for trace resource access
- **Resource Scope**: `resources: [dev]` limits to dev tenant traces
- **Verb Control**: `verbs: [get]` allows read-only access

#### Authenticated Users
- **Subject Group**: `system:authenticated` includes all authenticated users
- **Broad Access**: Initial setup allows authenticated users to read dev traces
- **Namespace Scoping**: Further refined by namespace-level permissions

**Reference**: [`01-install-tempo.yaml`](./01-install-tempo.yaml)

### Step 3: Deploy OpenTelemetry Collector for Multi-Tenant Ingestion

Create the collector that will handle trace ingestion for multiple tenants:

```bash
# This collector receives traces from multiple namespaces
# and forwards them to the TempoMonolithic gateway
# Reference: 02-install-otelcol.yaml
```

**Reference**: [`02-install-otelcol.yaml`](./02-install-otelcol.yaml)

### Step 4: Create Multiple Projects with Isolated ServiceAccounts

Set up separate OpenShift projects for RBAC testing:

```bash
oc apply -f - <<EOF
apiVersion: project.openshift.io/v1
kind: Project
metadata:
  name: chainsaw-mono-rbac-1
spec: {}
---
apiVersion: project.openshift.io/v1
kind: Project
metadata:
  name: chainsaw-mono-rbac-2
spec: {}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tempo-rbac-sa-1
  namespace: chainsaw-mono-rbac-1
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tempo-rbac-sa-2
  namespace: chainsaw-mono-rbac-2
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tempo-rbac-cluster-admin
  namespace: chainsaw-mmo-rbac
EOF
```

### Step 5: Configure Namespace-Level RBAC Permissions

Grant appropriate permissions to ServiceAccounts in their respective namespaces:

```bash
# Grant sa-1 admin permissions in mono-rbac-1 namespace
oc apply -f - <<EOF
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: chainsaw-mono-rbac-1-admin
  namespace: chainsaw-mono-rbac-1
subjects:
  - kind: ServiceAccount
    name: tempo-rbac-sa-1
    namespace: chainsaw-mono-rbac-1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: chainsaw-mono-rbac-2-admin
  namespace: chainsaw-mono-rbac-2
subjects:
  - kind: ServiceAccount
    name: tempo-rbac-sa-2
    namespace: chainsaw-mono-rbac-2
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tempo-rbac-cluster-admin-binding-monolithic
subjects:
  - kind: ServiceAccount
    name: tempo-rbac-cluster-admin
    namespace: chainsaw-mmo-rbac
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: chainsaw-test-rbac-1-testuser
  namespace: chainsaw-mono-rbac-1
subjects:
  - kind: User
    name: testuser-0
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
EOF
```

**RBAC Hierarchy Setup**:

#### Namespace-Level Admin Access
- **sa-1**: Admin access in `chainsaw-mono-rbac-1` only
- **sa-2**: Admin access in `chainsaw-mono-rbac-2` only
- **Isolation**: Each ServiceAccount limited to its namespace

#### Cluster-Level Admin Access
- **cluster-admin SA**: Full cluster access for testing admin override
- **Cross-Namespace**: Can access traces from all namespaces

#### User-Level Access
- **testuser-0**: Admin access in specific namespace for user-based testing

**Reference**: [`create-SAs-with-namespace-access.yaml`](./create-SAs-with-namespace-access.yaml)

### Step 6: Generate Namespace-Specific Traces

Create traces from each namespace with distinctive identifiers:

```bash
# Generate traces from chainsaw-mono-rbac-1
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces-grpc-sa-1
  namespace: chainsaw-mono-rbac-1
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
        args:
        - traces
        - --otlp-endpoint=dev-collector.chainsaw-mmo-rbac.svc:4317
        - --service=grpc-rbac-1
        - --otlp-insecure
        - --traces=2
        - --otlp-attributes=k8s.container.name="telemetrygen"
        - --otlp-attributes=k8s.namespace.name="chainsaw-mono-rbac-1"
      restartPolicy: Never
---
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces-http-sa-1
  namespace: chainsaw-mono-rbac-1
spec:
  template:
    spec:
      containers:
        - name: telemetrygen
          image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
          args:
            - traces
            - --otlp-endpoint=dev-collector.chainsaw-mmo-rbac.svc:4318
            - --otlp-http
            - --otlp-insecure
            - --service=http-rbac-1
            - --traces=2
            - --otlp-attributes=k8s.container.name="telemetrygen"
            - --otlp-attributes=k8s.namespace.name="chainsaw-mono-rbac-1"
      restartPolicy: Never
EOF

# Similar job for chainsaw-mono-rbac-2 namespace
# Reference: tempo-rbac-sa-2-traces-gen.yaml
```

**Trace Generation Strategy**:

#### Namespace Identification
- **Service Names**: `grpc-rbac-1`, `http-rbac-1` for namespace 1
- **Attributes**: Include namespace name for clear identification
- **Protocol Testing**: Both gRPC and HTTP protocols tested

#### Cross-Namespace Testing
- **Isolated Generation**: Each namespace generates its own traces
- **Distinctive Naming**: Service names indicate source namespace
- **Attribute Tagging**: K8s namespace attributes for verification

**References**: [`tempo-rbac-sa-1-traces-gen.yaml`](./tempo-rbac-sa-1-traces-gen.yaml), [`tempo-rbac-sa-2-traces-gen.yaml`](./tempo-rbac-sa-2-traces-gen.yaml)

### Step 7: Validate RBAC-Based Trace Access

Test that ServiceAccounts can only access traces from their authorized namespaces:

```bash
# Verify sa-1 can access only its traces
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-rbac-traces-sa-1
  namespace: chainsaw-mono-rbac-1
spec:
  template:
    spec:
      serviceAccountName: tempo-rbac-sa-1
      containers:
      - name: verify-traces
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          TOKEN=\$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
          
          # Query traces using ServiceAccount token
          curl -v -G \
            -H "Authorization: Bearer \$TOKEN" \
            https://tempo-mmo-rbac-gateway:8080/api/search/dev \
            --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
            --data-urlencode "q={.service.name=\\"grpc-rbac-1\\"}" | tee /tmp/rbac.out
          
          # Verify only authorized traces are returned
          num_traces=\$(jq ".traces | length" /tmp/rbac.out)
          if [[ "\$num_traces" -lt 1 ]]; then
            echo "Expected traces for grpc-rbac-1 service, got \$num_traces"
            exit 1
          fi
          
          # Attempt to access unauthorized traces (should fail)
          curl -v -G \
            -H "Authorization: Bearer \$TOKEN" \
            https://tempo-mmo-rbac-gateway:8080/api/search/dev \
            --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
            --data-urlencode "q={.service.name=\\"grpc-rbac-2\\"}" | tee /tmp/unauthorized.out
          
          unauthorized_traces=\$(jq ".traces | length" /tmp/unauthorized.out)
          if [[ "\$unauthorized_traces" -gt 0 ]]; then
            echo "ERROR: Found unauthorized traces from rbac-2: \$unauthorized_traces"
            exit 1
          fi
          
          echo "✓ RBAC isolation verified for sa-1"
      restartPolicy: Never
EOF
```

**RBAC Validation Strategy**:

#### Authorized Access Testing
- **ServiceAccount Token**: Uses namespace-specific ServiceAccount
- **Authorized Queries**: Queries for traces from same namespace
- **Expected Results**: Should return traces from authorized services

#### Unauthorized Access Testing
- **Cross-Namespace Queries**: Attempts to access traces from other namespaces
- **Expected Failure**: Should return empty results for unauthorized traces
- **Security Validation**: Confirms RBAC enforcement is working

**Reference**: [`tempo-rbac-sa-1-traces-verify.yaml`](./tempo-rbac-sa-1-traces-verify.yaml)

### Step 8: Validate Cluster Admin Override

Test that cluster administrators can access traces from all namespaces:

```bash
# Verify cluster-admin can view all traces
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-kubeadmin-traces
  namespace: chainsaw-mmo-rbac
spec:
  template:
    spec:
      serviceAccountName: tempo-rbac-cluster-admin
      containers:
      - name: verify-admin-access
        image: ghcr.io/grafana/tempo-operator/test-utils:main
        command:
        - /bin/bash
        - -eux
        - -c
        args:
        - |
          TOKEN=\$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
          
          # Query all traces as cluster admin
          curl -v -G \
            -H "Authorization: Bearer \$TOKEN" \
            https://tempo-mmo-rbac-gateway:8080/api/search/dev \
            --cacert /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt \
            --data-urlencode "q={}" | tee /tmp/admin.out
          
          # Verify admin can see traces from all namespaces
          total_traces=\$(jq ".traces | length" /tmp/admin.out)
          if [[ "\$total_traces" -lt 4 ]]; then
            echo "Expected at least 4 traces (from both namespaces), got \$total_traces"
            exit 1
          fi
          
          # Verify specific traces from both namespaces are visible
          rbac1_traces=\$(jq '.traces[] | select(.rootServiceName | contains("rbac-1"))' /tmp/admin.out | wc -l)
          rbac2_traces=\$(jq '.traces[] | select(.rootServiceName | contains("rbac-2"))' /tmp/admin.out | wc -l)
          
          if [[ "\$rbac1_traces" -lt 1 ]] || [[ "\$rbac2_traces" -lt 1 ]]; then
            echo "ERROR: Admin should see traces from both namespaces"
            echo "rbac-1 traces: \$rbac1_traces, rbac-2 traces: \$rbac2_traces"
            exit 1
          fi
          
          echo "✓ Cluster admin can access traces from all namespaces"
      restartPolicy: Never
EOF
```

**Admin Override Validation**:

#### Comprehensive Access
- **Cross-Namespace Visibility**: Admin can see traces from all namespaces
- **No RBAC Restrictions**: cluster-admin role bypasses namespace restrictions
- **Complete View**: Total trace count includes all generated traces

#### Permission Hierarchy
- **Role Priority**: cluster-admin overrides namespace-level restrictions
- **Security Model**: Maintains admin oversight while enforcing tenant isolation
- **Audit Capability**: Enables cluster-wide trace analysis when needed

**Reference**: [`kubeadmin-traces-verify.yaml`](./kubeadmin-traces-verify.yaml)

## RBAC-Based Multitenancy Features

### 1. **Namespace-Level Isolation**

#### Automatic Tenant Mapping
```yaml
# Namespace automatically maps to tenant context
spec:
  query:
    rbac:
      enabled: true
      # Namespace of requesting ServiceAccount becomes tenant context
```

#### Permission-Based Access
```yaml
# ServiceAccount permissions determine trace visibility
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: trace-access
  namespace: team-a
subjects:
- kind: ServiceAccount
  name: team-a-collector
roleRef:
  kind: ClusterRole
  name: allow-read-team-a-traces
```

### 2. **Multi-Level RBAC Hierarchy**

#### Project-Level Access
```yaml
# Admin access within specific project
roleRef:
  kind: ClusterRole
  name: admin
# Scope limited to single namespace
```

#### Cluster-Level Access
```yaml
# Override access for administrators
roleRef:
  kind: ClusterRole
  name: cluster-admin
# Access to all namespaces and tenants
```

#### User-Level Access
```yaml
# Individual user permissions
subjects:
- kind: User
  name: developer@company.com
# Can be scoped to specific namespaces
```

### 3. **Query Authorization Flow**

#### Request Processing
1. **Token Extraction**: Extract ServiceAccount token from request
2. **Identity Verification**: Validate token and extract namespace context
3. **Permission Check**: Verify RBAC permissions for target tenant
4. **Query Filtering**: Apply namespace-based filters to query results
5. **Response Generation**: Return only authorized trace data

#### Security Enforcement
```yaml
# Example RBAC rule for trace access
rules:
- apiGroups: [tempo.grafana.com]
  resources: [traces]
  resourceNames: [team-a-tenant]
  verbs: [get, list]
  namespaces: [team-a-namespace]
```

## Advanced RBAC Configurations

### 1. **Team-Based Access Control**

#### Development Team Setup
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: dev-team-trace-access
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev, staging]
  resourceNames: [traces]
  verbs: [get, list, create]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: dev-team-binding
subjects:
- kind: Group
  name: dev-team
roleRef:
  kind: ClusterRole
  name: dev-team-trace-access
```

#### Operations Team Setup
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ops-team-trace-access
rules:
- apiGroups: [tempo.grafana.com]
  resources: [dev, staging, production]
  resourceNames: [traces, metrics]
  verbs: [get, list]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ops-team-binding
subjects:
- kind: Group
  name: ops-team
roleRef:
  kind: ClusterRole
  name: ops-team-trace-access
```

### 2. **Service-Level Permissions**

#### Read-Only Access
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: trace-reader
rules:
- apiGroups: [tempo.grafana.com]
  resources: ["*"]
  resourceNames: [traces]
  verbs: [get, list]
```

#### Write-Only Access (for collectors)
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: trace-writer
rules:
- apiGroups: [tempo.grafana.com]
  resources: ["*"]
  resourceNames: [traces]
  verbs: [create]
```

### 3. **Time-Based Access Control**

#### Temporary Access Grants
```bash
# Grant temporary access to specific user
oc create rolebinding temp-trace-access \
  --clusterrole=trace-reader \
  --user=temp-user@company.com \
  --namespace=investigation-namespace

# Set expiration (manual cleanup required)
# Consider using a tool like kube-rbac-proxy for automatic expiration
```

## Security Considerations

### 1. **Token Security**

#### ServiceAccount Token Management
```bash
# Monitor token usage
oc get serviceaccount -A -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.secrets}{"\n"}{end}'

# Rotate tokens (OpenShift 4.11+)
oc create token <serviceaccount-name> --duration=1h
```

#### Token Scoping
```yaml
# Limit token permissions
apiVersion: v1
kind: ServiceAccount
metadata:
  name: limited-collector
  annotations:
    serviceaccounts.openshift.io/oauth-redirectreference.tempo: '{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"tempo-ui"}}'
automountServiceAccountToken: true
```

### 2. **Audit and Compliance**

#### Access Logging
```yaml
# Enable audit logging in cluster
apiVersion: config.openshift.io/v1
kind: APIServer
metadata:
  name: cluster
spec:
  audit:
    profile: Default
```

#### RBAC Monitoring
```bash
# Monitor RBAC changes
oc get events --field-selector reason=PolicyRule -A

# Regular RBAC audits
oc get clusterrolebinding -o json | jq '.items[] | select(.subjects[]?.name | contains("tempo"))'
```

### 3. **Namespace Isolation**

#### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-access-control
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: tempo-monolithic
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          tempo-access: "allowed"
```

#### Resource Quotas
```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tempo-quota
  namespace: tenant-namespace
spec:
  hard:
    requests.cpu: "2"
    requests.memory: 4Gi
    persistentvolumeclaims: "1"
```

## Troubleshooting RBAC Issues

### 1. **Access Denied Errors**

#### Permission Debugging
```bash
# Check user permissions
oc auth can-i get traces --as=system:serviceaccount:namespace:serviceaccount

# Debug RBAC rules
oc describe clusterrole trace-access
oc describe clusterrolebinding trace-access-binding

# Check ServiceAccount details
oc get serviceaccount <sa-name> -o yaml
oc describe secret <sa-token-secret>
```

#### Token Validation
```bash
# Validate token manually
TOKEN=$(oc create token <serviceaccount>)
curl -H "Authorization: Bearer $TOKEN" \
  https://kubernetes.default.svc/api/v1/namespaces
```

### 2. **Cross-Namespace Access Issues**

#### Namespace Visibility
```bash
# Check namespace labels and selectors
oc get namespace --show-labels

# Verify RoleBinding namespace scope
oc get rolebinding -A | grep tempo

# Check for conflicting permissions
oc auth reconcile --dry-run=server -f rbac.yaml
```

### 3. **Gateway Authorization Problems**

#### Gateway Logs Analysis
```bash
# Check gateway authorization logs
oc logs -l app.kubernetes.io/component=gateway | grep -i "auth\|rbac\|denied"

# Monitor failed requests
oc logs -l app.kubernetes.io/component=gateway | grep -E "(401|403|unauthorized)"

# Test gateway health
oc port-forward svc/tempo-mmo-rbac-gateway 8080:8080 &
curl -k https://localhost:8080/api/v1/status
```

## Production Deployment Best Practices

### 1. **RBAC Strategy Design**

#### Organizational Alignment
- **Team-Based**: Align tenants with organizational teams
- **Environment-Based**: Separate dev/staging/prod access
- **Service-Based**: Microservice-specific trace access
- **Hybrid Approach**: Combination based on business needs

#### Permission Granularity
```yaml
# Fine-grained permissions example
rules:
- apiGroups: [tempo.grafana.com]
  resources: [frontend-team]
  resourceNames: [traces]
  verbs: [get, list]
  namespaces: [frontend-dev, frontend-staging]
```

### 2. **Automation and GitOps**

#### RBAC as Code
```yaml
# Store all RBAC configurations in Git
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: tempo-rbac
spec:
  source:
    repoURL: https://github.com/company/tempo-rbac
    path: rbac/
  destination:
    server: https://kubernetes.default.svc
```

#### Automated Testing
```bash
# RBAC validation in CI/CD
#!/bin/bash
# Test that each ServiceAccount has correct permissions
for sa in $(oc get sa -o name); do
  oc auth can-i get traces --as=system:serviceaccount:$(oc project -q):$(basename $sa)
done
```

### 3. **Monitoring and Compliance**

#### Access Metrics
```yaml
# Prometheus metrics for RBAC monitoring
alert: UnauthorizedTraceAccess
expr: rate(tempo_gateway_unauthorized_requests_total[5m]) > 0
for: 1m
annotations:
  summary: "Unauthorized trace access attempts detected"
```

#### Compliance Reporting
```bash
# Generate access reports
oc get clusterrolebinding -o json | \
  jq '.items[] | select(.roleRef.name | contains("tempo"))' | \
  jq '{binding: .metadata.name, subjects: .subjects, role: .roleRef.name}'
```

## Related Configurations

- [OpenShift Native Multitenancy](../monolithic-multitenancy-openshift/README.md) - Full OpenShift integration
- [Static Multitenancy](../monolithic-multitenancy-static/README.md) - Configuration-based tenants
- [TempoStack RBAC](../multitenancy-rbac/README.md) - Distributed RBAC multitenancy

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monolithic-multitenancy-rbac
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test creates multiple OpenShift projects and requires cluster administrator privileges for RBAC setup. The test validates strict namespace-level isolation and proper RBAC enforcement across different access levels.

