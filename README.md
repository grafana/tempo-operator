# Grafana Tempo operator

This is a Kubernetes operator for [Grafana Tempo](https://github.com/grafana/tempo).

**The project is in active development and subject to large changes.**

## Docs

* [Tempo CRD design](https://docs.google.com/document/d/1avSSf__R226l2b3hbcpXlYH7w6iKtXZsd9VTcpxDqng/edit#)


## Deploy 

```yaml
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: Microservices
metadata:
  name: simplest
spec:
  limits:
    global: {}
    perTenant: {}
  retention:
    global: {}
    perTenant: {}
EOF
```

## Community 

* [Grafana Slack #tempo-operator channel](https://grafana.slack.com/archives/C0414EUU39A)
