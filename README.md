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

3. Create a secret for minio in the namespace you are using:
```yaml
kubectl apply --namespace ${NAMESPACE} -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: minio-test
stringData:
  endpoint: http://minio.minio-storage.svc:9000
  bucket: tempo
  access_key_id: tempo
  access_key_secret: supersecret
type: Opaque
EOF

```
4. Create Tempo CR:

```yaml
kubectl apply --namespace ${NAMESPACE} -f - <<EOF
apiVersion: tempo.grafana.com/v1alpha1
kind: Microservices
metadata:
  name: simplest
spec:
  storage:
    secret: minio-test
  template:
    queryFrontend:
      jaegerQuery:
        enabled: true
EOF
```

## Community 

* [Grafana Slack #tempo-operator channel](https://grafana.slack.com/archives/C0414EUU39A)
