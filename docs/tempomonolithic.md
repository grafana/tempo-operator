# Introduction
The `TempoMonolithic` Custom Resource (CR) creates a Tempo deployment in [monolithic mode](https://grafana.com/docs/tempo/latest/setup/deployment/). All components of the Tempo deployment (compactor, distributor, ingester, querier and query-frontend) are contained in a single container.

# Example Deployments
## Minimal Setup
The following manifest creates a deployment with trace ingestion over OTLP/gRPC and OTLP/HTTP, storing traces in a tmpfs (in-memory storage).

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
```

Once the pod is ready, you can send traces to `tempo-sample:4317` (OTLP/gRPC) and `tempo-sample:4318` (OTLP/HTTP).

The Tempo API is available at `http://tempo-sample:3200`.

## Using a Persistent Volume for storage
The following Tempo deployment stores traces in a Persistent Volume.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  storage:
    traces:
      backend: pv
```

Note: The size of the PVC can be configured with `.spec.storage.traces.size` and is `10Gi` by default.

## Using a S3 object storage
The following Tempo deployment stores traces in a S3-compatible object storage.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  storage:
    traces:
      backend: s3
      s3:
        secret: my-storage-secret
```

`my-storage-secret` must be a Kubernetes Secret in the same namespace as the `TempoMonolithic` instance, containing the following fields: `bucket`, `endpoint`, `access_key_id` and `access_key_secret`.

For more information on setting up object storage, please refer to the [Object storage docs](https://grafana.com/docs/tempo/latest/setup/operator/object-storage/).

## Jaeger UI
The following manifests enables Jaeger UI.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  jaegerui:
    enabled: true
```

Enable Ingress:
```yaml
spec:
  jaegerui:
    enabled: true
    ingress:
      enabled: true
```

Enable Route:
```yaml
spec:
  jaegerui:
    enabled: true
    route:
      enabled: true
```

## Specifying Resources
The following manifests shows how to specify resources for each component of the deployment.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  resources:
    limits:
      cpu: "2"
      memory: "2Gi"
  jaegerui:
    enabled: true
    resources:
      limits:
        cpu: "2"
        memory: "2Gi"
```

# Complete Specification
A manifest with all available configuration options is available here: [tempo.grafana.com_tempomonolithics.yaml](spec/tempo.grafana.com_tempomonolithics.yaml).

**Note: This file is auto-generated and does not constitute a valid CR**.
It provides an overview of the structure, the available configuration options and help texts.
