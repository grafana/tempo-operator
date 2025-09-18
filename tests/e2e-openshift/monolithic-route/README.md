# TempoMonolithic with OpenShift Route and Must-Gather Validation

This configuration blueprint demonstrates deploying TempoMonolithic with OpenShift Route for external access and validates the must-gather functionality for comprehensive troubleshooting and support data collection. This setup ensures proper external connectivity via OpenShift's native routing system and validates operational tooling for production support scenarios.

## Overview

This test validates OpenShift Route integration and operational tooling:
- **OpenShift Route**: Native external access through OpenShift's HAProxy router
- **Jaeger UI External Access**: Direct access to Jaeger UI from outside the cluster
- **Must-Gather Integration**: Comprehensive data collection for troubleshooting
- **Operational Readiness**: Production-ready deployment with support tooling

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ External Users          │───▶│   OpenShift Route        │───▶│ TempoMonolithic         │
│ - Web Browsers          │    │   - HAProxy Router       │    │ - Jaeger UI             │
│ - Direct Access         │    │   - TLS Termination      │    │ - In-memory Storage     │
│ - HTTPS/HTTP            │    │   - Load Balancing       │    │ - All-in-one Deployment │
└─────────────────────────┘    └──────────────────────────┘    └─────────────────────────┘

┌─────────────────────────┐    ┌──────────────────────────┐    
│ Must-Gather Tool        │───▶│   Operational Data       │    
│ - Resource Collection   │    │   Collection             │    
│ - Configuration Dump    │    │ ┌─────────────────────┐  │    
│ - Troubleshooting Info  │    │ │ Tempo Resources     │  │    
└─────────────────────────┘    │ │ - TempoMonolithic   │  │    
                               │ │ - StatefulSet       │  │    
┌─────────────────────────┐    │ │ - Services          │  │    
│ Support Bundle          │◀───│ │ - Routes            │  │    
│ - All K8s Resources     │    │ │ - ConfigMaps        │  │    
│ - Operator Logs         │    │ └─────────────────────┘  │    
│ - Configuration Files   │    │ OpenShift Resources      │    
│ - Event History         │    │ - OLM Components         │    
└─────────────────────────┘    │ - InstallPlans           │    
                               │ - Subscriptions          │    
                               └──────────────────────────┘    
```

## Prerequisites

- OpenShift cluster (4.11+)
- Tempo Operator installed via OLM
- Cluster administrator privileges for must-gather operations
- `oc` CLI access
- Understanding of OpenShift Routes and HAProxy configuration

## Step-by-Step Configuration

### Step 1: Deploy TempoMonolithic with OpenShift Route

Create a simple TempoMonolithic instance with route-enabled Jaeger UI:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: mono-route
  namespace: chainsaw-mono-route
spec:
  timeout: 2m
  jaegerui:
    enabled: true
    route:
      enabled: true
EOF
```

**Key Configuration Elements**:

#### Basic TempoMonolithic Setup
- `timeout: 2m`: Fast timeout for testing scenarios
- **In-Memory Storage**: Default storage for quick deployment
- **All-in-One**: Single pod deployment for simplicity

#### Jaeger UI with OpenShift Route
- `jaegerui.enabled: true`: Enables Jaeger query interface
- `route.enabled: true`: Creates OpenShift Route for external access
- **Automatic TLS**: OpenShift handles TLS termination

**Generated Resources**:
- **TempoMonolithic**: Primary custom resource
- **StatefulSet**: Tempo application deployment
- **Services**: Internal cluster services
- **Route**: External access endpoint
- **ConfigMaps**: Tempo configuration and CA bundles
- **ServiceAccount**: Tempo service account with appropriate permissions

**Reference**: [`install-tempo.yaml`](./install-tempo.yaml)

### Step 2: Validate Route Creation and Accessibility

Verify that the OpenShift Route is properly created and accessible:

```bash
# Check route creation
oc get route tempo-mono-route-jaegerui -n chainsaw-mono-route

# Get external URL
ROUTE_URL=$(oc get route tempo-mono-route-jaegerui -n chainsaw-mono-route -o jsonpath='{.spec.host}')
echo "Jaeger UI accessible at: https://$ROUTE_URL"

# Test external accessibility
curl -k https://$ROUTE_URL/

# Verify route configuration
oc describe route tempo-mono-route-jaegerui -n chainsaw-mono-route
```

**Route Configuration Details**:
- **Host**: Automatically generated hostname based on cluster domain
- **TLS Termination**: Edge termination with OpenShift's default certificates
- **Target Service**: Routes traffic to Jaeger UI service
- **Port**: Routes to Jaeger UI port (typically 16686)

### Step 3: Execute Must-Gather for Comprehensive Data Collection

Run the Tempo-specific must-gather tool to collect troubleshooting information:

```bash
# Create temporary directory for must-gather output
MUST_GATHER_DIR=$(mktemp -d)

# Determine operator namespace
TEMPO_NAMESPACE=$(oc get pods -A \
  -l control-plane=controller-manager \
  -l app.kubernetes.io/name=tempo-operator \
  -o jsonpath='{.items[0].metadata.namespace}')

# Execute must-gather
oc adm must-gather \
  --dest-dir=$MUST_GATHER_DIR \
  --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest \
  -- /usr/bin/must-gather --operator-namespace $TEMPO_NAMESPACE
```

**Must-Gather Execution Details**:

#### Must-Gather Image
- **Container Image**: `quay.io/rhn_support_ikanse/tempo-must-gather:latest`
- **Purpose**: Specialized data collection for Tempo Operator
- **Scope**: Comprehensive resource and configuration collection

#### Data Collection Process
1. **Operator Resources**: Controller deployment, RBAC, configurations
2. **OLM Components**: InstallPlans, ClusterServiceVersions, Subscriptions
3. **Tempo Resources**: All TempoMonolithic and related Kubernetes objects
4. **OpenShift Resources**: Routes, services, operator-specific objects
5. **Logs**: Operator and application logs for troubleshooting

**Reference**: [`check-must-gather.sh`](./check-must-gather.sh)

### Step 4: Validate Must-Gather Content Completeness

Verify that all required resources are collected in the must-gather output:

```bash
# Define required items for validation
REQUIRED_ITEMS=(
  "event-filter.html"                    # Event analysis dashboard
  "timestamp"                            # Collection timestamp
  "*sha*/deployment-tempo-operator-controller.yaml"  # Operator deployment
  "*sha*/olm/installplan-install-*.yaml"             # OLM install plans
  "*sha*/olm/clusterserviceversion-tempo-operator-*.yaml"  # CSV
  "*sha*/olm/operator-opentelemetry-product-openshift-opentelemetry-operator.yaml"  # OTel operator
  "*sha*/olm/operator-tempo-*-tempo-operator.yaml"   # Tempo operator OLM
  "*sha*/olm/subscription-tempo-*.yaml"              # OLM subscriptions
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/tempomonolithic-mono-route.yaml"  # Main resource
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/service-tempo-mono-route-jaegerui.yaml"  # Jaeger service
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/configmap-tempo-mono-route-serving-cabundle.yaml"  # CA bundle
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/statefulset-tempo-mono-route.yaml"  # Application
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/service-tempo-mono-route.yaml"  # Main service
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/route-tempo-mono-route-jaegerui.yaml"  # OpenShift Route
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/configmap-tempo-mono-route-config.yaml"  # Tempo config
  "*sha*/namespaces/chainsaw-mono-route/tempomonolithic/mono-route/serviceaccount-tempo-mono-route.yaml"  # Service account
  "*sha*/tempo-operator-controller-*"    # Operator logs and info
)

# Verify each required item exists
for item in "${REQUIRED_ITEMS[@]}"; do
  if ! find "$MUST_GATHER_DIR" -path "$MUST_GATHER_DIR/$item" -print -quit | grep -q .; then
    echo "Missing: $item"
    exit 1
  else
    echo "Found: $item"
  fi
done

echo "✓ All required must-gather items collected successfully"
```

**Must-Gather Content Validation**:

#### Core Operational Data
- **Event Filter Dashboard**: HTML dashboard for event analysis
- **Timestamp**: Collection execution time for correlation
- **Resource Definitions**: Complete YAML definitions of all related resources

#### Operator-Level Information
- **Controller Deployment**: Tempo operator deployment configuration
- **OLM Integration**: Complete OLM lifecycle components
- **RBAC Configuration**: Service accounts, roles, and bindings

#### Application-Level Resources
- **TempoMonolithic**: Primary custom resource with full specification
- **Workload Resources**: StatefulSets, services, configmaps
- **OpenShift Integration**: Routes, service CA bundles
- **Configuration**: Complete Tempo configuration and overrides

## OpenShift Route Features

### 1. **Route Configuration Options**

#### Basic Route Setup
```yaml
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
      # Uses default OpenShift route configuration
```

#### Advanced Route Configuration
```yaml
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
      host: tempo-ui.apps.cluster.example.com  # Custom hostname
      tls:
        termination: edge                        # TLS termination type
        insecureEdgeTerminationPolicy: Redirect # Redirect HTTP to HTTPS
      annotations:
        haproxy.router.openshift.io/timeout: 5m # Custom timeouts
        haproxy.router.openshift.io/rate-limit-connections: "true"
```

#### Route with Custom Certificates
```yaml
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
      tls:
        termination: edge
        certificate: |
          -----BEGIN CERTIFICATE-----
          # Custom certificate content
          -----END CERTIFICATE-----
        key: |
          -----BEGIN PRIVATE KEY-----
          # Private key content
          -----END PRIVATE KEY-----
        caCertificate: |
          -----BEGIN CERTIFICATE-----
          # CA certificate content
          -----END CERTIFICATE-----
```

### 2. **TLS Termination Strategies**

#### Edge Termination (Default)
```yaml
# TLS terminates at the router
spec:
  jaegerui:
    route:
      tls:
        termination: edge
        insecureEdgeTerminationPolicy: Redirect
```

#### Passthrough Termination
```yaml
# TLS terminates at the pod
spec:
  jaegerui:
    route:
      tls:
        termination: passthrough
  # Requires Tempo to handle TLS directly
```

#### Re-encryption Termination
```yaml
# TLS terminates at router, re-encrypts to pod
spec:
  jaegerui:
    route:
      tls:
        termination: reencrypt
        destinationCACertificate: |
          -----BEGIN CERTIFICATE-----
          # Destination CA certificate
          -----END CERTIFICATE-----
```

### 3. **Load Balancing and High Availability**

#### Multiple Replicas with Route
```yaml
spec:
  replicas: 3
  jaegerui:
    enabled: true
    route:
      enabled: true
      annotations:
        haproxy.router.openshift.io/balance: roundrobin
        haproxy.router.openshift.io/disable_cookies: "true"
```

#### Health Checks and Timeouts
```yaml
spec:
  jaegerui:
    route:
      annotations:
        haproxy.router.openshift.io/timeout: 30s
        haproxy.router.openshift.io/health-check-interval: 10s
        router.openshift.io/cookie_name: tempo-jaeger-ui
```

## Must-Gather Operational Features

### 1. **Comprehensive Data Collection**

#### Operator-Level Collection
- **Deployment Configuration**: Complete operator deployment YAML
- **RBAC Resources**: All roles, role bindings, service accounts
- **OLM Lifecycle**: InstallPlans, CSVs, Subscriptions
- **Custom Resource Definitions**: CRD specifications and status

#### Application-Level Collection
- **TempoMonolithic Resources**: All related Kubernetes objects
- **Configuration Data**: ConfigMaps, Secrets (sanitized)
- **Runtime Information**: Pod specifications, statuses
- **Networking**: Services, routes, ingress configurations

#### Platform Integration
- **OpenShift Routes**: Route configurations and status
- **Service CA Bundles**: Certificate authority information
- **Events**: Kubernetes events related to Tempo resources
- **Logs**: Operator and application logs

### 2. **Troubleshooting Information**

#### Resource Status Collection
```bash
# Must-gather collects detailed status information
oc get tempomonolithic mono-route -o yaml  # Complete resource status
oc describe tempomonolithic mono-route     # Detailed status and events
oc get events --field-selector involvedObject.name=mono-route  # Related events
```

#### Configuration Analysis
```bash
# Must-gather includes configuration analysis
oc get configmap tempo-mono-route-config -o yaml  # Tempo configuration
oc get secret tempo-mono-route-*              # Certificate and secret info
oc get route tempo-mono-route-jaegerui -o yaml   # Route configuration
```

#### Performance and Health Data
```bash
# Must-gather collects operational metrics
oc logs statefulset/tempo-mono-route          # Application logs
oc top pod tempo-mono-route-0                 # Resource utilization
oc get pod tempo-mono-route-0 -o yaml         # Pod specification and status
```

### 3. **Support Bundle Analysis**

#### Event Timeline Analysis
The must-gather includes an HTML event filter dashboard that provides:
- **Timeline View**: Chronological event sequence
- **Filtering Options**: Filter by resource type, severity, time range
- **Correlation**: Link events to specific resources and operations
- **Root Cause Analysis**: Identify event chains leading to issues

#### Resource Dependency Mapping
```bash
# Must-gather captures resource relationships
# - TempoMonolithic → StatefulSet → Pods
# - Services → Endpoints → Pods
# - Routes → Services → Pods
# - ConfigMaps → Pod volumes
```

## Production Deployment Considerations

### 1. **Route Security Configuration**

#### Secure Route Setup
```yaml
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
      host: secure-tempo.apps.prod.company.com
      tls:
        termination: edge
        insecureEdgeTerminationPolicy: Redirect
      annotations:
        haproxy.router.openshift.io/hsts_header: max-age=31536000;includeSubDomains;preload
        haproxy.router.openshift.io/ip_whitelist: 10.0.0.0/8 192.168.0.0/16
```

#### Authentication Integration
```yaml
# OAuth proxy integration
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
      annotations:
        haproxy.router.openshift.io/auth-type: oauth2
        haproxy.router.openshift.io/auth-url: https://oauth.company.com/auth
```

### 2. **Must-Gather Automation**

#### Automated Collection Scripts
```bash
#!/bin/bash
# Automated must-gather collection for production support

CLUSTER_NAME=$(oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}')
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT_DIR="/support-data/${CLUSTER_NAME}-${TIMESTAMP}"

# Run must-gather with automatic cleanup
oc adm must-gather \
  --dest-dir="$OUTPUT_DIR" \
  --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest \
  -- /usr/bin/must-gather --all-namespaces

# Create support bundle
tar -czf "${OUTPUT_DIR}.tar.gz" -C "$OUTPUT_DIR" .
rm -rf "$OUTPUT_DIR"

echo "Support bundle created: ${OUTPUT_DIR}.tar.gz"
```

#### Scheduled Health Checks
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: tempo-health-check
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: health-check
            image: quay.io/rhn_support_ikanse/tempo-must-gather:latest
            command:
            - /bin/bash
            - -c
            - |
              # Basic health check
              oc get tempomonolithic -A
              oc get route -l app.kubernetes.io/managed-by=tempo-operator
              
              # Collect must-gather if issues detected
              if ! oc get tempomonolithic -A --no-headers | grep -q Ready; then
                oc adm must-gather --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest
              fi
          restartPolicy: OnFailure
```

### 3. **Monitoring and Alerting**

#### Route Health Monitoring
```yaml
# Prometheus alerts for route health
alert: TempoRouteUnavailable
expr: probe_success{job="tempo-route-monitoring"} == 0
for: 2m
annotations:
  summary: "Tempo Jaeger UI route is unavailable"
  description: "External access to Tempo Jaeger UI via OpenShift route has failed"

alert: TempoRouteHighLatency
expr: probe_duration_seconds{job="tempo-route-monitoring"} > 5
for: 5m
annotations:
  summary: "High latency accessing Tempo route"
  description: "Tempo Jaeger UI route response time is above 5 seconds"
```

#### Must-Gather Trigger Automation
```bash
# Automatic must-gather on critical alerts
#!/bin/bash
# This script can be triggered by alert manager webhooks

ALERT_NAME="$1"
SEVERITY="$2"

if [[ "$SEVERITY" == "critical" ]]; then
  echo "Critical alert detected: $ALERT_NAME"
  echo "Collecting must-gather data..."
  
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  oc adm must-gather \
    --dest-dir="/tmp/critical-${ALERT_NAME}-${TIMESTAMP}" \
    --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest
    
  # Upload to support system or S3 bucket
  # Send notification to support team
fi
```

## Troubleshooting Route and Must-Gather Issues

### 1. **Route Connectivity Problems**

#### Route Configuration Issues
```bash
# Check route status
oc get route tempo-mono-route-jaegerui -o yaml

# Verify route admits
oc describe route tempo-mono-route-jaegerui | grep -A10 "Route Status"

# Test route resolution
nslookup $(oc get route tempo-mono-route-jaegerui -o jsonpath='{.spec.host}')

# Check router pods
oc get pods -n openshift-ingress -l ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default
```

#### TLS Certificate Issues
```bash
# Check certificate validity
openssl s_client -connect $(oc get route tempo-mono-route-jaegerui -o jsonpath='{.spec.host}'):443 -servername $(oc get route tempo-mono-route-jaegerui -o jsonpath='{.spec.host}')

# Verify certificate chain
curl -vvI https://$(oc get route tempo-mono-route-jaegerui -o jsonpath='{.spec.host}')

# Check router certificate configuration
oc get secret -n openshift-ingress | grep router-certs
```

### 2. **Must-Gather Collection Issues**

#### Image Pull Problems
```bash
# Check must-gather image availability
oc run test-must-gather --image=quay.io/rhn_support_ikanse/tempo-must-gather:latest --rm -it -- /bin/bash

# Verify image registry access
oc describe pod <must-gather-pod-name> | grep -A10 "Events:"

# Check cluster image registry configuration
oc get image.config/cluster -o yaml
```

#### Permission Issues
```bash
# Check must-gather service account permissions
oc adm policy who-can get pods --all-namespaces
oc adm policy who-can get secrets --all-namespaces

# Verify cluster-admin access for must-gather
oc auth can-i "*" "*" --as=system:admin
```

#### Resource Collection Failures
```bash
# Check for partial collection
find /must-gather-output -name "*.yaml" | wc -l

# Verify namespace access
oc get namespaces | grep tempo

# Check for resource access errors
grep -r "error\|failed\|denied" /must-gather-output/
```

## Related Configurations

- [Basic Route Configuration](../route/README.md) - TempoStack route setup
- [TempoStack External Access](../multitenancy/README.md) - Distributed external access
- [TLS Configuration](../tls-monolithic-singletenant/README.md) - Secure route setup

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift/monolithic-route
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test validates both route functionality and must-gather completeness. The must-gather validation ensures all critical resources are collected for production support scenarios. The test requires cluster administrator privileges for must-gather operations.

