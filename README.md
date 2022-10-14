# Grafana Tempo operator

This is a Kubernetes operator for [Grafana Tempo](https://github.com/grafana/tempo).

**The project is in active development and subject to large changes.**

## Docs

* [Tempo CRD design](https://docs.google.com/document/d/1avSSf__R226l2b3hbcpXlYH7w6iKtXZsd9VTcpxDqng/edit#)


## Deploy 

1. Deploy object storage `kubectl apply -f minio.yaml`

2. Build and deploy operator:

```bash
IMG=docker.io/${USER}/tempo-operator:dev-$(date +%s) make generate bundle docker-build docker-push deploy
``` 

3.Create Tempo CR:

```yaml
kubectl apply -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: Microservices
metadata:
  name: simplest
spec:
  storage:
    secret: test
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```

## Community 

* [Grafana Slack #tempo-operator channel](https://grafana.slack.com/archives/C0414EUU39A)
