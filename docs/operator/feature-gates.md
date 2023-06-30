
---
title: "Feature Gates"
description: "Generated API docs for the Tempo Operator"
lead: ""
draft: false
images: []
menu:
  docs:
    parent: "operator"
weight: 1000
toc: true
---

This Document contains the types introduced by the Tempo Operator to be consumed by users.

> This page is automatically generated with `gen-crd-api-reference-docs`.

# config.tempo.grafana.com/v1alpha1 { #config-tempo-grafana-com-v1alpha1 }

<div>

<p>Package v1alpha1 contains API Schema definitions for the config.tempo v1alpha1 API group.</p>

</div>

<b>Resource Types:</b>


## BuiltInCertManagement { #config-tempo-grafana-com-v1alpha1-BuiltInCertManagement }

<p>

(<em>Appears on:</em><a href="#config-tempo-grafana-com-v1alpha1-FeatureGates">FeatureGates</a>)

</p>

<div>

<p>BuiltInCertManagement is the configuration for the built-in facility to generate and rotate
TLS client and serving certificates for all Tempo services and internal clients. All necessary
secrets and configmaps for protecting the internal components will be created if this option is enabled.</p>

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

<code>caValidity</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<p>CACertValidity defines the total duration of the CA certificate validity.</p>

</td>
</tr>

<tr>

<td>

<code>caRefresh</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<p>CACertRefresh defines the duration of the CA certificate validity until a rotation
should happen. It can be set up to 80% of CA certificate validity or equal to the
CA certificate validity. Latter should be used only for rotating only when expired.</p>

</td>
</tr>

<tr>

<td>

<code>certValidity</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<p>CertValidity defines the total duration of the validity for all Tempo certificates.</p>

</td>
</tr>

<tr>

<td>

<code>certRefresh</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<p>CertRefresh defines the duration of the certificate validity until a rotation
should happen. It can be set up to 80% of certificate validity or equal to the
certificate validity. Latter should be used only for rotating only when expired.
The refresh is applied to all Tempo certificates at once.</p>

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

<p>Enabled defines to flag to enable/disable built-in certificate management feature gate.</p>

</td>
</tr>

</tbody>
</table>


## FeatureGates { #config-tempo-grafana-com-v1alpha1-FeatureGates }

<p>

(<em>Appears on:</em><a href="#config-tempo-grafana-com-v1alpha1-ProjectConfig">ProjectConfig</a>)

</p>

<div>

<p>FeatureGates is the supported set of all operator feature gates.</p>

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

<code>openshift</code><br/>

<em>

<a href="#config-tempo-grafana-com-v1alpha1-OpenShiftFeatureGates">

OpenShiftFeatureGates

</a>

</em>

</td>

<td>

<p>OpenShift contains a set of feature gates supported only on OpenShift.</p>

</td>
</tr>

<tr>

<td>

<code>builtInCertManagement</code><br/>

<em>

<a href="#config-tempo-grafana-com-v1alpha1-BuiltInCertManagement">

BuiltInCertManagement

</a>

</em>

</td>

<td>

<p>BuiltInCertManagement enables the built-in facility for generating and rotating
TLS client and serving certificates for the communication between ingesters and distributors and also between
query and query-frontend, In detail all internal Tempo HTTP and GRPC communication is lifted
to require mTLS.
In addition each service requires a configmap named as the MicroService CR with the
suffix <code>-ca-bundle</code>, e.g. <code>tempo-dev-ca-bundle</code> and the following data:
- <code>service-ca.crt</code>: The CA signing the service certificate in <code>tls.crt</code>.
All necessary secrets and configmaps for protecting the internal components will be created if this
option is enabled.</p>

</td>
</tr>

<tr>

<td>

<code>httpEncryption</code><br/>

<em>

bool

</em>

</td>

<td>

<p>HTTPEncryption enables TLS encryption for all HTTP TempoStack components.
Each HTTP component requires a secret, the name should be the name of the component with the
suffix <code>-mtls</code> and prefix by the TempoStack name e.g <code>tempo-dev-distributor-mtls</code>.
It should contains the following data:
- <code>tls.crt</code>: The TLS server side certificate.
- <code>tls.key</code>: The TLS key for server-side encryption.
In addition each service requires a configmap named as the TempoStack CR with the
suffix <code>-ca-bundle</code>, e.g. <code>tempo-dev-ca-bundle</code> and the following data:
- <code>service-ca.crt</code>: The CA signing the service certificate in <code>tls.crt</code>.
This will protect all internal communication between the distributors and ingestors and also
between ingestor and queriers, and between the queriers and the query-frontend component</p>

<p>If BuiltInCertManagement is enabled, you don&rsquo;t need to create this secrets manually.</p>

<p>Some considerations when enable mTLS:
- If JaegerUI is enabled, it won&rsquo;t be protected by mTLS as it will be considered a public facing
component.
- If JaegerUI is not enabled, HTTP Tempo API won´t be protected, this will be considered
public faced component.
- If Gateway is enabled, all comunications between the gateway and the tempo components will be protected
by mTLS, and the Gateway itself won´t be, as it will be the only public face component.</p>

</td>
</tr>

<tr>

<td>

<code>grpcEncryption</code><br/>

<em>

bool

</em>

</td>

<td>

<p>GRPCEncryption enables TLS encryption for all GRPC TempoStack services.
Each GRPC component requires a secret, the name should be the name of the component with the
suffix <code>-mtls</code> and prefix by the TempoStack name e.g <code>tempo-dev-distributor-mtls</code>.
It should contains the following data:
- <code>tls.crt</code>: The TLS server side certificate.
- <code>tls.key</code>: The TLS key for server-side encryption.
In addition each service requires a configmap named as the TempoStack CR with the
suffix <code>-ca-bundle</code>, e.g. <code>tempo-dev-ca-bundle</code> and the following data:
- <code>service-ca.crt</code>: The CA signing the service certificate in <code>tls.crt</code>.
This will protect all internal communication between the distributors and ingestors and also
between ingestor and queriers, and between the queriers and the query-frontend component.</p>

<p>If BuiltInCertManagement is enabled, you don&rsquo;t need to create this secrets manually.</p>

<p>Some considerations when enable mTLS:
- If JaegerUI is enabled, it won´t be protected by mTLS as it will be considered a public face
component.
- If Gateway is enabled, all comunications between the gateway and the tempo components will be protected
by mTLS, and the Gateway itself won´t be, as it will be the only public face component.</p>

</td>
</tr>

<tr>

<td>

<code>tlsProfile</code><br/>

<em>

string

</em>

</td>

<td>

<p>TLSProfile allows to chose a TLS security profile. Enforced
when using HTTPEncryption or GRPCEncryption.</p>

</td>
</tr>

<tr>

<td>

<code>prometheusOperator</code><br/>

<em>

bool

</em>

</td>

<td>

<p>PrometheusOperator defines whether the Prometheus Operator CRD exists in the cluster.
This CRD is part of prometheus-operator.</p>

</td>
</tr>

</tbody>
</table>


## ImagesSpec { #config-tempo-grafana-com-v1alpha1-ImagesSpec }

<p>

(<em>Appears on:</em><a href="#config-tempo-grafana-com-v1alpha1-ProjectConfig">ProjectConfig</a>)

</p>

<div>

<p>ImagesSpec defines the image for each container.</p>

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

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>Tempo defines the tempo container image.</p>

</td>
</tr>

<tr>

<td>

<code>tempoQuery</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoQuery defines the tempo-query container image.</p>

</td>
</tr>

<tr>

<td>

<code>tempoGateway</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoGateway defines the tempo-gateway container image.</p>

</td>
</tr>

<tr>

<td>

<code>tempoGatewayOpa</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>TempoGatewayOpa defines the OPA sidecar container for TempoGateway.</p>

</td>
</tr>

</tbody>
</table>


## OpenShiftFeatureGates { #config-tempo-grafana-com-v1alpha1-OpenShiftFeatureGates }

<p>

(<em>Appears on:</em><a href="#config-tempo-grafana-com-v1alpha1-FeatureGates">FeatureGates</a>)

</p>

<div>

<p>OpenShiftFeatureGates is the supported set of all operator features gates on OpenShift.</p>

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

<code>servingCertsService</code><br/>

<em>

bool

</em>

</td>

<td>

<p>ServingCertsService enables OpenShift service-ca annotations on the TempoStack gateway service only
to use the in-platform CA and generate a TLS cert/key pair per service for
in-cluster data-in-transit encryption.
More details: <a href="https://docs.openshift.com/container-platform/latest/security/certificate_types_descriptions/service-ca-certificates.html">https://docs.openshift.com/container-platform/latest/security/certificate_types_descriptions/service-ca-certificates.html</a></p>

</td>
</tr>

<tr>

<td>

<code>openshiftRoute</code><br/>

<em>

bool

</em>

</td>

<td>

<p>OpenShiftRoute enables creating OpenShift Route objects.
More details: <a href="https://docs.openshift.com/container-platform/latest/networking/understanding-networking.html">https://docs.openshift.com/container-platform/latest/networking/understanding-networking.html</a></p>

</td>
</tr>

<tr>

<td>

<code>baseDomain</code><br/>

<em>

string

</em>

</td>

<td>

<p>BaseDomain is used internally for redirect URL in gateway OpenShift auth mode.
If empty the operator automatically derives the domain from the cluster.</p>

</td>
</tr>

<tr>

<td>

<code>ClusterTLSPolicy</code><br/>

<em>

bool

</em>

</td>

<td>

<p>ClusterTLSPolicy enables usage of TLS policies set in the API Server.
More details: <a href="https://docs.openshift.com/container-platform/4.11/security/tls-security-profiles.html">https://docs.openshift.com/container-platform/4.11/security/tls-security-profiles.html</a></p>

</td>
</tr>

</tbody>
</table>


## ProjectConfig { #config-tempo-grafana-com-v1alpha1-ProjectConfig }

<div>

<p>ProjectConfig is the Schema for the projectconfigs API.</p>

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

<code>syncPeriod</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>SyncPeriod determines the minimum frequency at which watched resources are
reconciled. A lower period will correct entropy more quickly, but reduce
responsiveness to change if there are many watched resources. Change this
value only if you know what you are doing. Defaults to 10 hours if unset.
there will a 10 percent jitter between the SyncPeriod of all controllers
so that all controllers will not send list requests simultaneously.</p>

</td>
</tr>

<tr>

<td>

<code>leaderElection</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/component-base/config#LeaderElectionConfiguration">

Kubernetes v1alpha1.LeaderElectionConfiguration

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>LeaderElection is the LeaderElection config to be used when configuring
the manager.Manager leader election</p>

</td>
</tr>

<tr>

<td>

<code>cacheNamespace</code><br/>

<em>

string

</em>

</td>

<td>

<em>(Optional)</em>

<p>CacheNamespace if specified restricts the manager&rsquo;s cache to watch objects in
the desired namespace Defaults to all namespaces</p>

<p>Note: If a namespace is specified, controllers can still Watch for a
cluster-scoped resource (e.g Node).  For namespaced resources the cache
will only hold objects from the desired namespace.</p>

</td>
</tr>

<tr>

<td>

<code>gracefulShutDown</code><br/>

<em>

<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">

Kubernetes meta/v1.Duration

</a>

</em>

</td>

<td>

<p>GracefulShutdownTimeout is the duration given to runnable to stop before the manager actually returns on stop.
To disable graceful shutdown, set to time.Duration(0)
To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1)
The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.</p>

</td>
</tr>

<tr>

<td>

<code>controller</code><br/>

<em>

<a href="https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerConfigurationSpec">

K8S Controller-runtime v1alpha1.ControllerConfigurationSpec

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Controller contains global configuration options for controllers
registered within this manager.</p>

</td>
</tr>

<tr>

<td>

<code>metrics</code><br/>

<em>

<a href="https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerMetrics">

K8S Controller-runtime v1alpha1.ControllerMetrics

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Metrics contains the controller metrics configuration</p>

</td>
</tr>

<tr>

<td>

<code>health</code><br/>

<em>

<a href="https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerHealth">

K8S Controller-runtime v1alpha1.ControllerHealth

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Health contains the controller health configuration</p>

</td>
</tr>

<tr>

<td>

<code>webhook</code><br/>

<em>

<a href="https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/config/v1alpha1#ControllerWebhook">

K8S Controller-runtime v1alpha1.ControllerWebhook

</a>

</em>

</td>

<td>

<em>(Optional)</em>

<p>Webhook contains the controllers webhook configuration</p>

</td>
</tr>

<tr>

<td>

<code>images</code><br/>

<em>

<a href="#config-tempo-grafana-com-v1alpha1-ImagesSpec">

ImagesSpec

</a>

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>featureGates</code><br/>

<em>

<a href="#config-tempo-grafana-com-v1alpha1-FeatureGates">

FeatureGates

</a>

</em>

</td>

<td>

</td>
</tr>

<tr>

<td>

<code>distribution</code><br/>

<em>

string

</em>

</td>

<td>

<p>Distribution defines the operator distribution name.</p>

</td>
</tr>

</tbody>
</table>


## TLSProfileType { #config-tempo-grafana-com-v1alpha1-TLSProfileType }

(<code>string</code> alias)

<div>

<p>TLSProfileType is a TLS security profile based on the Mozilla definitions:
<a href="https://wiki.mozilla.org/Security/Server_Side_TLS">https://wiki.mozilla.org/Security/Server_Side_TLS</a></p>

</div>

<table>

<thead>

<tr>

<th>Value</th>

<th>Description</th>

</tr>

</thead>

<tbody><tr><td><p>&#34;Intermediate&#34;</p></td>

<td><p>TLSProfileIntermediateType is a TLS security profile based on:
<a href="https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28default.29">https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28default.29</a></p>
</td>

</tr><tr><td><p>&#34;Modern&#34;</p></td>

<td><p>TLSProfileModernType is a TLS security profile based on:
<a href="https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility">https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility</a></p>
</td>

</tr><tr><td><p>&#34;Old&#34;</p></td>

<td><p>TLSProfileOldType is a TLS security profile based on:
<a href="https://wiki.mozilla.org/Security/Server_Side_TLS#Old_backward_compatibility">https://wiki.mozilla.org/Security/Server_Side_TLS#Old_backward_compatibility</a></p>
</td>

</tr></tbody>
</table>

<hr/>




