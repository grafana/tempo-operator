# API Reference

Packages:

- [tempo.grafana.com/v1alpha1](#tempografanacomv1alpha1)

# tempo.grafana.com/v1alpha1

Resource Types:

- [Microservices](#microservices)




## Microservices
<sup><sup>[↩ Parent](#tempografanacomv1alpha1 )</sup></sup>






Microservices is the Schema for the microservices API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>tempo.grafana.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Microservices</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspec">spec</a></b></td>
        <td>object</td>
        <td>
          MicroservicesSpec defines the desired state of Microservices.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesstatus">status</a></b></td>
        <td>object</td>
        <td>
          MicroservicesStatus defines the observed state of Microservices.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec
<sup><sup>[↩ Parent](#microservices)</sup></sup>



MicroservicesSpec defines the desired state of Microservices.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspecstorage">storage</a></b></td>
        <td>object</td>
        <td>
          Storage defines S3 compatible object storage configuration. User is required to create secret and supply it.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspecimages">images</a></b></td>
        <td>object</td>
        <td>
          Images defines the image for each container.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspeclimits">limits</a></b></td>
        <td>object</td>
        <td>
          LimitSpec is used to limit ingestion and querying rates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicationFactor</b></td>
        <td>integer</td>
        <td>
          NOTE: currently this field is not considered. ReplicationFactor is used to define how many component replicas should exist.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspecresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources defines resources configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspecretention">retention</a></b></td>
        <td>object</td>
        <td>
          NOTE: currently this field is not considered. Retention period defined by dataset. User can specify how long data should be stored.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspecsearch">search</a></b></td>
        <td>object</td>
        <td>
          SearchSpec control the configuration for the search capabilities.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceAccount</b></td>
        <td>string</td>
        <td>
          ServiceAccount defines the service account to use for all tempo components.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          StorageClassName for PVCs used by ingester. Defaults to nil (default storage class in the cluster).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageSize</b></td>
        <td>int or string</td>
        <td>
          StorageSize for PVCs used by ingester. Defaults to 10Gi.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplate">template</a></b></td>
        <td>object</td>
        <td>
          Components defines requirements for a set of tempo components.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenants">tenants</a></b></td>
        <td>object</td>
        <td>
          Tenants defines the per-tenant authentication and authorization spec.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.storage
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



Storage defines S3 compatible object storage configuration. User is required to create secret and supply it.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret for object storage authentication. Name of a secret in the same namespace as the tempo Microservices custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspecstoragetls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS configuration for reaching the object storage endpoint.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.storage.tls
<sup><sup>[↩ Parent](#microservicesspecstorage)</sup></sup>



TLS configuration for reaching the object storage endpoint.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>caName</b></td>
        <td>string</td>
        <td>
          CA is the name of a ConfigMap containing a CA certificate. It needs to be in the same namespace as the Tempo custom resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.images
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



Images defines the image for each container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>tempo</b></td>
        <td>string</td>
        <td>
          Tempo defines the tempo container image.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tempoGateway</b></td>
        <td>string</td>
        <td>
          TempoGateway defines the tempo-gateway container image.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tempoQuery</b></td>
        <td>string</td>
        <td>
          TempoQuery defines the tempo-query container image.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



LimitSpec is used to limit ingestion and querying rates.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspeclimitsglobal">global</a></b></td>
        <td>object</td>
        <td>
          Global is used to define global rate limits.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspeclimitspertenantkey">perTenant</a></b></td>
        <td>map[string]object</td>
        <td>
          PerTenant is used to define rate limits per tenant.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.global
<sup><sup>[↩ Parent](#microservicesspeclimits)</sup></sup>



Global is used to define global rate limits.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspeclimitsglobalingestion">ingestion</a></b></td>
        <td>object</td>
        <td>
          Ingestion is used to define ingestion rate limits.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspeclimitsglobalquery">query</a></b></td>
        <td>object</td>
        <td>
          Query is used to define query rate limits.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.global.ingestion
<sup><sup>[↩ Parent](#microservicesspeclimitsglobal)</sup></sup>



Ingestion is used to define ingestion rate limits.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ingestionBurstSizeBytes</b></td>
        <td>integer</td>
        <td>
          IngestionBurstSizeBytes defines the burst size (bytes) used in ingestion.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ingestionRateLimitBytes</b></td>
        <td>integer</td>
        <td>
          IngestionRateLimitBytes defines the Per-user ingestion rate limit (bytes) used in ingestion.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytesPerTrace</b></td>
        <td>integer</td>
        <td>
          MaxBytesPerTrace defines the maximum number of bytes of an acceptable trace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxTracesPerUser</b></td>
        <td>integer</td>
        <td>
          MaxTracesPerUser defines the maximum number of traces a user can send.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.global.query
<sup><sup>[↩ Parent](#microservicesspeclimitsglobal)</sup></sup>



Query is used to define query rate limits.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxBytesPerTagValues</b></td>
        <td>integer</td>
        <td>
          MaxBytesPerTagValues defines the maximum size in bytes of a tag-values query.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxSearchBytesPerTrace</b></td>
        <td>integer</td>
        <td>
          MaxSearchBytesPerTrace defines the maximum size of search data for a single trace in bytes. default: `0` to disable.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.perTenant[key]
<sup><sup>[↩ Parent](#microservicesspeclimits)</sup></sup>



RateLimitSpec defines rate limits for Ingestion and Query components.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspeclimitspertenantkeyingestion">ingestion</a></b></td>
        <td>object</td>
        <td>
          Ingestion is used to define ingestion rate limits.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspeclimitspertenantkeyquery">query</a></b></td>
        <td>object</td>
        <td>
          Query is used to define query rate limits.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.perTenant[key].ingestion
<sup><sup>[↩ Parent](#microservicesspeclimitspertenantkey)</sup></sup>



Ingestion is used to define ingestion rate limits.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ingestionBurstSizeBytes</b></td>
        <td>integer</td>
        <td>
          IngestionBurstSizeBytes defines the burst size (bytes) used in ingestion.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ingestionRateLimitBytes</b></td>
        <td>integer</td>
        <td>
          IngestionRateLimitBytes defines the Per-user ingestion rate limit (bytes) used in ingestion.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxBytesPerTrace</b></td>
        <td>integer</td>
        <td>
          MaxBytesPerTrace defines the maximum number of bytes of an acceptable trace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxTracesPerUser</b></td>
        <td>integer</td>
        <td>
          MaxTracesPerUser defines the maximum number of traces a user can send.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits.perTenant[key].query
<sup><sup>[↩ Parent](#microservicesspeclimitspertenantkey)</sup></sup>



Query is used to define query rate limits.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxBytesPerTagValues</b></td>
        <td>integer</td>
        <td>
          MaxBytesPerTagValues defines the maximum size in bytes of a tag-values query.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxSearchBytesPerTrace</b></td>
        <td>integer</td>
        <td>
          MaxSearchBytesPerTrace defines the maximum size of search data for a single trace in bytes. default: `0` to disable.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.resources
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



Resources defines resources configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspecresourcestotal">total</a></b></td>
        <td>object</td>
        <td>
          The total amount of resources for Tempo instance. The operator autonomously splits resources between deployed Tempo components. Only limits are supported, the operator calculates requests automatically. See http://github.com/grafana/tempo/issues/1540.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.resources.total
<sup><sup>[↩ Parent](#microservicesspecresources)</sup></sup>



The total amount of resources for Tempo instance. The operator autonomously splits resources between deployed Tempo components. Only limits are supported, the operator calculates requests automatically. See http://github.com/grafana/tempo/issues/1540.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.retention
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



NOTE: currently this field is not considered. Retention period defined by dataset. User can specify how long data should be stored.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspecretentionglobal">global</a></b></td>
        <td>object</td>
        <td>
          Global is used to configure global retention.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspecretentionpertenantkey">perTenant</a></b></td>
        <td>map[string]object</td>
        <td>
          PerTenant is used to configure retention per tenant.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.retention.global
<sup><sup>[↩ Parent](#microservicesspecretention)</sup></sup>



Global is used to configure global retention.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>traces</b></td>
        <td>string</td>
        <td>
          Traces defines retention period. Supported parameter suffixes are “s”, “m” and “h”. example: 336h default: value is 48h.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.retention.perTenant[key]
<sup><sup>[↩ Parent](#microservicesspecretention)</sup></sup>



RetentionConfig defines how long data should be provided.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>traces</b></td>
        <td>string</td>
        <td>
          Traces defines retention period. Supported parameter suffixes are “s”, “m” and “h”. example: 336h default: value is 48h.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.search
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



SearchSpec control the configuration for the search capabilities.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultResultLimit</b></td>
        <td>integer</td>
        <td>
          Limit used for search requests if none is set by the caller (default: 20)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enable tempo search feature, default to true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxDuration</b></td>
        <td>string</td>
        <td>
          The maximum allowed time range for a search, default: 0s which means unlimited.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxResultLimit</b></td>
        <td>integer</td>
        <td>
          The maximum allowed value of the limit parameter on search requests. If the search request limit parameter exceeds the value configured here it will be set to the value configured here. The default value of 0 disables this limit.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



Components defines requirements for a set of tempo components.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspectemplatecompactor">compactor</a></b></td>
        <td>object</td>
        <td>
          Compactor defines the tempo compactor component spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatedistributor">distributor</a></b></td>
        <td>object</td>
        <td>
          Distributor defines the distributor component spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplategateway">gateway</a></b></td>
        <td>object</td>
        <td>
          Gateway defines the tempo gateway spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplateingester">ingester</a></b></td>
        <td>object</td>
        <td>
          Ingester defines the ingester component spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatequerier">querier</a></b></td>
        <td>object</td>
        <td>
          Querier defines the querier component spec.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatequeryfrontend">queryFrontend</a></b></td>
        <td>object</td>
        <td>
          TempoQueryFrontendSpec defines the query frontend spec.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.compactor
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



Compactor defines the tempo compactor component spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatecompactortolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.compactor.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplatecompactor)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.distributor
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



Distributor defines the distributor component spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatedistributortolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.distributor.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplatedistributor)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.gateway
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



Gateway defines the tempo gateway spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspectemplategatewaycomponent">component</a></b></td>
        <td>object</td>
        <td>
          TempoComponentSpec is embedded to extend this definition with further options. 
 Currently there is no way to inline this field. See: https://github.com/golang/go/issues/6213<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplategatewaytempogateway">tempoGateway</a></b></td>
        <td>object</td>
        <td>
          Gateway defines Tempo gateway spefic options.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.gateway.component
<sup><sup>[↩ Parent](#microservicesspectemplategateway)</sup></sup>



TempoComponentSpec is embedded to extend this definition with further options. 
 Currently there is no way to inline this field. See: https://github.com/golang/go/issues/6213

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplategatewaycomponenttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.gateway.component.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplategatewaycomponent)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.gateway.tempoGateway
<sup><sup>[↩ Parent](#microservicesspectemplategateway)</sup></sup>



Gateway defines Tempo gateway spefic options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is used to define if a tempo gateway component should be created.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.ingester
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



Ingester defines the ingester component spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplateingestertolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.ingester.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplateingester)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.querier
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



Querier defines the querier component spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatequeriertolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.querier.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplatequerier)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.queryFrontend
<sup><sup>[↩ Parent](#microservicesspectemplate)</sup></sup>



TempoQueryFrontendSpec defines the query frontend spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspectemplatequeryfrontendcomponent">component</a></b></td>
        <td>object</td>
        <td>
          TempoComponentSpec is embedded to extend this definition with further options. 
 Currently there is no way to inline this field. See: https://github.com/golang/go/issues/6213<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatequeryfrontendjaegerquery">jaegerQuery</a></b></td>
        <td>object</td>
        <td>
          JaegerQuerySpec defines Jaeger Query spefic options.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.queryFrontend.component
<sup><sup>[↩ Parent](#microservicesspectemplatequeryfrontend)</sup></sup>



TempoComponentSpec is embedded to extend this definition with further options. 
 Currently there is no way to inline this field. See: https://github.com/golang/go/issues/6213

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          NodeSelector is the simplest recommended form of node selection constraint.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas represents the number of replicas to create for this component.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplatequeryfrontendcomponenttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Tolerations defines component specific pod tolerations.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.queryFrontend.component.tolerations[index]
<sup><sup>[↩ Parent](#microservicesspectemplatequeryfrontendcomponent)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template.queryFrontend.jaegerQuery
<sup><sup>[↩ Parent](#microservicesspectemplatequeryfrontend)</sup></sup>



JaegerQuerySpec defines Jaeger Query spefic options.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enabled is used to define if Jaeger Query component should be created.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



Tenants defines the per-tenant authentication and authorization spec.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          Mode defines the multitenancy mode.<br/>
          <br/>
            <i>Enum</i>: static, dynamic<br/>
            <i>Default</i>: static<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthenticationindex">authentication</a></b></td>
        <td>[]object</td>
        <td>
          Authentication defines the tempo-gateway component authentication configuration spec per tenant.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthorization">authorization</a></b></td>
        <td>object</td>
        <td>
          Authorization defines the tempo-gateway component authorization configuration spec per tenant.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authentication[index]
<sup><sup>[↩ Parent](#microservicesspectenants)</sup></sup>



AuthenticationSpec defines the oidc configuration per tenant for tempo Gateway component.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>tenantId</b></td>
        <td>string</td>
        <td>
          TenantID defines the id of the tenant.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>tenantName</b></td>
        <td>string</td>
        <td>
          TenantName defines the name of the tenant.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthenticationindexoidc">oidc</a></b></td>
        <td>object</td>
        <td>
          OIDC defines the spec for the OIDC tenant's authentication.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authentication[index].oidc
<sup><sup>[↩ Parent](#microservicesspectenantsauthenticationindex)</sup></sup>



OIDC defines the spec for the OIDC tenant's authentication.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>issuerURL</b></td>
        <td>string</td>
        <td>
          IssuerURL defines the URL for issuer.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthenticationindexoidcsecret">secret</a></b></td>
        <td>object</td>
        <td>
          Secret defines the spec for the clientID, clientSecret and issuerCAPath for tenant's authentication.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupClaim</b></td>
        <td>string</td>
        <td>
          Group claim field from ID Token<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>redirectURL</b></td>
        <td>string</td>
        <td>
          RedirectURL defines the URL for redirect.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>usernameClaim</b></td>
        <td>string</td>
        <td>
          User claim field from ID Token<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authentication[index].oidc.secret
<sup><sup>[↩ Parent](#microservicesspectenantsauthenticationindexoidc)</sup></sup>



Secret defines the spec for the clientID, clientSecret and issuerCAPath for tenant's authentication.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of a secret in the namespace configured for tenant secrets.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authorization
<sup><sup>[↩ Parent](#microservicesspectenants)</sup></sup>



Authorization defines the tempo-gateway component authorization configuration spec per tenant.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesspectenantsauthorizationrolebindingsindex">roleBindings</a></b></td>
        <td>[]object</td>
        <td>
          RoleBindings defines configuration to bind a set of roles to a set of subjects.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthorizationrolesindex">roles</a></b></td>
        <td>[]object</td>
        <td>
          Roles defines a set of permissions to interact with a tenant.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authorization.roleBindings[index]
<sup><sup>[↩ Parent](#microservicesspectenantsauthorization)</sup></sup>



RoleBindingsSpec binds a set of roles to a set of subjects.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#microservicesspectenantsauthorizationrolebindingsindexsubjectsindex">subjects</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authorization.roleBindings[index].subjects[index]
<sup><sup>[↩ Parent](#microservicesspectenantsauthorizationrolebindingsindex)</sup></sup>



Subject represents a subject that has been bound to a role.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>enum</td>
        <td>
          SubjectKind is a kind of Tempo Gateway RBAC subject.<br/>
          <br/>
            <i>Enum</i>: user, group<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Microservices.spec.tenants.authorization.roles[index]
<sup><sup>[↩ Parent](#microservicesspectenantsauthorization)</sup></sup>



RoleSpec describes a set of permissions to interact with a tenant.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>permissions</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>resources</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>tenants</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Microservices.status
<sup><sup>[↩ Parent](#microservices)</sup></sup>



MicroservicesStatus defines the observed state of Microservices.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#microservicesstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Conditions of the Tempo deployment health.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tempoVersion</b></td>
        <td>string</td>
        <td>
          Version of the managed Tempo instance.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.status.conditions[index]
<sup><sup>[↩ Parent](#microservicesstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource. --- This struct is intended for direct use as an array at the field path .status.conditions.  For example, 
 type FooStatus struct{ // Represents the observations of a foo's current state. // Known .status.conditions.type are: "Available", "Progressing", and "Degraded" // +patchMergeKey=type // +patchStrategy=merge // +listType=map // +listMapKey=type Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"` 
 // other fields }

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition. This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase. --- Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important. The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>