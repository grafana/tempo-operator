# TLS Profile E2E Tests

This directory contains end-to-end tests for TLS profile management in the Tempo Operator on OpenShift. The tests verify that TLS security profiles (Intermediate, Modern) are correctly applied across all Tempo components at multiple configuration levels.

## Prerequisites

- OpenShift cluster with OLM-managed Tempo Operator
- `oc` and `kubectl` CLI tools
- Chainsaw test runner (`chainsaw`)
- The `tls-scanner` image (`quay.io/rhn_support_ikanse/tls-scanner:latest`)

## Running the Tests

```bash
# Run all TLS profile tests (may take up to 60 minutes)
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-tls-profile

# Run individual tests
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-tls-profile/tls-profile
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-tls-profile/tls-profile-override-mono
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-tls-profile/tls-profile-override-stack

# Debug a failing test (keep resources for inspection)
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-tls-profile/tls-profile-override-mono --skip-delete
```

All tests use `concurrent: false` because they modify cluster-wide resources (APIServer CR or operator Subscription).

## TLS Profile Precedence

The operator resolves TLS settings with the following priority (highest to lowest):

1. **Per-CR overrides** (`spec.storage.tls.minVersion`, `spec.template.distributor.tls.minVersion`, etc.)
2. **Operator env var** (`TLS_PROFILE` on the operator deployment, set via OLM Subscription)
3. **OpenShift APIServer CR** (when `openshift.clusterTLSPolicy` feature gate is enabled)
4. **Built-in default** (Intermediate profile)

> **Note:** Currently, levels 2 and 3 are mutually exclusive (`if/else` in `internal/tlsprofile/get.go`). The `openshift.clusterTLSPolicy` feature gate must be disabled to use the `TLS_PROFILE` env var. See the code for details.

## Test Descriptions

### 1. `tls-profile/` - APIServer-Level TLS Profile

**Purpose:** Verifies that the operator reads and applies TLS profiles from the OpenShift APIServer CR (`openshift.clusterTLSPolicy` feature gate).

**Namespace:** `chainsaw-tls-profile-gw`

**Deployment model:** TempoStack (distributed with gateway)

| Step | Description |
|------|-------------|
| 00 | Install MinIO object storage with self-signed TLS certificates |
| 01 | Deploy TempoStack with gateway, Jaeger UI, and storage TLS enabled |
| 02 | Deploy tls-scanner pod for TLS verification |
| 03 | **Verify Intermediate profile** - ConfigMap (`tls_min_version: VersionTLS12`), gateway args (`--tls.min-version=VersionTLS12`), nmap ssl-enum-ciphers on gateway and internal gRPC ports, operator webhook/metrics ports |
| 04 | Install OpenTelemetry Collector for trace forwarding |
| 05 | Generate traces via gRPC and HTTP pipelines |
| 06 | Verify traces are queryable through the gateway (Jaeger API + TraceQL) |
| 07 | **Patch APIServer to Modern** - Sets `tlsSecurityProfile: {type: Modern}`, waits for MCO node reconciliation and operator pod restart (up to 25 minutes) |
| 08 | **Verify Modern profile** - All components switch to TLSv1.3 only. Recreates tls-scanner if evicted by MCO |
| 09 | Generate traces under Modern profile |
| 10 | Verify traces are queryable under Modern profile |
| 11 | **Revert APIServer** to default profile, wait for operator webhook availability |

**Cleanup:** Catch block reverts the APIServer TLS profile on any failure.

**Components verified:**
- Gateway HTTP (8080) and gRPC (8090) ports
- Internal gRPC (9095) on ingester and query-frontend
- Operator webhook (9443) and metrics (8443) ports
- Storage TLS config in ConfigMap
- Trace ingestion and query end-to-end

---

### 2. `tls-profile-override-mono/` - Subscription + Per-CR Overrides (Monolithic)

**Purpose:** Tests two override mechanisms on TempoMonolithic:
- **Subscription-level:** Setting `TLS_PROFILE=Modern` via the OLM Subscription env var
- **Per-CR level:** Setting `minVersion: "1.3"` directly on individual CR components

**Namespace:** `chainsaw-tls-profile-mono`

**Deployment model:** TempoMonolithic (single pod with gRPC and HTTP receivers)

| Step | Description |
|------|-------------|
| 00 | **Patch operator Subscription** - Set `TLS_PROFILE=Modern`, disable `openshift.clusterTLSPolicy` feature gate, wait for OLM propagation and operator rollout |
| 01 | Install MinIO object storage with self-signed TLS |
| 02 | Deploy TempoMonolithic with gRPC and HTTP TLS receivers |
| 03 | Deploy tls-scanner pod |
| 04 | **Verify subscription override** - ConfigMap shows `min_version: "1.3"` for gRPC/HTTP receivers, `tls_min_version: VersionTLS13` for storage. Functional TLS checks and nmap verification on ports 4317/4318 |
| 05 | Install OpenTelemetry Collector (dual gRPC/HTTP pipelines) |
| 06 | Generate traces via gRPC and HTTP |
| 07 | Verify traces via Jaeger UI API |
| 08 | **Revert Subscription** - Remove `TLS_PROFILE`, restore feature gates, wait for rollout |
| 09 | **Apply per-CR overrides** - Update TempoMonolithic with `minVersion: "1.3"` on storage, gRPC, and HTTP TLS configs |
| 10 | **Verify per-CR overrides** - Wait for StatefulSet rollout, verify ConfigMap has correct override values (`min_version: "1.3"` for receivers, `tls_min_version: VersionTLS13` for storage), functional TLS checks |

**Cleanup:** Catch block reverts the operator Subscription on any failure.

**Per-CR override fields tested:**
- `spec.storage.traces.s3.tls.minVersion`
- `spec.ingestion.otlp.grpc.tls.minVersion`
- `spec.ingestion.otlp.http.tls.minVersion`

---

### 3. `tls-profile-override-stack/` - Subscription + Per-CR Overrides (TempoStack)

**Purpose:** Same override mechanisms as the mono test, but on a distributed TempoStack deployment with gateway.

**Namespace:** `chainsaw-tls-profile-gw-ovr`

**Deployment model:** TempoStack (distributed with gateway, multi-tenant)

| Step | Description |
|------|-------------|
| 00 | **Patch operator Subscription** - Set `TLS_PROFILE=Modern`, disable `openshift.clusterTLSPolicy` feature gate |
| 01 | Install MinIO object storage with self-signed TLS |
| 02 | Deploy TempoStack with gateway, multi-tenancy (OpenShift mode), storage TLS |
| 03 | Deploy tls-scanner pod |
| 04 | **Verify subscription override** - ConfigMap shows Modern profile for storage and distributor receivers. Gateway args show `--tls.min-version=VersionTLS13`. Functional checks and nmap on gateway ports. Internal gRPC scan via `-all-pods` |
| 05 | Install OpenTelemetry Collector with gateway bearer token auth |
| 06 | Generate traces via gRPC and HTTP through the gateway |
| 07 | Verify traces via gateway (Jaeger API + TraceQL, with SA token auth) |
| 08 | **Revert Subscription** - Remove `TLS_PROFILE`, restore feature gates |
| 09 | **Apply per-CR override** - Update TempoStack with `storage.tls.minVersion: "1.3"` |
| 10 | **Verify per-CR override** - Wait for rollouts, verify ConfigMap shows `tls_min_version: VersionTLS13` for storage, functional TLS checks on gateway |

**Cleanup:** Catch block reverts the operator Subscription on any failure.

**Per-CR override fields tested:**
- `spec.storage.tls.minVersion`

> **Note:** `spec.template.distributor.tls` cannot be enabled when `spec.template.gateway.enabled` is true (webhook validation rejects this combination).

## Verification Methods

The tests use multiple verification approaches:

| Method | What it verifies |
|--------|-----------------|
| **ConfigMap inspection** | Tempo config has correct `tls_min_version` and `min_version` values |
| **Deployment args inspection** | Gateway container args include correct `--tls.min-version` and `--tls.cipher-suites` |
| **tls-scanner functional check** | TLS handshake succeeds on each port |
| **nmap ssl-enum-ciphers** | Actual TLS versions and cipher suites offered by listening ports |
| **tls-scanner `-all-pods`** | Scans all pods in the namespace for TLS information per port |
| **Trace generation + query** | End-to-end validation that traces flow through TLS-secured pipelines |
