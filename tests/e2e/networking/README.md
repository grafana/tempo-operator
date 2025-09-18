# Tempo Operator Network Security Policies

This configuration blueprint demonstrates and validates the network security policies required for the Tempo Operator to function securely within Kubernetes clusters. This test ensures that proper network isolation and access controls are in place for the operator while maintaining necessary connectivity for its core functions.

## Overview

This test validates network security governance features:
- **Operator Network Isolation**: Restricts operator network access to essential communications only
- **API Server Connectivity**: Ensures operator can communicate with Kubernetes API
- **Metrics Endpoint Security**: Allows monitoring access while maintaining security
- **Webhook Security**: Secures admission webhook endpoints

## Architecture

```
┌─────────────────────┐    ┌──────────────────────────┐    ┌─────────────────────┐
│ Kubernetes API      │◀───│   Tempo Operator         │───▶│ Managed Resources   │
│ Server              │    │ ┌─────────────────────┐  │    │ - TempoStack        │
│ - Port 6443         │    │ │ Network Policies    │  │    │ - TempoMonolithic   │
└─────────────────────┘    │ │ ┌─────────────────┐ │  │    │ - ConfigMaps        │
                           │ │ │ Deny All       │ │  │    │ - Services          │
┌─────────────────────┐    │ │ │ (Default)      │ │  │    └─────────────────────┘
│ Monitoring System   │◀───│ │ └─────────────────┘ │  │
│ - Prometheus        │    │ │ ┌─────────────────┐ │  │
│ - Port 8443         │    │ │ │ API Access     │ │  │
└─────────────────────┘    │ │ │ (Port 6443)    │ │  │
                           │ │ └─────────────────┘ │  │
┌─────────────────────┐    │ │ ┌─────────────────┐ │  │
│ Kubernetes API      │◀───│ │ │ Webhook Access │ │  │
│ (Webhooks)          │    │ │ │ (Port 9443)    │ │  │
│ - Port 9443         │    │ │ └─────────────────┘ │  │
└─────────────────────┘    │ └─────────────────────┘  │
                           └──────────────────────────┘
```

## Prerequisites

- Kubernetes cluster with NetworkPolicy support
- Tempo Operator installed via OLM (Operator Lifecycle Manager)
- CNI that supports NetworkPolicy enforcement (Calico, Cilium, etc.)
- `kubectl` or `oc` CLI access
- Cluster administrator privileges

## Network Policy Components

### 1. **Default Deny-All Policy**

The foundational security policy that blocks all traffic by default:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-operator-deny-all
  namespace: tempo-operator-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: operator-lifecycle-manager
      app.kubernetes.io/name: tempo-operator
      app.kubernetes.io/part-of: tempo-operator
      control-plane: controller-manager
  policyTypes:
  - Ingress
  - Egress
```

**Security Benefits**:
- **Zero Trust Foundation**: Denies all traffic not explicitly allowed
- **Minimizes Attack Surface**: Reduces potential network-based vulnerabilities
- **Compliance**: Meets security hardening requirements
- **Audit Trail**: Clear documentation of allowed network paths

### 2. **API Server Egress Policy**

Allows the operator to communicate with the Kubernetes API server:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-operator-egress-to-apiserver
  namespace: tempo-operator-system
spec:
  egress:
  - ports:
    - port: 6443
      protocol: TCP
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: operator-lifecycle-manager
      app.kubernetes.io/name: tempo-operator
      app.kubernetes.io/part-of: tempo-operator
      control-plane: controller-manager
  policyTypes:
  - Egress
```

**Functionality Enabled**:
- **Resource Management**: Create, update, delete Tempo resources
- **Status Updates**: Report resource status back to Kubernetes
- **Event Recording**: Log operational events
- **Leader Election**: Coordinate multiple operator replicas

### 3. **Metrics Ingress Policy**

Allows monitoring systems to scrape operator metrics:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-operator-ingress-to-metrics
  namespace: tempo-operator-system
spec:
  ingress:
  - from:
    - namespaceSelector: {}
      podSelector: {}
    ports:
    - port: 8443
      protocol: TCP
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: operator-lifecycle-manager
      app.kubernetes.io/name: tempo-operator
      app.kubernetes.io/part-of: tempo-operator
      control-plane: controller-manager
  policyTypes:
  - Ingress
```

**Monitoring Access**:
- **Prometheus Integration**: Metrics endpoint accessible for scraping
- **Cluster-wide Access**: Any pod in any namespace can access metrics
- **Operator Health**: Monitor operator performance and resource usage
- **Custom Metrics**: Track Tempo-specific operational metrics

### 4. **Webhook Ingress Policy**

Allows Kubernetes API server to call admission webhooks:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tempo-operator-ingress-webhook
  namespace: tempo-operator-system
spec:
  ingress:
  - ports:
    - port: 9443
      protocol: TCP
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: operator-lifecycle-manager
      app.kubernetes.io/name: tempo-operator
      app.kubernetes.io/part-of: tempo-operator
      control-plane: controller-manager
  policyTypes:
  - Ingress
```

**Webhook Functionality**:
- **Validation Webhooks**: Validate TempoStack and TempoMonolithic configurations
- **Mutation Webhooks**: Apply default values and configuration transformations
- **Security Enforcement**: Prevent invalid or insecure configurations
- **API Compatibility**: Ensure backward compatibility during upgrades

## Test Execution and Validation

### Step 1: Discover Operator Namespace

The test dynamically discovers the Tempo Operator namespace:

```bash
# Find the operator namespace
TEMPO_NAMESPACE=$(kubectl get pods -A \
  -l control-plane=controller-manager \
  -l app.kubernetes.io/name=tempo-operator \
  -o jsonpath='{.items[0].metadata.namespace}')

echo "Tempo Operator is running in namespace: $TEMPO_NAMESPACE"
```

**Dynamic Discovery Benefits**:
- **Flexible Deployment**: Works regardless of operator installation method
- **Environment Agnostic**: Supports various cluster configurations
- **Automation Friendly**: No hardcoded namespace assumptions

### Step 2: Validate NetworkPolicy Existence

Verify that all required network policies are present:

```bash
# Check for deny-all policy
kubectl get networkpolicy tempo-operator-deny-all -n $TEMPO_NAMESPACE

# Check for API server egress policy
kubectl get networkpolicy tempo-operator-egress-to-apiserver -n $TEMPO_NAMESPACE

# Check for metrics ingress policy
kubectl get networkpolicy tempo-operator-ingress-to-metrics -n $TEMPO_NAMESPACE

# Check for webhook ingress policy
kubectl get networkpolicy tempo-operator-ingress-webhook -n $TEMPO_NAMESPACE
```

### Step 3: Test Network Connectivity

Validate that essential connectivity is maintained:

```bash
# Test API server connectivity
kubectl exec deployment/tempo-operator-controller-manager -n $TEMPO_NAMESPACE -- \
  curl -k https://kubernetes.default.svc:443/api/v1 -w "%{http_code}\n" -o /dev/null

# Test metrics endpoint
kubectl port-forward deployment/tempo-operator-controller-manager 8443:8443 -n $TEMPO_NAMESPACE &
curl -k https://localhost:8443/metrics

# Test webhook endpoint
curl -k -X POST https://localhost:9443/validate-tempo-grafana-com-v1alpha1-tempostack \
  -H "Content-Type: application/json" \
  -d '{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview"}'
```

## Security Policy Configuration

### 1. **Pod Selector Labels**

All network policies use consistent label selectors:

```yaml
podSelector:
  matchLabels:
    app.kubernetes.io/managed-by: operator-lifecycle-manager
    app.kubernetes.io/name: tempo-operator
    app.kubernetes.io/part-of: tempo-operator
    control-plane: controller-manager
```

**Label Purpose**:
- `app.kubernetes.io/managed-by: operator-lifecycle-manager`: OLM-managed operator
- `app.kubernetes.io/name: tempo-operator`: Identifies the specific operator
- `app.kubernetes.io/part-of: tempo-operator`: Groups related components
- `control-plane: controller-manager`: Identifies controller pods

### 2. **Traffic Direction Control**

Each policy specifies explicit traffic direction:

```yaml
# Ingress-only policy
policyTypes:
- Ingress

# Egress-only policy  
policyTypes:
- Egress

# Bidirectional policy (deny-all)
policyTypes:
- Ingress
- Egress
```

### 3. **Port-Specific Access**

Network policies define specific ports for precise access control:

- **Port 6443**: Kubernetes API server (HTTPS)
- **Port 8443**: Operator metrics endpoint (HTTPS)
- **Port 9443**: Admission webhook server (HTTPS)

## Advanced Network Security Configurations

### 1. **Namespace-Based Isolation**

Restrict metrics access to specific namespaces:

```yaml
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring-system
    ports:
    - port: 8443
      protocol: TCP
```

### 2. **Source IP Restrictions**

Limit access based on source IP ranges:

```yaml
spec:
  ingress:
  - from:
    - ipBlock:
        cidr: 10.0.0.0/8
        except:
        - 10.0.1.0/24
    ports:
    - port: 8443
      protocol: TCP
```

### 3. **DNS Policy Integration**

Control DNS resolution for enhanced security:

```yaml
spec:
  egress:
  - to: []
    ports:
    - port: 53
      protocol: UDP
    - port: 53
      protocol: TCP
  - ports:
    - port: 6443
      protocol: TCP
```

## Troubleshooting Network Issues

### 1. **NetworkPolicy Not Working**

#### Check CNI Support
```bash
# Verify CNI supports NetworkPolicy
kubectl get nodes -o wide
kubectl describe node <node-name> | grep -i "Container Runtime\|Network Plugin"

# Test with a simple deny-all policy
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-deny-all
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
EOF
```

#### Validate Policy Application
```bash
# Check if policies are applied
kubectl get networkpolicy -A

# Describe specific policy
kubectl describe networkpolicy tempo-operator-deny-all -n $TEMPO_NAMESPACE

# Check CNI logs (example for Calico)
kubectl logs -n kube-system -l k8s-app=calico-node
```

### 2. **Operator Connectivity Issues**

#### API Server Access Problems
```bash
# Check operator logs for API errors
kubectl logs deployment/tempo-operator-controller-manager -n $TEMPO_NAMESPACE | grep -i "connection refused\|timeout\|401\|403"

# Test direct API access
kubectl exec deployment/tempo-operator-controller-manager -n $TEMPO_NAMESPACE -- \
  curl -v -k https://kubernetes.default.svc:443/api/v1

# Verify service account permissions
kubectl auth can-i "*" "*" --as=system:serviceaccount:$TEMPO_NAMESPACE:tempo-operator-controller-manager
```

#### Webhook Failures
```bash
# Check webhook configuration
kubectl get validatingwebhookconfiguration
kubectl get mutatingwebhookconfiguration

# Test webhook connectivity
kubectl get events --field-selector reason=FailedAdmissionWebhook

# Check webhook certificate
kubectl get secret -n $TEMPO_NAMESPACE | grep webhook
```

### 3. **Metrics Collection Issues**

#### Prometheus Scraping Problems
```bash
# Check if metrics endpoint is accessible
kubectl port-forward deployment/tempo-operator-controller-manager 8443:8443 -n $TEMPO_NAMESPACE &
curl -k https://localhost:8443/metrics

# Verify ServiceMonitor configuration
kubectl get servicemonitor -A | grep tempo

# Check Prometheus configuration
kubectl get prometheus -A -o yaml | grep -A10 -B10 tempo
```

## Production Deployment Considerations

### 1. **CNI Selection and Configuration**

#### Recommended CNI Solutions
- **Calico**: Full NetworkPolicy support with advanced features
- **Cilium**: eBPF-based networking with rich policy capabilities
- **Antrea**: VMware-supported CNI with comprehensive NetworkPolicy support

#### CNI Configuration Validation
```bash
# Verify CNI installation
kubectl get daemonset -n kube-system

# Check CNI configuration
kubectl get configmap -n kube-system | grep -E "(calico|cilium|antrea)"

# Test NetworkPolicy enforcement
kubectl apply -f https://raw.githubusercontent.com/ahmetb/kubernetes-network-policy-recipes/master/01-deny-all-traffic-to-an-application.yaml
```

### 2. **Security Monitoring and Alerting**

#### Network Policy Violations
```bash
# Monitor CNI logs for policy violations
kubectl logs -n kube-system -l k8s-app=calico-node | grep -i "denied\|dropped"

# Set up alerts for policy violations
# Example Prometheus alert rule
alert: NetworkPolicyViolation
expr: increase(calico_denied_packets_total[5m]) > 0
for: 1m
annotations:
  summary: "Network policy violation detected"
```

#### Compliance Validation
```bash
# Regular policy compliance checks
kubectl get networkpolicy -A --no-headers | wc -l
kubectl get networkpolicy -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,POLICY_TYPES:.spec.policyTypes

# Audit network access patterns
kubectl get events --field-selector type=Warning | grep -i network
```

### 3. **Policy Maintenance and Updates**

#### Version Control Integration
```yaml
# GitOps approach for network policies
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: tempo-operator-network-policies
spec:
  source:
    repoURL: https://github.com/company/tempo-operator-policies
    path: network-policies/
  destination:
    server: https://kubernetes.default.svc
    namespace: tempo-operator-system
```

#### Automated Testing
```bash
# Network policy testing in CI/CD
#!/bin/bash
# Apply test policies
kubectl apply -f test-network-policies.yaml

# Run connectivity tests
kubectl run test-pod --image=nicolaka/netshoot --rm -it -- \
  sh -c "nc -zv tempo-operator-controller-manager 8443"

# Cleanup test resources
kubectl delete -f test-network-policies.yaml
```

## Related Security Configurations

- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Network Policy Recipes](https://github.com/ahmetb/kubernetes-network-policy-recipes)

## Test Execution

To run this test manually:

```bash
chainsaw test --test-dir ./tests/e2e/networking
```

**Test Definition**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)

**Note**: This test requires a Kubernetes cluster with NetworkPolicy support and the Tempo Operator installed via OLM. The test validates the presence of required network policies but does not test actual network isolation enforcement, which depends on the CNI implementation.

