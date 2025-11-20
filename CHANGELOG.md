Changes by Version
==================

<!-- next version -->

## 0.19.0

### 💡 Enhancements 💡

- `tempostack, tempomonolithic`: Update Tempo to 2.9.0 (#1308)

### 🧰 Bug fixes 🧰

- `tempomonolithic`: Scrape tempo metrics for monolithic. (#1275)
- `tempostack`: Restart pods when certificates are re-generated. (#1301)

### Components
- Tempo: [v2.9.0](https://github.com/grafana/tempo/releases/tag/v2.9.0)

### Support
This release supports Kubernetes 1.25 to 1.32.

## 0.18.0

### 💡 Enhancements 💡

- `operator`: Add feature gate to enable reconcile of Operator's default network policies. (#1248)
- `tempostack, tempomonolithic`: Update Tempo to 2.8.2 (#1276)
  Update Tempo to 2.8.2.
  Changelog: https://github.com/grafana/tempo/releases/tag/v2.8.2
  

### Components
- Tempo: [v2.8.2](https://github.com/grafana/tempo/releases/tag/v2.8.2)

### Support
This release supports Kubernetes 1.25 to 1.32.

## 0.17.1

### 🧰 Bug fixes 🧰

- `github action`: Fix release workflow (#1243)
  Fix the image tag of the must-gather image.
  

### Components
- Tempo: [v2.8.1](https://github.com/grafana/tempo/releases/tag/v2.8.1)

### Support
This release supports Kubernetes 1.25 to 1.32.

## 0.17.0

### 💡 Enhancements 💡

- `tempomonolithic`: Add attribute for configure tempo-query resources request and limits (#1227)
- `tempomonolithic`: Add `storageClassName` and `podSecurityContext` configuration options (#1231)
- `tempostack, tempomonolithic`: Update Tempo to 2.8.1 (#1238)
  Update Tempo to 2.8.1.
  Changelog: https://github.com/grafana/tempo/releases/tag/v2.8.0 and https://github.com/grafana/tempo/releases/tag/v2.8.1
  
- `tempostack, tempomonolithic`: Use SelfSubjectAccessReview instead of SubjectAccessReview when OpenShift multi-tenancy mode is enabled (#1241)

### 🧰 Bug fixes 🧰

- `tempostack`: Remove deprecated `storage.trace.cache` setting (#1136)

### Components
- Tempo: [v2.8.1](https://github.com/grafana/tempo/releases/tag/v2.8.1)

### Support
This release supports Kubernetes 1.25 to 1.32.

## 0.16.0

### 🛑 Breaking changes 🛑

- `tempostack, tempomonolithic`: Ensure the operator does not grant additional permissions when enabling OpenShift tenancy mode (resolves CVE-2025-2786) (#1145)
  Ensure the permissions the operator is granting to the Tempo Service Account
  do not exceed the permissions of the user creating (or modifying) the Tempo instance
  when enabling OpenShift tenancy mode.
  
  To enable the OpenShift tenancy mode, the user must have permissions to create `TokenReview` and `SubjectAccessReview`.
  
  This breaking change does not affect existing Tempo instances in the cluster.
  However, the required permissions are now mandatory when creating or modifying a TempoStack or TempoMonolithic CR.
  

### 💡 Enhancements 💡

- `tempostack, tempomonolithic`: Add short live token authentication for Azure Blob Storage (#1206)
  For use short live token on Azure, the secret should contain the following configuration: 
    ```
  data:
    container:         # Azure blob storage container name
    account_name:      # Azure blob storage account name
    client_id:         # Azure managed identity clientID
    tenant_id:         # Azure tenant ID in which the managed identity lives.
    audience:          # (optional) Audience of the token, default to api://AzureADTokenExchange
  ```
  
- `tempostack, tempomonolithic`: Support for AWS STS via cloudcredential operator (#1159)
- `tempostack, tempomonolithic`: Add support for GCS Shot Live Token authentication. (#1141)
  Now storage secret for GCS can contain
  ```
  data:
    bucketname:         # Bucket name
    iam_sa:             # a name for your the Google IAM service account
    iam_sa_project_id:  # The project ID for your IAM service account.
  ```
  
- `tempostack, tempomonolithic`: Set GOMEMLIMIT to 80% of memory limit, if any (#1196)
  This golang variable indicate to GoLang GC to be more aggressive when it is reaching out the
  memory limits. This is a soft limit, so still can produce OOM, but reduces the possibility. 
  
- `operator`: Kubernetes 1.32 enablement (#1157)
- `tempomonolithic`: Watch storage secrets for tempo monolithic (#1181)

### 🧰 Bug fixes 🧰

- `tempostack, tempomonolithic`: Add parameter to set audience in ID token for GCP Workload Identity Federation (#1209)
  Now that GCS token allow to set the audience, the secret configuration required channged, now it will require
  the following:
  ```
  data:
    bucketname:    # GCS Bucket  name
    audience:      # (Optional) default to openshift
    key.json:      # Credential file generated using gclient
  ```
  
  File key.json can be created using :
  
  ```
  gcloud iam workload-identity-pools create-cred-config \
    "projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<POOL_ID>/providers/<PROVIDER_ID>" \
    --service-account="<SERVICE_ACCOUNT_EMAIL>" \
    --credential-source-file=/var/run/secrets/storage/serviceaccount/token \
    --credential-source-type=text \
    --output-file="/tmp/key.json"
  ```
  credential-source-file= Should be pointing to `/var/run/secrets/storage/serviceaccount/token` which is the locationn
  operator mounts the projected volume.
  
- `tempostack, tempomonolithic`: Add namespace suffix to ClusterRole and ClusterRoleBinding of gateway (#1146)
  This resolves a naming conflict of the ClusterRole and ClusterRoleBinding when two TempoStack/TempoMonolithic instances with the same name, but in different namespaces are created.
  Only relevant when using multi-tenancy with OpenShift mode.
  
- `tempostack, tempomonolithic`: Fix pruning of cluster-scoped resources (#1168)
  Previously, when a non-multitenant TempoStack instance was created using the same name as an existing multitenant TempoStack instance, the operator erroneously deleted the Gateway ClusterRole and ClusterRoleBinding associated with the multitenant instance.
  
  With this change, cluster-scoped resources get an additional label `app.kubernetes.io/namespace` to signify the namespace of the TempoStack owning this cluster-scoped resource.
  
- `tempostack, tempomonolithic`: Cleanup gateway cluster roles and bindings after deleting tempo instance (#1190)
  Now the operator uses finalizer to clean up the cluster roles and bindings after deleting the tempo instance.
  
- `tempostack, tempomonolithic`: Allow OpenShift cluster admins to see all attributes when RBAC is enabled. (#1185)
  This change removes `--opa.admin-groups=system:cluster-admins,cluster-admin,dedicated-admin`
  from the OpenShift OPA configuration. This configures the OPA to always return
  all user's accessible namespaces required by the RBAC feature.
  
- `tempostack, tempomonolithic`: Don't set --opa.matcher=kubernetes_namespace_name when query RBAC is disabled (#1176)
- `tempostack`: Fix unimplemented per tenant retention and fix per tenant overrides after tempo 2.3 (#1134)
  In tempo 2.3 https://github.com/grafana/tempo/blob/main/CHANGELOG.md#v230--2023-10-30 they changes the overrides config
  which was not properly implemented in the operator.
  
  This patch also adds support for per tenant retention which was not implemented.
  
- `tempostack, tempomonolithic`: Assign a percentage of the resources to oauth-proxy if resources are not specified, fixed the name (#1107)
- `tempostack`: Limit granted permissions of the Tempo Service Account when enabling the Jaeger UI Monitor tab on OpenShift (resolves CVE-2025-2842) (#1144)
  Previously, the operator assigned the `cluster-monitoring-view` ClusterRole to the Tempo Service Account
  when the Prometheus endpoint of the Jaeger UI Monitor tab is set to the Thanos Querier on OpenShift.
  
  With this change, the operator limits the granted permissions to only view metrics of the namespace of the Tempo instance.
  Additionally, the recommended port of the Thanos Querier service changed from `9091` to `9092` (tenancy-aware port):
  `.spec.template.queryFrontend.jaegerQuery.monitorTab.prometheusEndpoint: https://thanos-querier.openshift-monitoring.svc.cluster.local:9092`.
  
  All existing installations, which have the Thanos Querier configured at port 9091, will be upgraded automatically to use port 9092.
  
- `tempostack, tempomonolithic`: Update Tempo to 2.7.2 (#1149)

### Components
- Tempo: [v2.7.2](https://github.com/grafana/tempo/releases/tag/v2.7.2)

### Support
This release supports Kubernetes 1.25 to 1.32.

## 0.15.3

### 💡 Enhancements 💡

- `tempomonolithic`: Add support for query RBAC (#1131)
  This feature allows users to apply query RBAC in the multitenancy mode.
  The RBAC allows filtering span/resource/scope attributes and events based on the namespaces which a user querying the data can access.
  For instance, a user can only see attributes from namespaces it can access.
  
  ```yaml
  spec:
    query:
      rbac:
        enabled: true
  ```
  

### Components
- Tempo: [v2.7.1](https://github.com/grafana/tempo/releases/tag/v2.7.1)

## 0.15.2

### Components
- Tempo: [v2.7.1](https://github.com/grafana/tempo/releases/tag/v2.7.1)

## 0.15.1

### Components
- Tempo: [v2.7.1](https://github.com/grafana/tempo/releases/tag/v2.7.1)

## 0.15.0

### 🛑 Breaking changes 🛑

- `tempostack, tempomonolithic`: Update Tempo to 2.7.0 (#1110)
  Update Tempo to 2.7.0 https://github.com/grafana/tempo/releases/tag/v2.7.0
  The Tempo instrumentation changed from Jaeger to OpenTelemetry with OTLP/http exporter.
  
  The `spec.observability.tracing.jaeger_agent_endpoint` is deprecated in favor of `spec.observability.tracing.otlp_http_endpoint`.
  ```yaml
  spec:
    observability:
      tracing:
        jaeger_agent_endpoint: # Deprecated!
        sampling_fraction: "1"
        otlp_http_endpoint: http://localhost:4320
  ```
  

### 💡 Enhancements 💡

- `tempostack`: Add support for query RBAC when Gateway/multitenancy is used. (#1100)
  This feature allows users to apply query RBAC in the multitenancy mode.
  The RBAC allows filtering span/resource/scope attributes and events based on the namespaces which a user querying the data can access.
  For instance, a user can only see attributes from namespaces it can access.
  
  ```yaml
  spec:
    template:
      gateway:
        enabled: true
        rbac:
          enabled: true
  ```
  
- `operator`: Remove kube-rbac-proxy (#1094)
  The image won't be available and won't be mantained, switched to use WithAuthenticationAndAuthorization

### 🧰 Bug fixes 🧰

- `tempostack`: Include insecure option and tls options when STS S3 token is enabled (#1109)
- `tempostack, tempomonolithic`: Assign a percentage of the resources to oauth-proxy if resources are not specified (#1107)

### Components
- Tempo: [v2.7.0](https://github.com/grafana/tempo/releases/tag/v2.7.0)

## 0.14.2

### 🧰 Bug fixes 🧰

- `tempostack`: Use default Jaeger RED metrics namespace if field is unset (#1096)
  Use the default Jaeger RED metrics namespace if `.spec.template.queryFrontend.jaegerQuery.monitorTab.redMetricsNamespace` is not set.
  Before Jaeger 1.62 the default namespace was empty, since [Jaeger 1.62](https://github.com/jaegertracing/jaeger/releases/tag/v1.62.0) (shipped in Tempo Operator v0.14.0) the default namespace is "traces_span_metrics".
  Before OpenTelemetry Collector v0.109.0 the default namespace of the spanmetrics connector was empty, since [OpenTelemetry Collector v0.109.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.109.0) the default namespace is "traces_span_metrics".
  

### Components
- Tempo: [v2.6.1](https://github.com/grafana/tempo/releases/tag/v2.6.1)

## 0.14.1

### 🧰 Bug fixes 🧰

- `tempostack`: Fix enabling `.spec.observability.tracing` with multi-tenancy on OpenShift (#1081)
- `tempostack, tempomonolithic`: Register missing Jaeger UI routes (#1082)
  Without these routes, hitting refresh on the trace detail, system architecture or monitor page of Jaeger UI results in a 404 when multi-tenancy is enabled.
  

### Components
- Tempo: [v2.6.1](https://github.com/grafana/tempo/releases/tag/v2.6.1)

## 0.14.0

### 🛑 Breaking changes 🛑

- `tempostack`: Use new default metrics namespace/prefix for span RED metrics in Jaeger query. (#1072)
  Use the new RED metrics default namespace `traces.span.metrics` for retrieval from Prometheus.
  Since OpenTelemetry Collector version 0.109.0 the default namespace is set to traces.span.metrics.
  The namespace taken into account by jaeger-query can be configured via a TempoStack CR entry.
  To achieve this the Operator will set the jaeger-query `--prometheus.query.namespace=` flag.
  Since Jaeger version 1.62, jaeger-query uses `traces.span.metrics` as default too.
  
  Example how to restore the default namespace used prior to version `0.109.0`, by configuring an empty value for `redMetricsNamespace` in the TempoStack CR:
  ```
  apiVersion: tempo.grafana.com/v1alpha1
  kind: TempoStack
  ...
  spec:
    template:
      queryFrontend:
        jaegerQuery:
          enabled: true
          monitorTab:
            enabled: true
            prometheusEndpoint: "http://myPromInstance:9090"
            redMetricsNamespace: ""
  ```
  More details can be found here:
  - https://github.com/jaegertracing/jaeger/pull/6007
  - https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/34485
  
- `tempostack, tempomonolithic`: Add unified timeout configuration. It changes the default to 30s. (#1045)
  Adding `spec.timeout` CRD option to configure timeout on all components and default it to 30s.
  Before Tempo server was defaulting to 3m, gateway to 2m, OpenShift route to 30s (for query), oauth-proxy to 30s (for query).
  

### 🚀 New components 🚀

- `must-gather`: Add must-gather to collect information about the components deployed by the operator in a cluster. (#1033)

### 💡 Enhancements 💡

- `tempostack`: Expose a way to set a PodSecurityContext on each component (#996)
- `tempostack, tempomonolithic`: bump jaeger to v1.62 (#1050)
- `tempostack`: Bump jaeger to v1.60 by replacing the tempo-query gRPC storage plugin due to the deprecation in Jaeger 1.58.0 with a gRPC standalone service. (#1025)
- `operator`: Kubernetes 1.30 enablement (#1030)
- `tempostack, tempomonolithic`: Make re-encrypt route the default TLS termination to allow access outside the cluster. (#1027)
- `tempostack, tempomonolithic`: Add tempo-query CRD option to speed up trace search. (#1048)
  Following CRD options were added to speed up trace search in Jaeger UI/API. The trace search first
  searches for traceids and then it gets a full trace. With this configuration option the requests
  to get the full trace can be run in parallel:
  For `TempoStack` - `spec.template.queryFrontend.jaegerQuery.findTracesConcurrentRequests`  
  For `TempoMonolithic` - `spec.jaegerui.findTracesConcurrentRequests`
  
- `tempostack`: bump tempo-query to version with separate tls settings for server and client (#1057)
- `operator`: Update Tempo to v2.6.1 (#1044, #1064)

### 🧰 Bug fixes 🧰

- `tempostack`: The default value for the IngressType type is now correctly "" (empty string). Previously, it was impossible to select it in tools like the OpenShift web console, what could cause some issues. (#1054)
- `tempostack`: Add support for memberlist bind network configuration (#1060)
  Adds support to configure the memberlist instance_addr field using the pod network IP range instead of the default private network range used. 
  In managed Kubernetes/OpenShift cluster environments as well as in special on-prem setup the private IP range might not be available for using them. 
  With this change set the TempoStack administrator can choose as a bind address the current pod network IP assigned by the cluster's pod network.
  
- `tempostack`: grant jaeer-query access to pki certs (#1051)
- `tempostack`: Create query-frontend service monitor with HTTP protocol when gateway is disabled (#1070)
- `tempostack`: Fix panic when toggling spec.storage.tls.enabled to true, when using Tempo with AWS STS (#1067)
- `tempostack, tempomonolithic`: Mount CA and Certs to tempo-query when tls is enabled. (#1038)
- `tempostack, tempomonolithic`: The operator no longer sets the `--prometheus.query.support-spanmetrics-connector` flag that got removed in Jaeger 1.58. (#1036)
  The Flag controled whether the metrics queries should match the OpenTelemetry Collector's spanmetrics connector naming or spanmetrics processor naming.
- `tempostack`: Use the ReadinessProbe to better indicate when tempo-query is ready to accept requests. Improving the startup reliability by avoiding lost data. (#1058)
  Without a readiness check in place, there is a risk that data will be lost when the queryfrontend pod is ready but the tempo query API is not yet available.

### Components
- Tempo: [v2.6.1](https://github.com/grafana/tempo/releases/tag/v2.6.1)

## 0.13.0

### 🧰 Bug fixes 🧰

- `operator`: Fix service account for monitoring-view cluster role binding when using oauth proxy. (#1016)
- `tempostack`: Fix setting annotations for Gateway route (#1014)
- `tempostack, tempomonolithic`: Fix infinite reconciliation on OpenShift when route for Jaeger UI is enabled. (#1018)
- `tempostack, tempomonolithic`: Cleanup instance metrics from the operator on instance delete action. (#1019)

### Components
- Tempo: [v2.5.0](https://github.com/grafana/tempo/releases/tag/v2.5.0)

## 0.12.0

### 💡 Enhancements 💡

- `tempostack, tempomonolithic`: Add support for AWS S3 STS authentication. (#978)
  Now storage secret for S3 can contain
  ```
  data:
    bucket:      # Bucket name
    region:      # A valid AWS region, e.g. us-east-1
    role_arn:    # The AWS IAM Role associated with a trust relationship to Tempo serviceaccount
  ```
- `tempostack`: Use TLS via OpenShift service annotation when gateway/multitenancy is disabled (#963)
  On OpenShift when operator config `servingCertsService` is enabled and the following TempoStack CR is used.
  The operator provisions OpenShift serving certificates for the distributor ingest APIs
  ```
    apiVersion: tempo.grafana.com/v1alpha1
    kind:  TempoStack
    spec:
      template:
        distributor:
          tls:
            enabled: true
  ```
  No `certName` and `caName` should be provided, If you specify it, those will be used instead.
  
  In order to use this on the client side, the openshift CA certificate should be used, there are two ways of get
  access to it. You can mount the configmap generated by the operator, which will have the name `<tempostack-name>-serving-cabundle`
  Or you can access to it on `var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`.
  
  An example of OTel configuration used:
  
  ```
     exporters:
      otlp:
        endpoint: tempo-simplest-distributor.chainsaw-tls-singletenant.svc.cluster.local:4317
        tls:
          insecure: false
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
  ```
- `tempomonolithic`: Use TLS via OpenShift service annotation when gateway/multitenancy is disabled (monolithic) (#963)
  On OpenShift when operator config `servingCertsService` is enabled and the following TempoMonolithic CR is used.
  The operator provisions OpenShift serving certificates for the distributor ingest APIs
  
  ```
    apiVersion: tempo.grafana.com/v1alpha1
    kind:  TempoMonolithic
    spec:
      ingestion:
        otlp:
          grpc:
            tls:
              enabled: true
  ```
  or
  ```
    apiVersion: tempo.grafana.com/v1alpha1
    kind:  TempoMonolithic
    spec:
      ingestion:
        otlp:
          http:
            tls:
              enabled: true
  ```
  No `certName` and `caName` should be provided, If you specify it, those will be used instead.
  
- `tempostack, tempomonolithic`: Bump observatorium gateway, (#991)
  In this version upstream certs and CA are reloaded if changed

### 🧰 Bug fixes 🧰

- `tempostack, tempomonolithic`: Allow configmaps and secrets with dot in the name (as it is valid for those objects to have dots as part of it's name) (#983)
- `tempostack`: Assign correct replicas in gateway component if it is specified in the CR, default is 1 if not set (#993)
- `tempomonolithic`: Allow create a monolithic with tls enabled on both grpc/http (#976)

### Components
- Tempo: [v2.5.0](https://github.com/grafana/tempo/releases/tag/v2.5.0)

## 0.11.1

### 🧰 Bug fixes 🧰

- `operator`: Avoid certificate prompt when accessing UI via gateway (#967)
- `operator`: Modify SA annotations managed by the operator, preserve others. (#970)
  This prevents other controllers that modified the SA from create an infinite loop where the other controller modifies something,
  and tempo-operator removes it, the other controller detect the changes and add its and so on and so on.
  
  This is specific for OpenShift case, where the openshift-controller-manager annotates the SA with
  openshift.io/internal-registry-pull-secret-ref.
  
  See https://github.com/openshift/openshift-controller-manager/pull/288/ and 
  https://docs.openshift.com/container-platform/4.16/release_notes/ocp-4-16-release-notes.html section about 
  "Legacy service account API token secrets are no longer generated for each service account"
  

### Components
- Tempo: [v2.5.0](https://github.com/grafana/tempo/releases/tag/v2.5.0)

## 0.11.0

### 🛑 Breaking changes 🛑

- `operator`: Update Tempo to 2.5.0 (#958)
  Upstream Tempo 2.5.0 image switched user from `root` to `tempo` (10001:10001) and ownership of `/var/tempo`.
  Therefore ingester's `/var/tempo/wal` created by previous deployment using Tempo 2.4.1 needs to be updated and
  changed ownership. The operator upgrades the `/var/tempo` ownership by deploying a `job` with `securityContext.runAsUser(0)`
  and it runs `chown -R /var/tempo 10001:10001`.
  

### 💡 Enhancements 💡

- `operator`: Enable OTLP HTTP on Gateway by default. (#948)
- `operator`: Use golang 1.22 to build the operator (#959)
- `operator`: Make configurable availability of the service names in Tempo monolithic (#942)
- `operator`: Add oauth-proxy support for tempo monolithic (#922)
- `operator`: Protect Jaeger UI when multi tenancy is disabled. (#909)

### Components
- Tempo: [v2.5.0](https://github.com/grafana/tempo/releases/tag/v2.5.0)

## 0.10.0

### 🛑 Breaking changes 🛑

- `operator`: TempoMonolithic: Split `tempo-<name>` service into `tempo-<name>` and `tempo-<name>-jaegerui` (#846)

### 💡 Enhancements 💡

- `operator`: Add the ability to configure an expiration time for jaeger UI services (#904)
- `operator`: Prevent creation of TempoStack and TempoMonolithic with same name (#879)
- `operator`: Bump tempo version to 2.4.1 (#901)
- `operator`: Add storage and managed operands gauge metric to the operator metrics. (#838)
- `operator`: Support Grafana instances in a different namespace (#840)
- `operator`: Support custom ServiceAccount in TempoMonolithic CR (#836)
- `operator`: Enable internal server for health checks in TempoMonolithic CR (#847)
- `operator`: Support multi-tenancy in TempoMonolithic CR (#816)
- `operator`: Support TLS Profile in TempoMonolithic CR (#862)
- `operator`: Support upgrading TempoMonolithic CR (#850)
  The metric series `tempooperator_upgrades_total{state="up-to-date"}` was removed.
  A new label `kind` (`TempoStack` or `TempoMonolithic`) was added to `tempooperator_upgrades_total{}`.
  
- `operator`: Updating Operator-sdk to 1.32 (#717)
- `operator`: Add security context to tempo-query container (#864)

### 🧰 Bug fixes 🧰

- `operator`: Fix parsing of `nodeSelector`, `tolerations` and `affinity` in TempoMonolithic CR (#867)

### Components
- Tempo: [v2.4.1](https://github.com/grafana/tempo/releases/tag/v2.4.1)

## 0.9.0

### 💡 Enhancements 💡

- `operator`: Kubernetes 1.29 enablement (#735)
- `operator`: Allow resource limits/requests override per component (#726)
- `operator`: Support creating ServiceMonitors, PrometheusRules and Grafana Data Sources in TempoMonolithic CR (#793)
- `operator`: Support scheduling rules (nodeSelector, tolerations and affinity) in TempoMonolithic CR (#782)
- `operator`: Expose operand status in TempoMonolithic CR (#787)

### 🧰 Bug fixes 🧰

- `operator`: Fix infinite reconciliation of serving CA Bundle ConfigMap (#818)

### Components
- Tempo: [v2.3.1](https://github.com/grafana/tempo/releases/tag/v2.3.1)

## 0.8.0

### 💡 Enhancements 💡

- `operator`: Make Tempo-Query forwarding on gateway optional (#628)
- `operator`: Support monolithic deployment mode (#710)

  The operator exposes a new CRD `TempoMonolithic`, which manages a Tempo instance in monolithic mode.
  The monolithic mode supports the following additional storage backends: in-memory and file system (persistent volume).
  

### 🧰 Bug fixes 🧰

- `operator`: Fix the cluster-monitoring-view RBAC when operator is deployed in arbitrary namespace (#741)
- `operator`: NIL pointer dereference when OIDC not specified for tenants in static mode (#647)

### Components
- Tempo: [v2.3.1](https://github.com/grafana/tempo/releases/tag/v2.3.1)

## 0.7.0

### 💡 Enhancements 💡

- `operator`: Divide assigned limits with replicas (#721)
- `operator`: Allow override arbitrary tempo configurations (#629)
- `operator`: Create Grafana Tempo Operator datasource (#423)
- `operator`: Add .spec.hashRing.memberlist.enableIPv6 option to enable IPv6 support (#704)
- `operator`: Propagating proxy env vars to component containers (#700)
- `operator`: Upgrade tempo to v2.3.1 (#729)

### 🧰 Bug fixes 🧰

- `operator`: Configure the number of replicas for compactor, querier and query-frontend according to the CR (#712)

### Components
- Tempo: [v2.3.1](https://github.com/grafana/tempo/releases/tag/v2.3.1)

## 0.6.0

### 🛑 Breaking changes 🛑

- `operator`: Move default images from operator configuration to environment variable (#591)
- `operator`: Unset (default) images in TempoStack CR (#674)
  This upgrade reverts any change to the `spec.images` fields of any TempoStack instance.
  Beginning with version 0.6.0, the image location is not stored in the TempoStack instance unless it is changed manually.
  

### 💡 Enhancements 💡

- `operator`: Support configuration of TLS in receiver settings (#527)
- `operator`: Exposing the Tempo API through the gateway (#672)
- `operator`: Reduce log level of certrotation messages (#623)
- `operator`: Upgrade tempo to v2.3.0 (#688)

### 🧰 Bug fixes 🧰

- `gateway`: fix CVE-2023-45142 tempo-gateway-container: opentelemetry: DoS vulnerability in otelhttp (#691)

### Components
- Tempo: [v2.3.0](https://github.com/grafana/tempo/releases/tag/v2.3.0)

## 0.5.0

### 🛑 Breaking changes 🛑

- `operator`: Install operator in tempo-operator-system namespace by default when installed with OLM or manifests of the OpenShift variant (#538)

### 💡 Enhancements 💡

- `operator`: Bump tempo version to 2.2.3 (#646)
- `operands`: Bump operands to fix CVE-2023-39325 (#650)
- `operator`: Expose the OTLP HTTP port in the distributor service. (#610)
- `operator`: Add pprof flag to optionally expose pprof data (#242)
- `operator`: Use tempo service account to query metrics from OpenShift monitoring stack. (#526)
  On OpenShift tempo service account is used to query metrics from OpenShift monitoring stack for the monitor tab.
- `operator`: Support setting a custom CA certificate for S3 object storage (#545)
- `operator`: Enable ingress (or route) in samples, add MinLength validation to .spec.storage.secret.name of the TempoStack CR (#541)
- `operator`: Support monitor tab in Jaeger console (#470)
- `operator`: Explicitly specify log level for all components. (#550)
- `operator`: Support Tempo 2.2.0 (#525)

### 🧰 Bug fixes 🧰

- `operator`: Fix ingester StatefulSet reconciliation if ingester is in an unhealthy state (#597)
- `operator`: Enable mTLS for all components except query-frontend. (#561)
  Only enable mTLS for query-frontend when the gateway is enabled.
- `operator`: Fix for Http2 reset vulnerability CVE-2023-39325 (#642)
- `operator`: Upgrade TempoStack instances once they are switched back from Unmanaged to Managed (#478)

### Components
- Tempo: [v2.2.3](https://github.com/grafana/tempo/releases/tag/v2.2.3)

## 0.4.0

### 💡 Enhancements 💡

- `operator`: Remove operator ServiceMonitor and PrometheusRule when operator deployment is removed (#536)

### 🧰 Bug fixes 🧰

- `operator`: Disable mTLS by default, to allow connections from Grafana to the query-frontend component (#552)
- `apis/tempo/v1alpha1`: provide correct mode name via operator-sdk annotation for oc console (#556)

### Components
- Tempo: [v2.1.1](https://github.com/grafana/tempo/releases/tag/v2.1.1)

## 0.3.0

### 🛑 Breaking changes 🛑

- `operator`: Upgrade tempo to v2.1.1 (#408)

  The `maxSearchBytesPerTrace` global and per-tenant limit is deprecated.
  A new `maxSearchDuration` global and per-tenant limit is available.
  Some metrics got renamed or deleted, see the [Tempo v2.1.0 release notes](https://github.com/grafana/tempo/releases/tag/v2.1.0) for details.
  

### 💡 Enhancements 💡

- `operator`: Add .spec.managedState to TempoStack. It allows disabling reconciliation of the TempoStack CR. The validation and defaulting webhooks remain enabled in unmanaged state. (#411)
- `operator`: Enable mTLS by default for internal components (#505)
- `operator`: Expose the Jaeger GRPC query port (#513)
- `operator`: Expose supported protocols on the distributor (#436)

  The following protocols are now exposed: 
    - Jaeger: thrift-http port 14268, thrift-binary port 6832, thrift-compact port 6831, Grpc port 14250
    - Zipkin port 9411
  
- `operator`: Use internal certificate for internal HTTP server of gateway (#480)
- `operator`: Add ability to create and configure route or ingress for the gateway (#265)
- `operator`: The operator is now at Level 4 - Deep Insights (#504)

  The operator optionally exposes metrics and alerts for the operator and the operand.
  
- `operator`: Enable mTLS on receivers when gateway is enabled (#535)
- `operator`: Enable multitenancy without need the gateway (#224)
- `operator`: Add operator alerts and runbook. (#309)
- `operator`: Add new operator configuration options to enable or disable the creation of ServiceMonitor and PrometheusRule for the operator itself (#491)
- `operator`: Probe webhook server in operator health checks (#459)
- `operator`: Rename Degraded condition to ConfigurationError and expose reconcile errors via a new FailedReconciliation status (#400, #422)
- `operator`: Use consistent log format, specify logger names and update log severity levels of reconcile logs (#430)
- `operator`: Implement operator upgrade (#296)
- `operator`: Validate if createServiceMonitors is enabled when enabling createPrometheusRules in the webhook (#510)
- `operator`: Set tempo version in the status field based on the default tempo version of the operator (#400, #422)

### 🧰 Bug fixes 🧰

- `operator`: Fix a panic when an invalid tenant configuration is provided to the operator. If the authentication is provided but the authorization is not, the validator panics (#494)
- `operator`: Fix TLS configuration of ServiceMonitors (#481)
- `operator`: Always set all status condition values in the tempostack_status_condition metric (#452)
  Additionally, deprecate the `status` label of the tempostack_status_condition metric.
  
- `operator`: Update operator container image location in bundle (#443)
- `operator`: Scope PrometheusRule to a specific TempoStack instance (#485)

### Components
- Tempo: [v2.1.1](https://github.com/grafana/tempo/releases/tag/v2.1.1)

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
  
