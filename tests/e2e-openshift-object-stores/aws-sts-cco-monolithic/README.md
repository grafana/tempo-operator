# TempoMonolithic with AWS STS and OpenShift Cloud Credential Operator (CCO)

This configuration blueprint demonstrates TempoMonolithic deployment with AWS STS authentication using OpenShift's Cloud Credential Operator (CCO) for enterprise-grade credential management. This setup provides the most secure and automated approach to cloud credentials in OpenShift, leveraging CCO's advanced credential lifecycle management and operator-level security controls.

## Overview

This test validates OpenShift CCO integration with AWS STS featuring:
- **Cloud Credential Operator**: OpenShift's native cloud credential management system
- **Operator-Level Security**: CCO manages credentials at the operator level for enhanced security
- **Advanced STS Integration**: CCO-managed STS token lifecycle with automatic renewal
- **Enterprise Compliance**: Meets enterprise security standards with automated credential rotation
- **Operator Subscription Management**: Dynamic operator configuration for role-based access

## Architecture

```
┌─────────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────────┐
│ OpenShift CCO           │───▶│   Tempo Operator         │───▶│ TempoMonolithic         │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │    │ ┌─────────────────────┐ │
│ │ Credential          │ │    │ │ ROLEARN Environment │  │    │ │ S3 Storage          │ │
│ │ Management          │ │    │ │ Variable            │  │    │ │ - AWS STS Auth      │ │
│ │ - Role ARN          │ │    │ │ - Subscription      │  │    │ │ - token-cco Mode    │ │
│ │ - Token Refresh     │ │    │ │ - Patch             │  │    │ └─────────────────────┘ │
│ └─────────────────────┘ │    │ └─────────────────────┘  │    └─────────────────────────┘
└─────────────────────────┘    └──────────────────────────┘
          │                              ▲
          │ ┌────────────────────────────┴─────────────────────────┐
          │ │           AWS STS Service                            │
          └▶│ ┌─────────────────────┐ ┌─────────────────────┐     │
            │ │ AssumeRoleWithWeb   │ │ Temporary           │     │
            │ │ Identity            │ │ Credentials         │     │
            │ └─────────────────────┘ └─────────────────────┘     │
            │                                  ▲                  │
            └──────────────────────────────────┼──────────────────┘
                                               │
┌─────────────────────────┐    ┌──────────────────────────┐
│ OpenShift OIDC          │───▶│    AWS S3 Bucket         │
│ ┌─────────────────────┐ │    │ ┌─────────────────────┐  │
│ │ Service Account     │ │    │ │ Trace Storage       │  │
│ │ JWT Tokens          │ │    │ │ - Bucket Policy     │  │
│ │ - Federated         │ │    │ │ - Cross-Region      │  │
│ │ - Identity          │ │    │ │ - Lifecycle         │  │
│ └─────────────────────┘ │    │ └─────────────────────┘  │
└─────────────────────────┘    └──────────────────────────┘

CCO Management Flow:
1. CCO detects AWS STS requirement
2. Operator subscription patched with ROLEARN
3. Operator restarted with new environment
4. TempoMonolithic uses token-cco mode
5. CCO manages credential lifecycle
```

## Prerequisites

- OpenShift cluster (4.10+) with Cloud Credential Operator
- AWS account with CCO-compatible IAM permissions
- Tempo Operator installed via OLM
- Understanding of OpenShift CCO and subscription management
- AWS CLI configured (for CI environments)

## Step-by-Step Configuration

### Step 1: Create AWS Infrastructure with CCO Integration

Execute the CCO-specific AWS setup script:

```bash
./aws-sts-s3-create.sh tmonocco chainsaw-awscco-mono
```

**Script Functionality from [`aws-sts-s3-create.sh`](./aws-sts-s3-create.sh)**:

#### OpenShift Build Namespace Detection
```bash
if [ -z "${OPENSHIFT_BUILD_NAMESPACE+x}" ]; then
    OPENSHIFT_BUILD_NAMESPACE="cioptmonocco"
    export OPENSHIFT_BUILD_NAMESPACE
fi
```
- **CI Integration**: Automatically detects OpenShift CI build namespace
- **Unique Naming**: Ensures unique resource names across CI builds
- **Environment Isolation**: Prevents resource conflicts in CI environment

#### S3 Bucket Creation
```bash
bucket_name=tracing-$tempo_ns-$OPENSHIFT_BUILD_NAMESPACE
aws s3api create-bucket --bucket $bucket_name --region $region \
  --create-bucket-configuration LocationConstraint=$region
```
- **Naming Convention**: `tracing-{namespace}-{build-namespace}`
- **Region Configuration**: Creates bucket in specified region (us-east-2)
- **Cross-Region Support**: Configurable for different AWS regions

#### CCO-Compatible IAM Trust Relationship
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:aws:iam::{account}:oidc-provider/{oidc-provider}"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "{oidc-provider}:sub": [
          "system:serviceaccount:{namespace}:tempo-{name}",
          "system:serviceaccount:{namespace}:tempo-{name}-query-frontend"
        ]
      }
    }
  }]
}
```

#### Operator Subscription Patching
```bash
oc patch subscription "$TEMPO_OPERATOR_SUB" -n "$TEMPO_OPERATOR_NAMESPACE" \
    --type='merge' -p '{"spec": {"config": {"env": [{"name": "ROLEARN", "value": "'"$role_arn"'"}]}}}'
```
- **Dynamic Configuration**: Injects IAM role ARN into operator environment
- **Operator Restart**: Triggers operator restart with new configuration
- **CCO Integration**: Enables CCO-managed credential mode

#### CSV Validation
```bash
if oc -n "$TEMPO_OPERATOR_NAMESPACE" describe csv \
    --selector=operators.coreos.com/tempo-operator.openshift-tempo-operator= | \
    tail -n 1 | grep -qi "InstallSucceeded"; then
    echo "CSV updated successfully, continuing script execution..."
else
    echo "Operator CSV update failed, exiting with error."
    exit 1
fi
```
- **Operator Health**: Validates operator restart and CSV status
- **Configuration Verification**: Ensures ROLEARN environment variable is applied
- **Failure Detection**: Exits with error if operator update fails

### Step 2: Deploy TempoMonolithic with CCO Credential Mode

Apply TempoMonolithic with CCO-specific configuration:

```bash
oc apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: tmonocco
  namespace: chainsaw-awscco-mono
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: aws-sts
        credentialMode: token-cco
  jaegerui:
    enabled: true
    route:
      enabled: true
EOF
```

**Key CCO Configuration Elements**:

#### CCO Credential Mode
- `credentialMode: token-cco`: Enables Cloud Credential Operator management
- **Operator-Level Security**: CCO manages credentials at operator level
- **Automatic Rotation**: CCO handles token lifecycle and renewal
- **Enhanced Security**: No credential exposure to individual pods

#### S3 Storage Configuration
- `backend: s3`: Specifies AWS S3 as storage backend
- `secret: aws-sts`: References CCO-managed credential secret
- **Bucket Information**: Secret contains bucket name, region, and role ARN

#### External Access
- `jaegerui.enabled: true`: Enables Jaeger UI for trace visualization
- `route.enabled: true`: Creates OpenShift Route for external access

**Reference**: [`install-monolithic.yaml`](./install-monolithic.yaml)

### Step 3: Verify CCO-Managed Deployment

Wait for TempoMonolithic to initialize with CCO credential management:

```bash
oc get --namespace chainsaw-awscco-mono tempomonolithic tmonocco \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
# Should return: True
```

#### CCO Integration Validation
```bash
# Check operator environment variables
oc get deployment -n openshift-tempo-operator tempo-operator-controller \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROLEARN")].value}'

# Verify subscription patch
oc get subscription -n openshift-tempo-operator \
  -o jsonpath='{.items[0].spec.config.env[?(@.name=="ROLEARN")].value}'

# Check CCO-managed secret
oc get secret aws-sts -o yaml
```

### Step 4: Generate Traces with CCO Authentication

Create traces to validate CCO-managed S3 storage:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces
  namespace: chainsaw-awscco-mono
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.92.0
        args:
        - traces
        - --otlp-endpoint=tempo-tmonocco:4317
        - --otlp-insecure
        - --traces=10
      restartPolicy: Never
  backoffLimit: 4
EOF
```

**Trace Flow with CCO**:
1. **Trace Generation**: telemetrygen creates sample traces
2. **Ingestion**: TempoMonolithic receives traces via OTLP
3. **CCO Authentication**: CCO provides STS credentials to operator
4. **S3 Storage**: Traces stored in S3 with CCO-managed authentication
5. **Credential Rotation**: CCO automatically refreshes credentials

**Reference**: [`generate-traces.yaml`](./generate-traces.yaml)

### Step 5: Verify CCO-Authenticated Trace Storage

Validate that traces are properly stored via CCO-managed S3 access:

```bash
oc apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
  namespace: chainsaw-awscco-mono
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
          curl -v -G \
            http://tempo-tmonocco:3200/api/search \
            --data-urlencode "q={}" | tee /tmp/tempo.out
          
          num_traces=\$(jq ".traces | length" /tmp/tempo.out)
          if [[ "\$num_traces" -ne 10 ]]; then
            echo "Expected 10 traces, got \$num_traces"
            exit 1
          fi
          echo "✓ Successfully verified \$num_traces traces via CCO-managed S3"
      restartPolicy: Never
EOF
```

**Reference**: [`verify-traces.yaml`](./verify-traces.yaml)

### Step 6: Cleanup CCO-Managed Resources

Clean up AWS resources and operator configuration:

```bash
./aws-sts-s3-delete.sh tmonocco chainsaw-awscco-mono
```

**Cleanup Process**:
- **S3 Bucket Deletion**: Removes test S3 bucket and objects
- **IAM Role Cleanup**: Deletes created IAM role and policies
- **Operator Reset**: Resets operator subscription to original state
- **Secret Cleanup**: Removes CCO-managed secrets

**Reference**: [`aws-sts-s3-delete.sh`](./aws-sts-s3-delete.sh)

## Cloud Credential Operator (CCO) Features

### 1. **Credential Lifecycle Management**

#### Operator-Level Credential Injection
```bash
# CCO injects credentials at operator level
ROLEARN environment variable → Tempo Operator
↓
Operator uses role for all TempoMonolithic instances
↓
Enhanced security: no pod-level credential exposure
```

#### Automatic Credential Rotation
- **Token Refresh**: CCO automatically refreshes STS tokens before expiration
- **Operator Awareness**: Operator automatically detects credential changes
- **Zero Downtime**: Credential rotation without service interruption
- **Audit Trail**: Complete credential usage tracking

### 2. **Security Model**

#### Operator-Scoped Access
```yaml
# Traditional pod-level credentials (NOT used in CCO mode)
apiVersion: v1
kind: Pod
spec:
  serviceAccountName: tempo-pod-sa  # Individual pod credentials

# CCO operator-level credentials (USED in CCO mode)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tempo-operator-controller
spec:
  template:
    spec:
      env:
      - name: ROLEARN
        value: "arn:aws:iam::account:role/tempo-role"
```

#### Reduced Attack Surface
- **Centralized Management**: Single point of credential control
- **No Pod Exposure**: Credentials never exposed to individual pods
- **Operator Trust**: Only operator has direct AWS access
- **Audit Simplification**: Centralized credential usage logging

### 3. **Enterprise Integration**

#### CI/CD Pipeline Integration
```bash
# CCO works seamlessly with OpenShift CI
export OPENSHIFT_BUILD_NAMESPACE="ci-build-12345"
./aws-sts-s3-create.sh tempo-instance namespace
# CCO automatically manages credentials for CI environment
```

#### Multi-Tenant Security
```yaml
# CCO supports multiple TempoMonolithic instances
# All sharing same operator-level credentials
spec:
  credentialMode: token-cco  # Shared CCO management
```

## Advanced CCO Configuration

### 1. **Custom Credential Modes**

#### Token-CCO vs Direct Credentials
```yaml
# CCO-managed credentials (recommended)
spec:
  storage:
    traces:
      s3:
        credentialMode: token-cco

# Direct STS credentials (alternative)
spec:
  storage:
    traces:
      s3:
        credentialMode: token

# Static credentials (not recommended)
spec:
  storage:
    traces:
      s3:
        credentialMode: static
```

#### Hybrid Credential Management
```yaml
# Different credentials for different components
spec:
  storage:
    traces:
      s3:
        credentialMode: token-cco
  # Query components can use different credentials if needed
```

### 2. **Operator Subscription Management**

#### Dynamic Role Assignment
```bash
# Update operator with new role
NEW_ROLE_ARN="arn:aws:iam::account:role/new-tempo-role"
oc patch subscription tempo-operator -n openshift-tempo-operator \
  --type='merge' -p '{
    "spec": {
      "config": {
        "env": [{
          "name": "ROLEARN",
          "value": "'$NEW_ROLE_ARN'"
        }]
      }
    }
  }'
```

#### Multi-Environment Support
```yaml
# Different environments with different roles
# Development
ROLEARN: "arn:aws:iam::dev-account:role/tempo-dev-role"
# Production  
ROLEARN: "arn:aws:iam::prod-account:role/tempo-prod-role"
```

### 3. **Monitoring and Observability**

#### CCO Credential Metrics
```bash
# Monitor credential refresh cycles
oc logs -n openshift-cloud-credential-operator \
  deployment/cloud-credential-operator | grep tempo

# Check operator credential status
oc get deployment -n openshift-tempo-operator tempo-operator-controller \
  -o jsonpath='{.spec.template.spec.containers[0].env}'
```

#### AWS STS Usage Tracking
```bash
# Monitor STS API calls
aws logs filter-log-events \
  --log-group-name /aws/cloudtrail \
  --filter-pattern '{ $.eventSource = "sts.amazonaws.com" && $.userIdentity.arn = "*tempo*" }'
```

## Production Deployment Considerations

### 1. **CCO Security Best Practices**

#### Least Privilege IAM Policies
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket"
    ],
    "Resource": [
      "arn:aws:s3:::tempo-bucket/*",
      "arn:aws:s3:::tempo-bucket"
    ]
  }]
}
```

#### Trust Relationship Hardening
```json
{
  "Condition": {
    "StringEquals": {
      "oidc-provider:aud": "openshift",
      "oidc-provider:sub": [
        "system:serviceaccount:tempo-namespace:tempo-sa"
      ]
    },
    "StringLike": {
      "oidc-provider:sub": "system:serviceaccount:tempo-*:tempo-*"
    }
  }
}
```

### 2. **High Availability with CCO**

#### Multi-Region Credential Management
```bash
# CCO supports multi-region deployments
PRIMARY_REGION="us-east-1"
SECONDARY_REGION="us-west-2"

# Same IAM role works across regions
ROLEARN="arn:aws:iam::account:role/tempo-global-role"
```

#### Disaster Recovery
```yaml
# CCO credentials work for cross-region backup
spec:
  storage:
    traces:
      s3:
        bucket: tempo-primary-bucket
        region: us-east-1
        credentialMode: token-cco
  # Backup configuration automatically inherits credentials
```

### 3. **Compliance and Governance**

#### Audit Requirements
```bash
# CCO provides comprehensive audit trail
# - Operator subscription changes
# - Credential refresh cycles  
# - AWS STS API usage
# - S3 access patterns
```

#### Regulatory Compliance
```yaml
# CCO supports compliance frameworks
# - SOX: Automated credential rotation
# - PCI: No stored credentials
# - HIPAA: Encrypted credential transport
# - SOC 2: Complete audit trail
```

## Troubleshooting CCO Integration

### 1. **Operator Subscription Issues**

#### Subscription Patch Failures
```bash
# Check subscription status
oc get subscription -n openshift-tempo-operator

# Verify subscription patch
oc describe subscription tempo-operator -n openshift-tempo-operator

# Check CSV status after patch
oc get csv -n openshift-tempo-operator
```

#### Environment Variable Validation
```bash
# Verify ROLEARN in operator deployment
oc get deployment -n openshift-tempo-operator tempo-operator-controller \
  -o jsonpath='{.spec.template.spec.containers[0].env}' | jq '.'

# Check operator logs for role usage
oc logs -n openshift-tempo-operator deployment/tempo-operator-controller | grep -i role
```

### 2. **CCO Credential Problems**

#### Credential Mode Validation
```bash
# Check TempoMonolithic configuration
oc get tempomonolithic tmonocco -o yaml | grep credentialMode

# Verify secret contains CCO fields
oc get secret aws-sts -o jsonpath='{.data}' | base64 -d

# Test AWS access with CCO credentials
oc exec deployment/tempo-tmonocco -- \
  aws sts get-caller-identity --region us-east-2
```

#### AWS Trust Relationship Issues
```bash
# Verify OIDC provider registration
aws iam list-open-id-connect-providers

# Check role trust relationship
aws iam get-role --role-name tracing-chainsaw-awscco-mono-cioptmonocco

# Test role assumption
aws sts assume-role-with-web-identity \
  --role-arn $ROLE_ARN \
  --role-session-name test-session \
  --web-identity-token $JWT_TOKEN
```

### 3. **Performance and Reliability**

#### Credential Refresh Monitoring
```bash
# Monitor credential refresh patterns
oc logs -n openshift-tempo-operator deployment/tempo-operator-controller | \
  grep -i "credential\|token\|refresh"

# Check for credential expiration warnings
oc get events --field-selector reason=CredentialExpiring
```

#### S3 Access Performance
```bash
# Monitor S3 operation latency
oc port-forward deployment/tempo-tmonocco 3200:3200 &
curl http://localhost:3200/metrics | grep s3_operation_duration

# Check for S3 throttling
oc logs deployment/tempo-tmonocco | grep -i "throttl\|rate.limit"
```

## Related Configurations

- [AWS STS TempoStack](../aws-sts-tempostack/README.md) - Distributed STS deployment
- [AWS STS without CCO](../aws-sts-monolithic/README.md) - Direct STS integration
- [Azure CCO Integration](../azure-wif-monolithic/README.md) - Azure equivalent

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e-openshift-object-stores/aws-sts-cco-monolithic
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires OpenShift CI environment with CCO capabilities and runs sequentially (`concurrent: false`) due to operator restart requirements. The test demonstrates enterprise-grade cloud credential management using OpenShift's advanced CCO system.

