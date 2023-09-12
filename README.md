# Grafana Tempo operator

This is a Kubernetes operator for [Grafana Tempo](https://github.com/grafana/tempo).


## Features

* **Resource Limits** - Specify overall resource requests and limits in the `TempoStack` CR; the operator assigns fractions of it to each component
* **AuthN and AuthZ** - Supports OpenID Control (OIDC) and role-based access control (RBAC)
* **Managed upgrades** - Updating the operator will automatically update all managed Tempo clusters
* **Multitenancy** - Multiple tenants can send traces to the same Tempo cluster
* **mTLS** - Communication between the Tempo components can be secured via mTLS
* **Jaeger UI** - Traces can be visualized in Jaeger UI and exposed via Ingress or OpenShift Route
* **Observability** - The operator and `TempoStack` operands expose telemetry (metrics, traces) and integrate with Prometheus `ServiceMonitor` and `PrometheusRule`


## Documentation

* [Operator documentation](https://grafana.com/docs/tempo/next/setup/operator/)
* [Tempo CRD design](https://docs.google.com/document/d/1avSSf__R226l2b3hbcpXlYH7w6iKtXZsd9VTcpxDqng/edit)


## Deploy

1. Install cert-manager and minio: `make cert-manager deploy-minio`

2. Build and deploy operator:

```bash
IMG_PREFIX=docker.io/${USER} OPERATOR_VERSION=$(date +%s).0.0 make docker-build docker-push deploy
``` 

3. Create a secret for minio in the namespace you are using:
```yaml
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
4. Create Tempo CR:

```yaml
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simplest
spec:
  storage:
    secret:
      name: minio-test
      type: s3
  storageSize: 1Gi
  resources:
    total:
      limits:
        memory: 2Gi
        cpu: 2000m
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```


## Community

* [Grafana Slack #tempo-operator](https://grafana.slack.com/archives/C0414EUU39A)
