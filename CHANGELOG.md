Changes by Version
==================

<!-- next version -->

## 0.2.0

### 🛑 Breaking changes 🛑

- `operator`: Rename operator deployment to enable upgrading from 0.1.0 (#432)

  If you have installed the operator via Kubernetes manifests, please run `kubectl -n tempo-operator-system delete deployment tempo-operator-controller-manager` to prune the old deployment.
  If you have installed the operator via OLM, no action is required.


### 💡 Enhancements 💡

- `operator`: Add support for Kubernetes 1.26 and 1.27. (#385, #365)
- `operator`: Configure logging (#217)
- `tests`: Add a smoketest for tempo + opentelemetry-collector + multitenancy (OpenShift) (#202)
- `operator`: Add mTLS support to the communication between gateway and internal components. (#240)
- `operator`: Create ServiceMonitors for Tempo components (#298, #333)
- `operator`: Add operator metrics (#308, #334)
- `operator`: Recover the resource.requests field for the operator manager as the OpenShift guidelines recommend (#426)
- `operator`: add tempo gateway to resource pool, when is enable it will take into account the gateway in the resource calculation. (#201)
- `operator`: Sanitize generated manifest names (#223)
- `operator`: Create one TLS cert/key per component/service instead of having different certs for HTTP and GRPC (#383)
- `operator`: Introducing alerts for operands (#307)

### Components
- Tempo: [v2.0.1](https://github.com/grafana/tempo/releases/tag/v2.0.1)

## 0.1.0

### 🚀 New components 🚀

- `operator`: Initial release of tempo operator
  - Supports [Tempo - v2.0.1](https://github.com/grafana/tempo/releases/tag/v2.0.1)
  
