# Grafana Tempo operator

This is a Kubernetes operator for [Grafana Tempo](https://github.com/grafana/tempo).


## Documentation

* [Operator documentation](https://tempo-operator.netlify.app)
* [Tempo CRD design](https://docs.google.com/document/d/1avSSf__R226l2b3hbcpXlYH7w6iKtXZsd9VTcpxDqng/edit)


## Deploy

1. Install cert-manager and minio: `make cert-manager deploy-minio`

2. Build and deploy operator:

```bash
IMG_PREFIX=docker.io/${USER} OPERATOR_VERSION=$(date +%s) make generate bundle docker-build docker-push deploy
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
