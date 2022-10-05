# API Reference

Packages:

- [tempo.grafana.com/v1alpha1](#tempografanacomv1alpha1)
- [config.tempo.grafana.com/v1alpha1](#configtempografanacomv1alpha1)

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
        <td><b>status</b></td>
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
        <td><b><a href="#microservicesspeclimits">limits</a></b></td>
        <td>object</td>
        <td>
          NOTE: currently this field is not considered. LimitSpec is used to limit ingestion and querying rates.<br/>
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
          NOTE: currently this field is not considered. The resources are split in between components. Tempo operator knows how to split them appropriately based on grafana/tempo/issues/1540.<br/>
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
        <td><b><a href="#microservicesspecstorage">storage</a></b></td>
        <td>object</td>
        <td>
          NOTE: currently this field is not considered. Storage defines S3 compatible object storage configuration. User is required to create secret and supply it.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>
          NOTE: currently this field is not considered. StorageClassName for PVCs used by ingester/querier.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#microservicesspectemplate">template</a></b></td>
        <td>object</td>
        <td>
          NOTE: currently this field is not considered. Components defines requierements for a set of tempo components.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.limits
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



NOTE: currently this field is not considered. LimitSpec is used to limit ingestion and querying rates.

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



NOTE: currently this field is not considered. The resources are split in between components. Tempo operator knows how to split them appropriately based on grafana/tempo/issues/1540.

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
          Traces defines retention period. Supported parameter suffixes are “s”, “m” and “h”. example: 336h<br/>
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
          Traces defines retention period. Supported parameter suffixes are “s”, “m” and “h”. example: 336h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.storage
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



NOTE: currently this field is not considered. Storage defines S3 compatible object storage configuration. User is required to create secret and supply it.

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
        <td>false</td>
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
          CA is the name of a ConfigMap containing a CA certificate. It needs to be in the same namespace as the LokiStack custom resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Microservices.spec.template
<sup><sup>[↩ Parent](#microservicesspec)</sup></sup>



NOTE: currently this field is not considered. Components defines requierements for a set of tempo components.

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
          Compactor defines the lokistack compactor component spec.<br/>
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



Compactor defines the lokistack compactor component spec.

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

# config.tempo.grafana.com/v1alpha1

Resource Types:

- [ProjectConfig](#projectconfig)




## ProjectConfig
<sup><sup>[↩ Parent](#configtempografanacomv1alpha1 )</sup></sup>






ProjectConfig is the Schema for the projectconfigs API.

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
      <td>config.tempo.grafana.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ProjectConfig</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b>cacheNamespace</b></td>
        <td>string</td>
        <td>
          CacheNamespace if specified restricts the manager's cache to watch objects in the desired namespace Defaults to all namespaces 
 Note: If a namespace is specified, controllers can still Watch for a cluster-scoped resource (e.g Node).  For namespaced resources the cache will only hold objects from the desired namespace.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#projectconfigcontroller">controller</a></b></td>
        <td>object</td>
        <td>
          Controller contains global configuration options for controllers registered within this manager.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gracefulShutDown</b></td>
        <td>string</td>
        <td>
          GracefulShutdownTimeout is the duration given to runnable to stop before the manager actually returns on stop. To disable graceful shutdown, set to time.Duration(0) To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1) The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#projectconfighealth">health</a></b></td>
        <td>object</td>
        <td>
          Health contains the controller health configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#projectconfigleaderelection">leaderElection</a></b></td>
        <td>object</td>
        <td>
          LeaderElection is the LeaderElection config to be used when configuring the manager.Manager leader election<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#projectconfigmetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          Metrics contains thw controller metrics configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>syncPeriod</b></td>
        <td>string</td>
        <td>
          SyncPeriod determines the minimum frequency at which watched resources are reconciled. A lower period will correct entropy more quickly, but reduce responsiveness to change if there are many watched resources. Change this value only if you know what you are doing. Defaults to 10 hours if unset. there will a 10 percent jitter between the SyncPeriod of all controllers so that all controllers will not send list requests simultaneously.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#projectconfigwebhook">webhook</a></b></td>
        <td>object</td>
        <td>
          Webhook contains the controllers webhook configuration<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ProjectConfig.controller
<sup><sup>[↩ Parent](#projectconfig)</sup></sup>



Controller contains global configuration options for controllers registered within this manager.

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
        <td><b>cacheSyncTimeout</b></td>
        <td>integer</td>
        <td>
          CacheSyncTimeout refers to the time limit set to wait for syncing caches. Defaults to 2 minutes if not set.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupKindConcurrency</b></td>
        <td>map[string]integer</td>
        <td>
          GroupKindConcurrency is a map from a Kind to the number of concurrent reconciliation allowed for that controller. 
 When a controller is registered within this manager using the builder utilities, users have to specify the type the controller reconciles in the For(...) call. If the object's kind passed matches one of the keys in this map, the concurrency for that controller is set to the number specified. 
 The key is expected to be consistent in form with GroupKind.String(), e.g. ReplicaSet in apps group (regardless of version) would be `ReplicaSet.apps`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ProjectConfig.health
<sup><sup>[↩ Parent](#projectconfig)</sup></sup>



Health contains the controller health configuration

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
        <td><b>healthProbeBindAddress</b></td>
        <td>string</td>
        <td>
          HealthProbeBindAddress is the TCP address that the controller should bind to for serving health probes<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>livenessEndpointName</b></td>
        <td>string</td>
        <td>
          LivenessEndpointName, defaults to "healthz"<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readinessEndpointName</b></td>
        <td>string</td>
        <td>
          ReadinessEndpointName, defaults to "readyz"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ProjectConfig.leaderElection
<sup><sup>[↩ Parent](#projectconfig)</sup></sup>



LeaderElection is the LeaderElection config to be used when configuring the manager.Manager leader election

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
        <td><b>leaderElect</b></td>
        <td>boolean</td>
        <td>
          leaderElect enables a leader election client to gain leadership before executing the main loop. Enable this when running replicated components for high availability.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>leaseDuration</b></td>
        <td>string</td>
        <td>
          leaseDuration is the duration that non-leader candidates will wait after observing a leadership renewal until attempting to acquire leadership of a led but unrenewed leader slot. This is effectively the maximum duration that a leader can be stopped before it is replaced by another candidate. This is only applicable if leader election is enabled.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>renewDeadline</b></td>
        <td>string</td>
        <td>
          renewDeadline is the interval between attempts by the acting master to renew a leadership slot before it stops leading. This must be less than or equal to the lease duration. This is only applicable if leader election is enabled.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>resourceLock</b></td>
        <td>string</td>
        <td>
          resourceLock indicates the resource object type that will be used to lock during leader election cycles.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>resourceName</b></td>
        <td>string</td>
        <td>
          resourceName indicates the name of resource object that will be used to lock during leader election cycles.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>resourceNamespace</b></td>
        <td>string</td>
        <td>
          resourceName indicates the namespace of resource object that will be used to lock during leader election cycles.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>retryPeriod</b></td>
        <td>string</td>
        <td>
          retryPeriod is the duration the clients should wait between attempting acquisition and renewal of a leadership. This is only applicable if leader election is enabled.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ProjectConfig.metrics
<sup><sup>[↩ Parent](#projectconfig)</sup></sup>



Metrics contains thw controller metrics configuration

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
        <td><b>bindAddress</b></td>
        <td>string</td>
        <td>
          BindAddress is the TCP address that the controller should bind to for serving prometheus metrics. It can be set to "0" to disable the metrics serving.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ProjectConfig.webhook
<sup><sup>[↩ Parent](#projectconfig)</sup></sup>



Webhook contains the controllers webhook configuration

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
        <td><b>certDir</b></td>
        <td>string</td>
        <td>
          CertDir is the directory that contains the server key and certificate. if not set, webhook server would look up the server key and certificate in {TempDir}/k8s-webhook-server/serving-certs. The server key and certificate must be named tls.key and tls.crt, respectively.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>host</b></td>
        <td>string</td>
        <td>
          Host is the hostname that the webhook server binds to. It is used to set webhook.Server.Host.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port is the port that the webhook server serves at. It is used to set webhook.Server.Port.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>