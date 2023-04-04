---
title: Quick Start
description: Quick Start
lead: ""
lastmod: "2021-03-08T08:48:57+00:00"
draft: false
images: []
menu:
  docs:
    parent: prologue
weight: 200
toc: true
---

One page summary on how to start with Tempo Operator and TempoStack.

## Requirements

The easiest way to start with the Tempo Operator is to use Kubernetes [kind](sigs.k8s.io/kind).

## Deploy

To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/) and run:

```shell
kubectl apply -f https://github.com/os-observability/tempo-operator/releases/latest/download/tempo-operator.yaml
```

Once you have the operator deployed you need to install a storage backend. For this quick start guide  we will install [`minio`](https://min.io/) as follows:

```shell
kubectl apply -f https://raw.githubusercontent.com/os-observability/tempo-operator/main/minio.yaml
```

After minio was deployed, create a secret for minio in the namespace you are using:

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

Then create Tempo CR:

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

After create the `TempoStack` CR you should see a some pods on the namespace, wait until the stack to stabilized.

The stack deployed above is configured to receive jaeger Thrift HTTP and OTLP, also because the jaeger query is enable you can use the jaeger UI to inspect the data.

In order to do a quick test we will deploy a Job that will generate some traces.

```yaml
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: tracegen
spec:
  template:
    spec:
      containers:
        - name: tracegen
          image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/tracegen:latest
          command:
            - "./tracegen"
          args:
            - -otlp-endpoint=tempo-simplest-distributor:4317
            - -otlp-insecure
            - -duration=30s
            - -workers=1
      restartPolicy: Never
  backoffLimit: 4
EOF
```

Then to see the traces we can forward the jaeger query port

```
kubectl port-forward svc/tempo-simplest-query-frontend 16686:16686
```

And visit http://localhost:16686.