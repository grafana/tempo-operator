---
title: "Observability Tracing"
description: "Configure tracing of Operands."
lead: ""
date: 2023-03-26T08:48:45+00:00
lastmod: 2023-03-26T08:48:45+00:00
draft: false
images: []
menu:
  docs:
    parent: "tempostack"
weight: 100
toc: true
---

All tempo components as well as the tempo gateway ([observatorium-api](https://github.com/observatorium/api)) support the export of traces in `thrift_compact` format.


## Configure tracing of Operands

### Requirements

* A [OpenTelemetry Operator](https://opentelemetry.io/docs/k8s-operator/#getting-started) installation.
* *Optional:* Another tracing backend would be ideal - If none exists, a Jaeger instance can be created.

### Installation

* Deploy the Tempo Operator to your cluster.

* Create an `OpenTelemetryCollector` Object that points to your desired trace destination. The following example exports the traces received from the operands to a service (`jaeger-collector`) via otlp installed the same namespace.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: sidecar-for-tempo
spec:
  mode: sidecar
  config: |
    receivers:
      jaeger:
        protocols:
          thrift_compact:

    exporters:
      otlp:
        endpoint: jaeger-collector:4317
        tls:
          insecure: true

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          exporters: [otlp]
```

* *Optional:* If no trace destination is available, a Jaeger all-in-one instance can be created as follows:

```yaml
apiVersion: v1
kind: List
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: jaeger
      labels:
        app: jaeger
        app.kubernetes.io/name: jaeger
        app.kubernetes.io/component: all-in-one
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: jaeger
      strategy:
        type: Recreate
      template:
        metadata:
          labels:
            app: jaeger
            app.kubernetes.io/name: jaeger
            app.kubernetes.io/component: all-in-one
          annotations:
            prometheus.io/scrape: "true"
            prometheus.io/port: "16686"
        spec:
          containers:
            -   env:
                  - name: COLLECTOR_OTLP_ENABLED
                    value: "true"
                  - name: JAEGER_SERVICE_NAME
                    value: "self"
                image: jaegertracing/all-in-one:1.42.0
                name: jaeger
                ports:
                  - containerPort: 16686
                    protocol: TCP
                  - containerPort: 4317
                    protocol: TCP
                  - containerPort: 4318
                    protocol: TCP
  - apiVersion: v1
    kind: Service
    metadata:
      name: jaeger-query
      labels:
        app: jaeger
        app.kubernetes.io/name: jaeger
        app.kubernetes.io/component: query
    spec:
      ports:
        - name: query-http
          port: 80
          protocol: TCP
          targetPort: 16686
      selector:
        app.kubernetes.io/name: jaeger
        app.kubernetes.io/component: all-in-one
      type: ClusterIP
  - apiVersion: v1
    kind: Service
    metadata:
      name: jaeger-collector
      labels:
        app: jaeger
        app.kubernetes.io/name: jaeger
        app.kubernetes.io/component: collector
    spec:
      ports:
        - name: grpc-otlp
          port: 4317
          protocol: TCP
          targetPort: 4317
        - name: http-otlp
          port: 4318
          protocol: TCP
          targetPort: 4318
      selector:
        app.kubernetes.io/name: jaeger
        app.kubernetes.io/component: all-in-one
      type: ClusterIP
```


### Configuration

Finally, we create a tempostack instance. As `jaeger_agent_endpoint` we choose `localhost`. The OpenTelemetry Operator injects a sidecar into the pod of all operands, which is then listining on localhost.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simple-stack
spec:
  template:
    queryFrontend:
      jaegerQuery:
        enabled:
  storage:
    secret:
      type: s3
      name: minio-test
  storageSize: 200M
  observability:
    tracing:
      sampling_fraction: "1.0"
      jaeger_agent_endpoint: localhost:6831
```
