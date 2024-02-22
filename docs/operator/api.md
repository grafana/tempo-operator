
This Document contains the types introduced by the Tempo Operator to be consumed by users.

> This page is automatically generated with `gen-crd-api-reference-docs`.

# tempo.grafana.com/v1alpha1 { #tempo-grafana-com-v1alpha1 }

<div>

<p>Package v1alpha1 contains API Schema definitions for the tempo v1alpha1 API group.</p>

</div>

<b>Resource Types:</b>

## AuthenticationSpec { #tempo-grafana-com-v1alpha1-AuthenticationSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TenantsSpec">TenantsSpec</a>)

</p>

<div>

<p>AuthenticationSpec defines the oidc configuration per tenant for tempo Gateway component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>tenantName</code><br/>

<em>

string

</em>

</td>

<td>

<p>TenantName defines a human readable, unique name of the tenant.
The value of this field must be specified in the X-Scope-OrgID header and in the resources field of a ClusterRole to identify the tenant.</p>

</td>
</tr>

<tr>

<td>

<code>tenantId</code><br/>

<em>

string

</em>

</td>

<td>

<p>TenantID defines a universally unique identifier of the tenant.
Unlike the tenantName, which must be unique at a given time, the tenantId must be unique over the entire lifetime of the Tempo deployment.
Tempo uses this ID to prefix objects in the object storage.</p>

</td>
</tr>

<tr>

<td>

<code>oidc</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-OIDCSpec">

OIDCSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>OIDC defines the spec for the OIDC tenant&rsquo;s authentication.</p>

</td>
</tr>

</tbody>
</table>

## AuthorizationSpec { #tempo-grafana-com-v1alpha1-AuthorizationSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TenantsSpec">TenantsSpec</a>)

</p>

<div>

<p>AuthorizationSpec defines the opa, role bindings and roles
configuration per tenant for tempo Gateway component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>roles</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RoleSpec">

[]RoleSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Roles defines a set of permissions to interact with a tenant.</p>

</td>
</tr>

<tr>

<td>

<code>roleBindings</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RoleBindingsSpec">

[]RoleBindingsSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>RoleBindings defines configuration to bind a set of roles to a set of subjects.</p>

</td>
</tr>

</tbody>
</table>

## ComponentStatus { #tempo-grafana-com-v1alpha1-ComponentStatus }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackStatus">TempoStackStatus</a>)

</p>

<div>

<p>ComponentStatus defines the status of each component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>compactor</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Compactor is a map to the pod status of the compactor pod.</p>

</td>
</tr>

<tr>

<td>

<code>distributor</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Distributor is a map to the per pod status of the distributor deployment</p>

</td>
</tr>

<tr>

<td>

<code>ingester</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Ingester is a map to the per pod status of the ingester statefulset</p>

</td>
</tr>

<tr>

<td>

<code>querier</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Querier is a map to the per pod status of the querier deployment</p>

</td>
</tr>

<tr>

<td>

<code>queryFrontend</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>QueryFrontend is a map to the per pod status of the query frontend deployment</p>

</td>
</tr>

<tr>

<td>

<code>gateway</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Gateway is a map to the per pod status of the query frontend deployment</p>

</td>
</tr>

</tbody>
</table>

## ConditionReason { #tempo-grafana-com-v1alpha1-ConditionReason }

(<code>string</code> alias)

<div>

<p>ConditionReason defines possible reasons for each condition.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;CouldNotGetOpenShiftBaseDomain&#34;</p></td>

<td><p>ReasonCouldNotGetOpenShiftBaseDomain when operator cannot get OpenShift base domain, that is used for OAuth redirect URL.</p>
</td>

</tr><tr><td><p>&#34;CouldNotGetOpenShiftTLSPolicy&#34;</p></td>

<td><p>ReasonCouldNotGetOpenShiftTLSPolicy when operator cannot get OpenShift TLS security cluster policy.</p>
</td>

</tr><tr><td><p>&#34;FailedComponents&#34;</p></td>

<td><p>ReasonFailedComponents when all/some Tempo components fail to roll out.</p>
</td>

</tr><tr><td><p>&#34;FailedReconciliation&#34;</p></td>

<td><p>ReasonFailedReconciliation when the operator failed to reconcile.</p>
</td>

</tr><tr><td><p>&#34;InvalidStorageConfig&#34;</p></td>

<td><p>ReasonInvalidStorageConfig defines that the object storage configuration is invalid (missing or incomplete storage secret).</p>
</td>

</tr><tr><td><p>&#34;InvalidTenantsConfiguration&#34;</p></td>

<td><p>ReasonInvalidTenantsConfiguration when the tenant configuration provided is invalid.</p>
</td>

</tr><tr><td><p>&#34;ReasonMissingGatewayTenantSecret&#34;</p></td>

<td><p>ReasonMissingGatewayTenantSecret when operator cannot get Secret containing sensitive Gateway information.</p>
</td>

</tr><tr><td><p>&#34;PendingComponents&#34;</p></td>

<td><p>ReasonPendingComponents when all/some Tempo components pending dependencies.</p>
</td>

</tr><tr><td><p>&#34;Ready&#34;</p></td>

<td><p>ReasonReady defines a healthy tempo instance.</p>
</td>

</tr></tbody>
</table>

## ConditionStatus { #tempo-grafana-com-v1alpha1-ConditionStatus }

(<code>string</code> alias)

<div>

<p>ConditionStatus defines the status of a condition (e.g. ready, failed, pending or configuration error).</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;ConfigurationError&#34;</p></td>

<td><p>ConditionConfigurationError defines that there is a configuration error.</p>
</td>

</tr><tr><td><p>&#34;Failed&#34;</p></td>

<td><p>ConditionFailed defines that one or more components are in a failed state.</p>
</td>

</tr><tr><td><p>&#34;Pending&#34;</p></td>

<td><p>ConditionPending defines that one or more components are in a pending state.</p>
</td>

</tr><tr><td><p>&#34;Ready&#34;</p></td>

<td><p>ConditionReady defines that all components are ready.</p>
</td>

</tr></tbody>
</table>

## ExtraConfigSpec { #tempo-grafana-com-v1alpha1-ExtraConfigSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>ExtraConfigSpec defines extra configurations for tempo that will be merged with the operator generated, configurations defined here
has precedence and could override generated config.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>tempo</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">

k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tempo defines any extra Tempo configuration, which will be merged with the operator&rsquo;s generated Tempo configuration</p>

</td>
</tr>

</tbody>
</table>

## GrafanaConfigSpec { #tempo-grafana-com-v1alpha1-GrafanaConfigSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ObservabilitySpec">ObservabilitySpec</a>)

</p>

<div>

<p>GrafanaConfigSpec defines configuration for Grafana.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>createDatasource</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>CreateDatasource specifies if a Grafana Datasource should be created for Tempo.</p>

</td>
</tr>

<tr>

<td>

<code>instanceSelector</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta">

Kubernetes meta/v1.LabelSelector

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>InstanceSelector specifies the Grafana instance where the datasource should be created.</p>

</td>
</tr>

</tbody>
</table>

## HashRingSpec { #tempo-grafana-com-v1alpha1-HashRingSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>HashRingSpec defines the hash ring configuration.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>memberlist</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MemberListSpec">

MemberListSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>MemberList configuration spec</p>

</td>
</tr>

</tbody>
</table>

## IngestionLimitSpec { #tempo-grafana-com-v1alpha1-IngestionLimitSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-RateLimitSpec">RateLimitSpec</a>)

</p>

<div>

<p>IngestionLimitSpec defines the limits applied at the ingestion path.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>ingestionBurstSizeBytes</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>IngestionBurstSizeBytes defines the burst size (bytes) used in ingestion.</p>

</td>
</tr>

<tr>

<td>

<code>ingestionRateLimitBytes</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>IngestionRateLimitBytes defines the Per-user ingestion rate limit (bytes) used in ingestion.</p>

</td>
</tr>

<tr>

<td>

<code>maxBytesPerTrace</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>MaxBytesPerTrace defines the maximum number of bytes of an acceptable trace.</p>

</td>
</tr>

<tr>

<td>

<code>maxTracesPerUser</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>MaxTracesPerUser defines the maximum number of traces a user can send.</p>

</td>
</tr>

</tbody>
</table>

## IngressSpec { #tempo-grafana-com-v1alpha1-IngressSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-JaegerQuerySpec">JaegerQuerySpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoGatewaySpec">TempoGatewaySpec</a>)

</p>

<div>

<p>IngressSpec defines Jaeger Query Ingress options.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>type</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-IngressType">

IngressType

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Type defines the type of Ingress for the Jaeger Query UI.
Currently ingress, route and none are supported.</p>

</td>
</tr>

<tr>

<td>

<code>annotations</code><br/>

<em>

map[string]string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Annotations defines the annotations of the Ingress object.</p>

</td>
</tr>

<tr>

<td>

<code>host</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Host defines the hostname of the Ingress object.</p>

</td>
</tr>

<tr>

<td>

<code>ingressClassName</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>IngressClassName defines the name of an IngressClass cluster resource.
Defines which ingress controller serves this ingress resource.</p>

</td>
</tr>

<tr>

<td>

<code>route</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RouteSpec">

RouteSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Route defines the options for the OpenShift route.</p>

</td>
</tr>

</tbody>
</table>

## IngressType { #tempo-grafana-com-v1alpha1-IngressType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-IngressSpec">IngressSpec</a>)

</p>

<div>

<p>IngressType represents how a service should be exposed (ingress vs route).</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;ingress&#34;</p></td>

<td><p>IngressTypeIngress specifies that an ingress entry should be created.</p>
</td>

</tr><tr><td><p>&#34;&#34;</p></td>

<td><p>IngressTypeNone specifies that no ingress or route entry should be created.</p>
</td>

</tr><tr><td><p>&#34;route&#34;</p></td>

<td><p>IngressTypeRoute specifies that a route entry should be created.</p>
</td>

</tr></tbody>
</table>

## JaegerQueryMonitor { #tempo-grafana-com-v1alpha1-JaegerQueryMonitor }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-JaegerQuerySpec">JaegerQuerySpec</a>)

</p>

<div>

<p>JaegerQueryMonitor defines configuration for the service monitoring tab in the Jaeger console.
The monitoring tab uses Prometheus to query span RED metrics.
This feature requires running OpenTelemetry collector with spanmetricsconnector -
<a href="https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/spanmetricsconnector">https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/spanmetricsconnector</a>
which derives span RED metrics from spans and exports the metrics to Prometheus.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>Enabled enables the monitor tab in the Jaeger console.
The PrometheusEndpoint must be configured to enable this feature.</p>

</td>
</tr>

<tr>

<td>

<code>prometheusEndpoint</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>PrometheusEndpoint defines the endpoint to the Prometheus instance that contains the span rate, error, and duration (RED) metrics.
For instance on OpenShift this is set to <a href="https://thanos-querier.openshift-monitoring.svc.cluster.local:9091">https://thanos-querier.openshift-monitoring.svc.cluster.local:9091</a></p>

</td>
</tr>

</tbody>
</table>

## JaegerQuerySpec { #tempo-grafana-com-v1alpha1-JaegerQuerySpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoQueryFrontendSpec">TempoQueryFrontendSpec</a>)

</p>

<div>

<p>JaegerQuerySpec defines Jaeger Query options.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>Enabled defines if the Jaeger Query component should be created.</p>

</td>
</tr>

<tr>

<td>

<code>ingress</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-IngressSpec">

IngressSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Ingress defines the options for the Jaeger Query ingress.</p>

</td>
</tr>

<tr>

<td>

<code>monitorTab</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-JaegerQueryMonitor">

JaegerQueryMonitor

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>MonitorTab defines the monitor tab configuration.</p>

</td>
</tr>

</tbody>
</table>

## LimitSpec { #tempo-grafana-com-v1alpha1-LimitSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>LimitSpec defines Global and PerTenant rate limits.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>perTenant</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RateLimitSpec">

map[string]github.com/grafana/tempo-operator/apis/tempo/v1alpha1.RateLimitSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>PerTenant is used to define rate limits per tenant.</p>

</td>
</tr>

<tr>

<td>

<code>global</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RateLimitSpec">

RateLimitSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Global is used to define global rate limits.</p>

</td>
</tr>

</tbody>
</table>

## ManagementStateType { #tempo-grafana-com-v1alpha1-ManagementStateType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>ManagementStateType defines the type for CR management states.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;Managed&#34;</p></td>

<td><p>ManagementStateManaged when the TempoStack custom resource should be
reconciled by the operator.</p>
</td>

</tr><tr><td><p>&#34;Unmanaged&#34;</p></td>

<td><p>ManagementStateUnmanaged when the TempoStack custom resource should not be
reconciled by the operator.</p>
</td>

</tr></tbody>
</table>

## MemberListSpec { #tempo-grafana-com-v1alpha1-MemberListSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-HashRingSpec">HashRingSpec</a>)

</p>

<div>

<p>MemberListSpec defines the configuration for the memberlist based hash ring.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enableIPv6</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>EnableIPv6 enables IPv6 support for the memberlist based hash ring.</p>

</td>
</tr>

</tbody>
</table>

## MetricsConfigSpec { #tempo-grafana-com-v1alpha1-MetricsConfigSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ObservabilitySpec">ObservabilitySpec</a>)

</p>

<div>

<p>MetricsConfigSpec defines a metrics config.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>createServiceMonitors</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>CreateServiceMonitors specifies if ServiceMonitors should be created for Tempo components.</p>

</td>
</tr>

<tr>

<td>

<code>createPrometheusRules</code><br/>

<em>

bool

</em>

</td>

<td>

<em>(Optional)</em>

<p>CreatePrometheusRules specifies if Prometheus rules for alerts should be created for Tempo components.</p>

</td>
</tr>

</tbody>
</table>

## ModeType { #tempo-grafana-com-v1alpha1-ModeType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TenantsSpec">TenantsSpec</a>)

</p>

<div>

<p>ModeType is the authentication/authorization mode in which Tempo Gateway
will be configured.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;openshift&#34;</p></td>

<td><p>ModeOpenShift mode uses TokenReview API for authentication and subject access review for authorization.</p>
</td>

</tr><tr><td><p>&#34;static&#34;</p></td>

<td><p>ModeStatic mode asserts the Authorization Spec&rsquo;s Roles and RoleBindings
using an in-process OpenPolicyAgent Rego authorizer.</p>
</td>

</tr></tbody>
</table>

## MonolithicComponentStatus { #tempo-grafana-com-v1alpha1-MonolithicComponentStatus }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicStatus">TempoMonolithicStatus</a>)

</p>

<div>

<p>MonolithicComponentStatus defines the status of each component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>tempo</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PodStatusMap">

PodStatusMap

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tempo is a map of the pod status of the Tempo pods.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicIngestionOTLPProtocolsGRPCSpec { #tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsGRPCSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPSpec">MonolithicIngestionOTLPSpec</a>)

</p>

<div>

<p>MonolithicIngestionOTLPProtocolsGRPCSpec defines the settings for OTLP ingestion over GRPC.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if OTLP over gRPC is enabled.
Default: enabled.</p>

</td>
</tr>

<tr>

<td>

<code>tls</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSSpec">

TLSSpec

</a>

</em>

</td>

<td>

<p>TLS defines the TLS configuration for OTLP/gRPC ingestion.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicIngestionOTLPProtocolsHTTPSpec { #tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsHTTPSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPSpec">MonolithicIngestionOTLPSpec</a>)

</p>

<div>

<p>MonolithicIngestionOTLPProtocolsHTTPSpec defines the settings for OTLP ingestion over HTTP.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if OTLP over HTTP is enabled.
Default: enabled.</p>

</td>
</tr>

<tr>

<td>

<code>tls</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSSpec">

TLSSpec

</a>

</em>

</td>

<td>

<p>TLS defines the TLS configuration for OTLP/HTTP ingestion.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicIngestionOTLPSpec { #tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionSpec">MonolithicIngestionSpec</a>)

</p>

<div>

<p>MonolithicIngestionOTLPSpec defines the settings for OTLP ingestion.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>grpc</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsGRPCSpec">

MonolithicIngestionOTLPProtocolsGRPCSpec

</a>

</em>

</td>

<td>

<p>GRPC defines the OTLP over gRPC configuration.</p>

</td>
</tr>

<tr>

<td>

<code>http</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsHTTPSpec">

MonolithicIngestionOTLPProtocolsHTTPSpec

</a>

</em>

</td>

<td>

<p>HTTP defines the OTLP over HTTP configuration.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicIngestionSpec { #tempo-grafana-com-v1alpha1-MonolithicIngestionSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicIngestionSpec defines the ingestion settings.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>otlp</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPSpec">

MonolithicIngestionOTLPSpec

</a>

</em>

</td>

<td>

<p>OTLP defines the ingestion configuration for the OTLP protocol.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicJaegerUIIngressSpec { #tempo-grafana-com-v1alpha1-MonolithicJaegerUIIngressSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUISpec">MonolithicJaegerUISpec</a>)

</p>

<div>

<p>MonolithicJaegerUIIngressSpec defines the settings for the Jaeger UI ingress.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if an Ingress object should be created for Jaeger UI.</p>

</td>
</tr>

<tr>

<td>

<code>annotations</code><br/>

<em>

map[string]string

</em>

</td>

<td>

<p>Annotations defines the annotations of the Ingress object.</p>

</td>
</tr>

<tr>

<td>

<code>host</code><br/>

<em>

string

</em>

</td>

<td>

<p>Host defines the hostname of the Ingress object.</p>

</td>
</tr>

<tr>

<td>

<code>ingressClassName</code><br/>

<em>

string

</em>

</td>

<td>

<p>IngressClassName defines the name of an IngressClass cluster resource.
Defines which ingress controller serves this ingress resource.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicJaegerUIRouteSpec { #tempo-grafana-com-v1alpha1-MonolithicJaegerUIRouteSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUISpec">MonolithicJaegerUISpec</a>)

</p>

<div>

<p>MonolithicJaegerUIRouteSpec defines the settings for the Jaeger UI route.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if a Route object should be created for Jaeger UI.</p>

</td>
</tr>

<tr>

<td>

<code>annotations</code><br/>

<em>

map[string]string

</em>

</td>

<td>

<p>Annotations defines the annotations of the Route object.</p>

</td>
</tr>

<tr>

<td>

<code>host</code><br/>

<em>

string

</em>

</td>

<td>

<p>Host defines the hostname of the Route object.</p>

</td>
</tr>

<tr>

<td>

<code>termination</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSRouteTerminationType">

TLSRouteTerminationType

</a>

</em>

</td>

<td>

<p>Termination specifies the termination type.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicJaegerUISpec { #tempo-grafana-com-v1alpha1-MonolithicJaegerUISpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicJaegerUISpec defines the settings for the Jaeger UI.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if the Jaeger UI component should be created.</p>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core">

Kubernetes core/v1.ResourceRequirements

</a>

</em>

</td>

<td>

<p>Resources defines the compute resource requirements of the Jaeger UI container.</p>

</td>
</tr>

<tr>

<td>

<code>ingress</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUIIngressSpec">

MonolithicJaegerUIIngressSpec

</a>

</em>

</td>

<td>

<p>Ingress defines the Ingress configuration for the Jaeger UI.</p>

</td>
</tr>

<tr>

<td>

<code>route</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUIRouteSpec">

MonolithicJaegerUIRouteSpec

</a>

</em>

</td>

<td>

<p>Route defines the OpenShift route configuration for the Jaeger UI.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicMultitenancySpec { #tempo-grafana-com-v1alpha1-MonolithicMultitenancySpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicMultitenancySpec defines the multi-tenancy settings for Tempo.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if multi-tenancy is enabled.</p>

</td>
</tr>

<tr>

<td>

<code>mode</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ModeType">

ModeType

</a>

</em>

</td>

<td>

<p>Mode defines the multitenancy mode.</p>

</td>
</tr>

<tr>

<td>

<code>authentication</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-AuthenticationSpec">

[]AuthenticationSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Authentication defines the tempo-gateway component authentication configuration spec per tenant.</p>

</td>
</tr>

<tr>

<td>

<code>authorization</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-AuthorizationSpec">

AuthorizationSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Authorization defines the tempo-gateway component authorization configuration spec per tenant.</p>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core">

Kubernetes core/v1.ResourceRequirements

</a>

</em>

</td>

<td>

<p>Resources defines the compute resource requirements of the gateway container.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilityGrafanaDataSourceSpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilityGrafanaDataSourceSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityGrafanaSpec">MonolithicObservabilityGrafanaSpec</a>)

</p>

<div>

<p>MonolithicObservabilityGrafanaDataSourceSpec defines the Grafana data source configuration of the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if a Grafana data source should be created for this Tempo deployment.</p>

</td>
</tr>

<tr>

<td>

<code>instanceSelector</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta">

Kubernetes meta/v1.LabelSelector

</a>

</em>

</td>

<td>

<p>InstanceSelector defines the Grafana instance where the data source should be created.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilityGrafanaSpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilityGrafanaSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilitySpec">MonolithicObservabilitySpec</a>)

</p>

<div>

<p>MonolithicObservabilityGrafanaSpec defines the Grafana configuration of the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>dataSource</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityGrafanaDataSourceSpec">

MonolithicObservabilityGrafanaDataSourceSpec

</a>

</em>

</td>

<td>

<p>DataSource defines the Grafana data source configuration.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilityMetricsPrometheusRulesSpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsPrometheusRulesSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsSpec">MonolithicObservabilityMetricsSpec</a>)

</p>

<div>

<p>MonolithicObservabilityMetricsPrometheusRulesSpec defines the PrometheusRules settings.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if PrometheusRule objects should be created for this Tempo deployment.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilityMetricsServiceMonitorsSpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsServiceMonitorsSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsSpec">MonolithicObservabilityMetricsSpec</a>)

</p>

<div>

<p>MonolithicObservabilityMetricsServiceMonitorsSpec defines the ServiceMonitor settings.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if ServiceMonitor objects should be created for this Tempo deployment.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilityMetricsSpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilitySpec">MonolithicObservabilitySpec</a>)

</p>

<div>

<p>MonolithicObservabilityMetricsSpec defines the metrics settings of the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>serviceMonitors</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsServiceMonitorsSpec">

MonolithicObservabilityMetricsServiceMonitorsSpec

</a>

</em>

</td>

<td>

<p>ServiceMonitors defines the ServiceMonitor configuration.</p>

</td>
</tr>

<tr>

<td>

<code>prometheusRules</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsPrometheusRulesSpec">

MonolithicObservabilityMetricsPrometheusRulesSpec

</a>

</em>

</td>

<td>

<p>ServiceMonitors defines the PrometheusRule configuration.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicObservabilitySpec { #tempo-grafana-com-v1alpha1-MonolithicObservabilitySpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicObservabilitySpec defines the observability configuration of the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>metrics</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityMetricsSpec">

MonolithicObservabilityMetricsSpec

</a>

</em>

</td>

<td>

<p>Metrics defines the metric configuration of the Tempo deployment.</p>

</td>
</tr>

<tr>

<td>

<code>grafana</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilityGrafanaSpec">

MonolithicObservabilityGrafanaSpec

</a>

</em>

</td>

<td>

<p>Grafana defines the Grafana configuration of the Tempo deployment.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicSchedulerSpec { #tempo-grafana-com-v1alpha1-MonolithicSchedulerSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicSchedulerSpec defines schedule settings for Tempo.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>nodeSelector</code><br/>

<em>

map[string]string

</em>

</td>

<td>

<p>NodeSelector defines which labels are required by a node to schedule the pod onto it.</p>

</td>
</tr>

<tr>

<td>

<code>tolerations</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#toleration-v1-core">

[]Kubernetes core/v1.Toleration

</a>

</em>

</td>

<td>

<p>Tolerations defines the tolerations of a node to schedule the pod onto it.</p>

</td>
</tr>

<tr>

<td>

<code>affinity</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#affinity-v1-core">

Kubernetes core/v1.Affinity

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Affinity defines the Affinity rules for scheduling pods.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicStorageSpec { #tempo-grafana-com-v1alpha1-MonolithicStorageSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">TempoMonolithicSpec</a>)

</p>

<div>

<p>MonolithicStorageSpec defines the storage for the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>traces</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageSpec">

MonolithicTracesStorageSpec

</a>

</em>

</td>

<td>

<p>Traces defines the storage configuration for traces.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicTracesObjectStorageSpec { #tempo-grafana-com-v1alpha1-MonolithicTracesObjectStorageSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageS3Spec">MonolithicTracesStorageS3Spec</a>, <a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageSpec">MonolithicTracesStorageSpec</a>)

</p>

<div>

<p>MonolithicTracesObjectStorageSpec defines object storage configuration.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>secret</code><br/>

<em>

string

</em>

</td>

<td>

<p>Secret is the name of a Secret containing credentials for accessing object storage.
It needs to be in the same namespace as the TempoMonolithic custom resource.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicTracesStorageBackend { #tempo-grafana-com-v1alpha1-MonolithicTracesStorageBackend }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageSpec">MonolithicTracesStorageSpec</a>)

</p>

<div>

<p>MonolithicTracesStorageBackend defines the backend storage for traces.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;azure&#34;</p></td>

<td><p>MonolithicTracesStorageBackendAzure defines storing traces in Azure Storage.</p>
</td>

</tr><tr><td><p>&#34;gcs&#34;</p></td>

<td><p>MonolithicTracesStorageBackendGCS defines storing traces in Google Cloud Storage.</p>
</td>

</tr><tr><td><p>&#34;memory&#34;</p></td>

<td><p>MonolithicTracesStorageBackendMemory defines storing traces in a tmpfs (in-memory filesystem).</p>
</td>

</tr><tr><td><p>&#34;pv&#34;</p></td>

<td><p>MonolithicTracesStorageBackendPV defines storing traces in a Persistent Volume.</p>
</td>

</tr><tr><td><p>&#34;s3&#34;</p></td>

<td><p>MonolithicTracesStorageBackendS3 defines storing traces in Amazon S3.</p>
</td>

</tr></tbody>
</table>

## MonolithicTracesStorageS3Spec { #tempo-grafana-com-v1alpha1-MonolithicTracesStorageS3Spec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageSpec">MonolithicTracesStorageSpec</a>)

</p>

<div>

<p>MonolithicTracesStorageS3Spec defines the Amazon S3 configuration.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>secret</code><br/>

<em>

string

</em>

</td>

<td>

<p>Secret is the name of a Secret containing credentials for accessing object storage.
It needs to be in the same namespace as the TempoMonolithic custom resource.</p>

</td>
</tr>

<tr>

<td>

<code>tls</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSSpec">

TLSSpec

</a>

</em>

</td>

<td>

<p>TLS defines the TLS configuration for Amazon S3.</p>

</td>
</tr>

</tbody>
</table>

## MonolithicTracesStorageSpec { #tempo-grafana-com-v1alpha1-MonolithicTracesStorageSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicStorageSpec">MonolithicStorageSpec</a>)

</p>

<div>

<p>MonolithicTracesStorageSpec defines the traces storage for the Tempo deployment.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>backend</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageBackend">

MonolithicTracesStorageBackend

</a>

</em>

</td>

<td>

<p>Backend defines the backend for storing traces.
Default: memory.</p>

</td>
</tr>

<tr>

<td>

<code>size</code><br/>

<em>

k8s.io/apimachinery/pkg/api/resource.Quantity

</em>

</td>

<td>

<p>Size defines the size of the volume where traces are stored.
For in-memory storage, this defines the size of the tmpfs volume.
For persistent volume storage, this defines the size of the persistent volume.
For object storage, this defines the size of the persistent volume containing the Write-Ahead Log (WAL) of Tempo.
Default: 10Gi.</p>

</td>
</tr>

<tr>

<td>

<code>s3</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageS3Spec">

MonolithicTracesStorageS3Spec

</a>

</em>

</td>

<td>

<p>S3 defines the configuration for Amazon S3.</p>

</td>
</tr>

<tr>

<td>

<code>azure</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicTracesObjectStorageSpec">

MonolithicTracesObjectStorageSpec

</a>

</em>

</td>

<td>

<p>Azure defines the configuration for Azure Storage.</p>

</td>
</tr>

<tr>

<td>

<code>gcs</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicTracesObjectStorageSpec">

MonolithicTracesObjectStorageSpec

</a>

</em>

</td>

<td>

<p>GCP defines the configuration for Google Cloud Storage.</p>

</td>
</tr>

</tbody>
</table>

## OIDCSpec { #tempo-grafana-com-v1alpha1-OIDCSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-AuthenticationSpec">AuthenticationSpec</a>)

</p>

<div>

<p>OIDCSpec defines the oidc configuration spec for Tempo Gateway component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>secret</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TenantSecretSpec">

TenantSecretSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Secret defines the spec for the clientID, clientSecret and issuerCAPath for tenant&rsquo;s authentication.</p>

</td>
</tr>

<tr>

<td>

<code>issuerURL</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>IssuerURL defines the URL for issuer.</p>

</td>
</tr>

<tr>

<td>

<code>redirectURL</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>RedirectURL defines the URL for redirect.</p>

</td>
</tr>

<tr>

<td>

<code>groupClaim</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Group claim field from ID Token</p>

</td>
</tr>

<tr>

<td>

<code>usernameClaim</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>User claim field from ID Token</p>

</td>
</tr>

</tbody>
</table>

## ObjectStorageSecretSpec { #tempo-grafana-com-v1alpha1-ObjectStorageSecretSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ObjectStorageSpec">ObjectStorageSpec</a>)

</p>

<div>

<p>ObjectStorageSecretSpec is a secret reference containing name only, no namespace.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>type</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ObjectStorageSecretType">

ObjectStorageSecretType

</a>

</em>

</td>

<td>

<p>Type of object storage that should be used</p>

</td>
</tr>

<tr>

<td>

<code>name</code><br/>

<em>

string

</em>

</td>

<td>

<p>Name of a secret in the namespace configured for object storage secrets.</p>

</td>
</tr>

</tbody>
</table>

## ObjectStorageSecretType { #tempo-grafana-com-v1alpha1-ObjectStorageSecretType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ObjectStorageSecretSpec">ObjectStorageSecretSpec</a>)

</p>

<div>

<p>ObjectStorageSecretType defines the type of storage which can be used with the Tempo cluster.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;azure&#34;</p></td>

<td><p>ObjectStorageSecretAzure when using Azure Storage for Tempo storage.</p>
</td>

</tr><tr><td><p>&#34;gcs&#34;</p></td>

<td><p>ObjectStorageSecretGCS when using Google Cloud Storage for Tempo storage.</p>
</td>

</tr><tr><td><p>&#34;s3&#34;</p></td>

<td><p>ObjectStorageSecretS3 when using S3 for Tempo storage.</p>
</td>

</tr></tbody>
</table>

## ObjectStorageSpec { #tempo-grafana-com-v1alpha1-ObjectStorageSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>ObjectStorageSpec defines the requirements to access the object
storage bucket to persist traces by the ingester component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>tls</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSSpec">

TLSSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TLS configuration for reaching the object storage endpoint.</p>

</td>
</tr>

<tr>

<td>

<code>secret</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ObjectStorageSecretSpec">

ObjectStorageSecretSpec

</a>

</em>

</td>

<td>

<p>Secret for object storage authentication.
Name of a secret in the same namespace as the TempoStack custom resource.</p>

</td>
</tr>

</tbody>
</table>

## ObservabilitySpec { #tempo-grafana-com-v1alpha1-ObservabilitySpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>ObservabilitySpec defines how telemetry data gets handled.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>metrics</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MetricsConfigSpec">

MetricsConfigSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Metrics defines the metrics configuration for operands.</p>

</td>
</tr>

<tr>

<td>

<code>tracing</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TracingConfigSpec">

TracingConfigSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tracing defines a config for operands.</p>

</td>
</tr>

<tr>

<td>

<code>grafana</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-GrafanaConfigSpec">

GrafanaConfigSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Grafana defines the Grafana configuration for operands.</p>

</td>
</tr>

</tbody>
</table>

## PermissionType { #tempo-grafana-com-v1alpha1-PermissionType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-RoleSpec">RoleSpec</a>)

</p>

<div>

<p>PermissionType is a Tempo Gateway RBAC permission.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;read&#34;</p></td>

<td><p>Read gives access to read data from a tenant.</p>
</td>

</tr><tr><td><p>&#34;write&#34;</p></td>

<td><p>Write gives access to write data to a tenant.</p>
</td>

</tr></tbody>
</table>

## PodStatusMap { #tempo-grafana-com-v1alpha1-PodStatusMap }

(<code>map[k8s.io/api/core/v1.PodPhase][]string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ComponentStatus">ComponentStatus</a>, <a href="#tempo-grafana-com-v1alpha1-MonolithicComponentStatus">MonolithicComponentStatus</a>)

</p>

<div>

<p>PodStatusMap defines the type for mapping pod status to pod name.</p>

</div>

## QueryLimit { #tempo-grafana-com-v1alpha1-QueryLimit }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-RateLimitSpec">RateLimitSpec</a>)

</p>

<div>

<p>QueryLimit defines query limits.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>maxBytesPerTagValues</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>MaxBytesPerTagValues defines the maximum size in bytes of a tag-values query.</p>

</td>
</tr>

<tr>

<td>

<code>maxSearchBytesPerTrace</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>DEPRECATED. MaxSearchBytesPerTrace defines the maximum size of search data for a single
trace in bytes.
default: <code>0</code> to disable.</p>

</td>
</tr>

<tr>

<td>

<code>maxSearchDuration</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>MaxSearchDuration defines the maximum allowed time range for a search.
If this value is not set, then spec.search.maxDuration is used.</p>

</td>
</tr>

</tbody>
</table>

## RateLimitSpec { #tempo-grafana-com-v1alpha1-RateLimitSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-LimitSpec">LimitSpec</a>)

</p>

<div>

<p>RateLimitSpec defines rate limits for Ingestion and Query components.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>ingestion</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-IngestionLimitSpec">

IngestionLimitSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Ingestion is used to define ingestion rate limits.</p>

</td>
</tr>

<tr>

<td>

<code>query</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-QueryLimit">

QueryLimit

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Query is used to define query rate limits.</p>

</td>
</tr>

</tbody>
</table>

## Resources { #tempo-grafana-com-v1alpha1-Resources }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>Resources defines resources configuration.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>total</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core">

Kubernetes core/v1.ResourceRequirements

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>The total amount of resources for Tempo instance.
The operator autonomously splits resources between deployed Tempo components.
Only limits are supported, the operator calculates requests automatically.
See <a href="http://github.com/grafana/tempo/issues/1540">http://github.com/grafana/tempo/issues/1540</a>.</p>

</td>
</tr>

</tbody>
</table>

## RetentionConfig { #tempo-grafana-com-v1alpha1-RetentionConfig }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-RetentionSpec">RetentionSpec</a>)

</p>

<div>

<p>RetentionConfig defines how long data should be provided.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>traces</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Traces defines retention period. Supported parameter suffixes are &ldquo;s&rdquo;, &ldquo;m&rdquo; and &ldquo;h&rdquo;.
example: 336h
default: value is 48h.</p>

</td>
</tr>

</tbody>
</table>

## RetentionSpec { #tempo-grafana-com-v1alpha1-RetentionSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>RetentionSpec defines global and per tenant retention configurations.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>perTenant</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RetentionConfig">

map[string]github.com/grafana/tempo-operator/apis/tempo/v1alpha1.RetentionConfig

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>PerTenant is used to configure retention per tenant.</p>

</td>
</tr>

<tr>

<td>

<code>global</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RetentionConfig">

RetentionConfig

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Global is used to configure global retention.</p>

</td>
</tr>

</tbody>
</table>

## RoleBindingsSpec { #tempo-grafana-com-v1alpha1-RoleBindingsSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-AuthorizationSpec">AuthorizationSpec</a>)

</p>

<div>

<p>RoleBindingsSpec binds a set of roles to a set of subjects.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>name</code><br/>

<em>

string

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>subjects</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-Subject">

[]Subject

</a>

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>roles</code><br/>

<em>

[]string

</em>

</td>

<td>

</td>
</tr>

</tbody>
</table>

## RoleSpec { #tempo-grafana-com-v1alpha1-RoleSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-AuthorizationSpec">AuthorizationSpec</a>)

</p>

<div>

<p>RoleSpec describes a set of permissions to interact with a tenant.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>name</code><br/>

<em>

string

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

[]string

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>tenants</code><br/>

<em>

[]string

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>permissions</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-PermissionType">

[]PermissionType

</a>

</em>

</td>

<td>

</td>
</tr>

</tbody>
</table>

## RouteSpec { #tempo-grafana-com-v1alpha1-RouteSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-IngressSpec">IngressSpec</a>)

</p>

<div>

<p>RouteSpec defines OpenShift Route specific options.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>termination</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSRouteTerminationType">

TLSRouteTerminationType

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Termination defines the termination type.
The default is &ldquo;edge&rdquo;.</p>

</td>
</tr>

</tbody>
</table>

## SearchSpec { #tempo-grafana-com-v1alpha1-SearchSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>SearchSpec specified the global search parameters.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>defaultResultLimit</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>Limit used for search requests if none is set by the caller (default: 20)</p>

</td>
</tr>

<tr>

<td>

<code>maxDuration</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>The maximum allowed time range for a search, default: 0s which means unlimited.</p>

</td>
</tr>

<tr>

<td>

<code>maxResultLimit</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>The maximum allowed value of the limit parameter on search requests. If the search request limit parameter
exceeds the value configured here it will be set to the value configured here.
The default value of 0 disables this limit.</p>

</td>
</tr>

</tbody>
</table>

## Subject { #tempo-grafana-com-v1alpha1-Subject }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-RoleBindingsSpec">RoleBindingsSpec</a>)

</p>

<div>

<p>Subject represents a subject that has been bound to a role.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>name</code><br/>

<em>

string

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>kind</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-SubjectKind">

SubjectKind

</a>

</em>

</td>

<td>

</td>
</tr>

</tbody>
</table>

## SubjectKind { #tempo-grafana-com-v1alpha1-SubjectKind }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-Subject">Subject</a>)

</p>

<div>

<p>SubjectKind is a kind of Tempo Gateway RBAC subject.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;group&#34;</p></td>

<td><p>Group represents a subject that is a group.</p>
</td>

</tr><tr><td><p>&#34;user&#34;</p></td>

<td><p>User represents a subject that is a user.</p>
</td>

</tr></tbody>
</table>

## TLSRouteTerminationType { #tempo-grafana-com-v1alpha1-TLSRouteTerminationType }

(<code>string</code> alias)

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUIRouteSpec">MonolithicJaegerUIRouteSpec</a>, <a href="#tempo-grafana-com-v1alpha1-RouteSpec">RouteSpec</a>)

</p>

<div>

<p>TLSRouteTerminationType is used to indicate which TLS settings should be used.</p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;edge&#34;</p></td>

<td><p>TLSRouteTerminationTypeEdge indicates that encryption should be terminated
at the edge router.</p>
</td>

</tr><tr><td><p>&#34;insecure&#34;</p></td>

<td><p>TLSRouteTerminationTypeInsecure indicates that insecure connections are allowed.</p>
</td>

</tr><tr><td><p>&#34;passthrough&#34;</p></td>

<td><p>TLSRouteTerminationTypePassthrough indicates that the destination service is
responsible for decrypting traffic.</p>
</td>

</tr><tr><td><p>&#34;reencrypt&#34;</p></td>

<td><p>TLSRouteTerminationTypeReencrypt indicates that traffic will be decrypted on the edge
and re-encrypt using a new certificate.</p>
</td>

</tr></tbody>
</table>

## TLSSpec { #tempo-grafana-com-v1alpha1-TLSSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsGRPCSpec">MonolithicIngestionOTLPProtocolsGRPCSpec</a>, <a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionOTLPProtocolsHTTPSpec">MonolithicIngestionOTLPProtocolsHTTPSpec</a>, <a href="#tempo-grafana-com-v1alpha1-MonolithicTracesStorageS3Spec">MonolithicTracesStorageS3Spec</a>, <a href="#tempo-grafana-com-v1alpha1-ObjectStorageSpec">ObjectStorageSpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoDistributorSpec">TempoDistributorSpec</a>)

</p>

<div>

<p>TLSSpec is the TLS configuration.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

<p>Enabled defines if TLS is enabled.</p>

</td>
</tr>

<tr>

<td>

<code>caName</code><br/>

<em>

string

</em>

</td>

<td>

<p>CA is the name of a ConfigMap containing a CA certificate (service-ca.crt).
It needs to be in the same namespace as the Tempo custom resource.</p>

</td>
</tr>

<tr>

<td>

<code>certName</code><br/>

<em>

string

</em>

</td>

<td>

<p>Cert is the name of a Secret containing a certificate (tls.crt) and private key (tls.key).
It needs to be in the same namespace as the Tempo custom resource.</p>

</td>
</tr>

<tr>

<td>

<code>minVersion</code><br/>

<em>

string

</em>

</td>

<td>

<p>MinVersion defines the minimum acceptable TLS version.</p>

</td>
</tr>

</tbody>
</table>

## TempoComponentSpec { #tempo-grafana-com-v1alpha1-TempoComponentSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoDistributorSpec">TempoDistributorSpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoGatewaySpec">TempoGatewaySpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoQueryFrontendSpec">TempoQueryFrontendSpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoTemplateSpec">TempoTemplateSpec</a>)

</p>

<div>

<p>TempoComponentSpec defines specific schedule settings for tempo components.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>replicas</code><br/>

<em>

int32

</em>

</td>

<td>

<em>(Optional)</em>

<p>Replicas defines the number of replicas to be created for this component.</p>

</td>
</tr>

<tr>

<td>

<code>nodeSelector</code><br/>

<em>

map[string]string

</em>

</td>

<td>

<em>(Optional)</em>

<p>NodeSelector defines the simple form of the node-selection constraint.</p>

</td>
</tr>

<tr>

<td>

<code>tolerations</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#toleration-v1-core">

[]Kubernetes core/v1.Toleration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tolerations defines component-specific pod tolerations.</p>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core">

Kubernetes core/v1.ResourceRequirements

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Resources defines resources for this component, this will override the calculated resources derived from total</p>

</td>
</tr>

</tbody>
</table>

## TempoDistributorSpec { #tempo-grafana-com-v1alpha1-TempoDistributorSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoTemplateSpec">TempoTemplateSpec</a>)

</p>

<div>

<p>TempoDistributorSpec defines the template of all requirements to configure
scheduling of Tempo distributor component to be deployed.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>component</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoComponentSpec is embedded to extend this definition with further options.</p>

<p>Currently, there is no way to inline this field.
See: <a href="https://github.com/golang/go/issues/6213">https://github.com/golang/go/issues/6213</a></p>

</td>
</tr>

<tr>

<td>

<code>tls</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TLSSpec">

TLSSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TLS defines TLS configuration for distributor receivers</p>

</td>
</tr>

</tbody>
</table>

## TempoGatewaySpec { #tempo-grafana-com-v1alpha1-TempoGatewaySpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoTemplateSpec">TempoTemplateSpec</a>)

</p>

<div>

<p>TempoGatewaySpec extends TempoComponentSpec with gateway parameters.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>component</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoComponentSpec is embedded to extend this definition with further options.</p>

<p>Currently there is no way to inline this field.
See: <a href="https://github.com/golang/go/issues/6213">https://github.com/golang/go/issues/6213</a></p>

</td>
</tr>

<tr>

<td>

<code>enabled</code><br/>

<em>

bool

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>ingress</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-IngressSpec">

IngressSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Ingress defines gateway Ingress options.</p>

</td>
</tr>

</tbody>
</table>

## TempoMonolithic { #tempo-grafana-com-v1alpha1-TempoMonolithic }

<div>

<p>TempoMonolithic manages a Tempo deployment in monolithic mode.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>metadata</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#objectmeta-v1-meta">

Kubernetes meta/v1.ObjectMeta

</a>

</em>

</td>

<td>

Refer to the Kubernetes API documentation for the fields of the

<code>metadata</code> field.

</td>
</tr>

<tr>

<td>

<code>spec</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoMonolithicSpec">

TempoMonolithicSpec

</a>

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>status</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoMonolithicStatus">

TempoMonolithicStatus

</a>

</em>

</td>

<td>

</td>
</tr>

</tbody>
</table>

## TempoMonolithicSpec { #tempo-grafana-com-v1alpha1-TempoMonolithicSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithic">TempoMonolithic</a>)

</p>

<div>

<p>TempoMonolithicSpec defines the desired state of TempoMonolithic.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>storage</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicStorageSpec">

MonolithicStorageSpec

</a>

</em>

</td>

<td>

<p>Storage defines the storage configuration.</p>

</td>
</tr>

<tr>

<td>

<code>ingestion</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicIngestionSpec">

MonolithicIngestionSpec

</a>

</em>

</td>

<td>

<p>Ingestion defines the trace ingestion configuration.</p>

</td>
</tr>

<tr>

<td>

<code>jaegerui</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicJaegerUISpec">

MonolithicJaegerUISpec

</a>

</em>

</td>

<td>

<p>JaegerUI defines the Jaeger UI configuration.</p>

</td>
</tr>

<tr>

<td>

<code>multitenancy</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicMultitenancySpec">

MonolithicMultitenancySpec

</a>

</em>

</td>

<td>

<p>Multitenancy defines the multi-tenancy configuration.</p>

</td>
</tr>

<tr>

<td>

<code>observability</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicObservabilitySpec">

MonolithicObservabilitySpec

</a>

</em>

</td>

<td>

<p>Observability defines the observability configuration of the Tempo deployment.</p>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core">

Kubernetes core/v1.ResourceRequirements

</a>

</em>

</td>

<td>

<p>Resources defines the compute resource requirements of the Tempo container.</p>

</td>
</tr>

<tr>

<td>

<code>management</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ManagementStateType">

ManagementStateType

</a>

</em>

</td>

<td>

<p>ManagementState defines whether this instance is managed by the operator or self-managed.
Default: Managed.</p>

</td>
</tr>

<tr>

<td>

<code>extraConfig</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ExtraConfigSpec">

ExtraConfigSpec

</a>

</em>

</td>

<td>

<p>ExtraConfig defines any extra (overlay) configuration of components.</p>

</td>
</tr>

</tbody>
</table>

## TempoMonolithicStatus { #tempo-grafana-com-v1alpha1-TempoMonolithicStatus }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoMonolithic">TempoMonolithic</a>)

</p>

<div>

<p>TempoMonolithicStatus defines the observed state of TempoMonolithic.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>components</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-MonolithicComponentStatus">

MonolithicComponentStatus

</a>

</em>

</td>

<td>

<p>Components provides summary of all Tempo pod status, grouped per component.</p>

</td>
</tr>

<tr>

<td>

<code>conditions</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#condition-v1-meta">

[]Kubernetes meta/v1.Condition

</a>

</em>

</td>

<td>

<p>Conditions of the Tempo deployment health.</p>

</td>
</tr>

</tbody>
</table>

## TempoQueryFrontendSpec { #tempo-grafana-com-v1alpha1-TempoQueryFrontendSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoTemplateSpec">TempoTemplateSpec</a>)

</p>

<div>

<p>TempoQueryFrontendSpec extends TempoComponentSpec with frontend specific parameters.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>component</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoComponentSpec is embedded to extend this definition with further options.</p>

<p>Currently there is no way to inline this field.
See: <a href="https://github.com/golang/go/issues/6213">https://github.com/golang/go/issues/6213</a></p>

</td>
</tr>

<tr>

<td>

<code>jaegerQuery</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-JaegerQuerySpec">

JaegerQuerySpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>JaegerQuery defines options specific to the Jaeger Query component.</p>

</td>
</tr>

</tbody>
</table>

## TempoStack { #tempo-grafana-com-v1alpha1-TempoStack }

<div>

<p>TempoStack manages a Tempo deployment in microservices mode.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>status</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoStackStatus">

TempoStackStatus

</a>

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>metadata</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#objectmeta-v1-meta">

Kubernetes meta/v1.ObjectMeta

</a>

</em>

</td>

<td>

Refer to the Kubernetes API documentation for the fields of the

<code>metadata</code> field.

</td>
</tr>

<tr>

<td>

<code>spec</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">

TempoStackSpec

</a>

</em>

</td>

<td>

</td>
</tr>

</tbody>
</table>

## TempoStackSpec { #tempo-grafana-com-v1alpha1-TempoStackSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStack">TempoStack</a>)

</p>

<div>

<p>TempoStackSpec defines the desired state of TempoStack.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>managementState</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ManagementStateType">

ManagementStateType

</a>

</em>

</td>

<td>

<p>ManagementState defines if the CR should be managed by the operator or not.
Default is managed.</p>

</td>
</tr>

<tr>

<td>

<code>limits</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-LimitSpec">

LimitSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>LimitSpec is used to limit ingestion and querying rates.</p>

</td>
</tr>

<tr>

<td>

<code>storageClassName</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>StorageClassName for PVCs used by ingester. Defaults to nil (default storage class in the cluster).</p>

</td>
</tr>

<tr>

<td>

<code>resources</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-Resources">

Resources

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Resources defines resources configuration.</p>

</td>
</tr>

<tr>

<td>

<code>storageSize</code><br/>

<em>

k8s.io/apimachinery/pkg/api/resource.Quantity

</em>

</td>

<td>

<em>(Optional)</em>

<p>StorageSize for PVCs used by ingester. Defaults to 10Gi.</p>

</td>
</tr>

<tr>

<td>

<code>images</code><br/>

<em>

<a href="../v1/feature-gates.md#tempo-grafana-com-v1alpha1-ImagesSpec">

Feature Gates.ImagesSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Images defines the image for each container.</p>

</td>
</tr>

<tr>

<td>

<code>storage</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ObjectStorageSpec">

ObjectStorageSpec

</a>

</em>

</td>

<td>

<p>Storage defines the spec for the object storage endpoint to store traces.
User is required to create secret and supply it.</p>

</td>
</tr>

<tr>

<td>

<code>retention</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-RetentionSpec">

RetentionSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>NOTE: currently this field is not considered.
Retention period defined by dataset.
User can specify how long data should be stored.</p>

</td>
</tr>

<tr>

<td>

<code>serviceAccount</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>ServiceAccount defines the service account to use for all tempo components.</p>

</td>
</tr>

<tr>

<td>

<code>search</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-SearchSpec">

SearchSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>SearchSpec control the configuration for the search capabilities.</p>

</td>
</tr>

<tr>

<td>

<code>hashRing</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-HashRingSpec">

HashRingSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>HashRing defines the spec for the distributed hash ring configuration.</p>

</td>
</tr>

<tr>

<td>

<code>template</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoTemplateSpec">

TempoTemplateSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Template defines requirements for a set of tempo components.</p>

</td>
</tr>

<tr>

<td>

<code>replicationFactor</code><br/>

<em>

int

</em>

</td>

<td>

<em>(Optional)</em>

<p>ReplicationFactor is used to define how many component replicas should exist.</p>

</td>
</tr>

<tr>

<td>

<code>tenants</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TenantsSpec">

TenantsSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tenants defines the per-tenant authentication and authorization spec.</p>

</td>
</tr>

<tr>

<td>

<code>observability</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ObservabilitySpec">

ObservabilitySpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>ObservabilitySpec defines how telemetry data gets handled.</p>

</td>
</tr>

<tr>

<td>

<code>extraConfig</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ExtraConfigSpec">

ExtraConfigSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

</td>
</tr>

</tbody>
</table>

## TempoStackStatus { #tempo-grafana-com-v1alpha1-TempoStackStatus }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStack">TempoStack</a>)

</p>

<div>

<p>TempoStackStatus defines the observed state of TempoStack.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>operatorVersion</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Version of the Tempo Operator.</p>

</td>
</tr>

<tr>

<td>

<code>tempoVersion</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Version of the managed Tempo instance.</p>

</td>
</tr>

<tr>

<td>

<code>tempoQueryVersion</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>DEPRECATED. Version of the Tempo Query component used.</p>

</td>
</tr>

<tr>

<td>

<code>components</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ComponentStatus">

ComponentStatus

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Components provides summary of all Tempo pod status grouped
per component.</p>

</td>
</tr>

<tr>

<td>

<code>conditions</code><br/>

<em>

<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#condition-v1-meta">

[]Kubernetes meta/v1.Condition

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Conditions of the Tempo deployment health.</p>

</td>
</tr>

</tbody>
</table>

## TempoTemplateSpec { #tempo-grafana-com-v1alpha1-TempoTemplateSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>TempoTemplateSpec defines the template of all requirements to configure
scheduling of all Tempo components to be deployed.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>distributor</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoDistributorSpec">

TempoDistributorSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Distributor defines the distributor component spec.</p>

</td>
</tr>

<tr>

<td>

<code>ingester</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Ingester defines the ingester component spec.</p>

</td>
</tr>

<tr>

<td>

<code>compactor</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Compactor defines the tempo compactor component spec.</p>

</td>
</tr>

<tr>

<td>

<code>querier</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoComponentSpec">

TempoComponentSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Querier defines the querier component spec.</p>

</td>
</tr>

<tr>

<td>

<code>queryFrontend</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoQueryFrontendSpec">

TempoQueryFrontendSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoQueryFrontendSpec defines the query frontend spec.</p>

</td>
</tr>

<tr>

<td>

<code>gateway</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-TempoGatewaySpec">

TempoGatewaySpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Gateway defines the tempo gateway spec.</p>

</td>
</tr>

</tbody>
</table>

## TenantSecretSpec { #tempo-grafana-com-v1alpha1-TenantSecretSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-OIDCSpec">OIDCSpec</a>)

</p>

<div>

<p>TenantSecretSpec is a secret reference containing name only
for a secret living in the same namespace as the (Tempo) TempoStack custom resource.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>name</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Name of a secret in the namespace configured for tenant secrets.</p>

</td>
</tr>

</tbody>
</table>

## TenantsSpec { #tempo-grafana-com-v1alpha1-TenantsSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-MonolithicMultitenancySpec">MonolithicMultitenancySpec</a>, <a href="#tempo-grafana-com-v1alpha1-TempoStackSpec">TempoStackSpec</a>)

</p>

<div>

<p>TenantsSpec defines the mode, authentication and authorization
configuration of the tempo gateway component.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>mode</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-ModeType">

ModeType

</a>

</em>

</td>

<td>

<p>Mode defines the multitenancy mode.</p>

</td>
</tr>

<tr>

<td>

<code>authentication</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-AuthenticationSpec">

[]AuthenticationSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Authentication defines the tempo-gateway component authentication configuration spec per tenant.</p>

</td>
</tr>

<tr>

<td>

<code>authorization</code><br/>

<em>

<a href="#tempo-grafana-com-v1alpha1-AuthorizationSpec">

AuthorizationSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Authorization defines the tempo-gateway component authorization configuration spec per tenant.</p>

</td>
</tr>

</tbody>
</table>

## TracingConfigSpec { #tempo-grafana-com-v1alpha1-TracingConfigSpec }

<p>

(<em>Appears on:</em><a href="#tempo-grafana-com-v1alpha1-ObservabilitySpec">ObservabilitySpec</a>)

</p>

<div>

<p>TracingConfigSpec defines a tracing config including endpoints and sampling.</p>

</div>

<table>

<thead>

<tr>

<th>Field</th>

<th>Description</th>

</tr>

</thead>

<tbody>

<tr>

<td>

<code>sampling_fraction</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>SamplingFraction defines the sampling ratio. Valid values are 0 to 1.</p>

</td>
</tr>

<tr>

<td>

<code>jaeger_agent_endpoint</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>JaegerAgentEndpoint defines the jaeger endpoint data gets send to.</p>

</td>
</tr>

</tbody>
</table>

<hr/>

+newline

