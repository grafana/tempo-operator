---
title: "Object Storage"
description: "Setup for storing traces to Object Storage"
lead: ""
date: 2022-06-21T08:48:45+00:00
lastmod: 2022-06-21T08:48:45+00:00
draft: false
images: []
menu:
  docs:
    parent: "tempostack"
weight: 100
toc: true
---

Tempo Operator supports [AWS S3](https://aws.amazon.com/), [Azure](https://azure.microsoft.com), [Minio](https://min.io/) and [OpenShift Data Foundation](https://www.redhat.com/en/technologies/cloud-computing/openshift-data-foundation) for TempoStack object storage.

## AWS S3

### Requirements

* Create a [bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html) on AWS.

### Installation

* Deploy the Tempo Operator to your cluster.

* Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-s3 \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="<AWS_BUCKET_ENDPOINT>" \
      --from-literal=access_key_id="<AWS_ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<AWS_ACCESS_KEY_SECRET>"
    ```

    where `tempostack-dev-s3` is the secret name.

* Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-s3
        type: s3
  ```

## Minio

### Requirements

* Deploy Minio on your Cluster, e.g. using the [Minio Operator](https://operator.min.io/)

* Create a [bucket](https://docs.min.io/docs/minio-client-complete-guide.html) on Minio via CLI.

### Installation

* Deploy the Tempo Operator to your cluster.

* Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-minio \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="<MINIO_BUCKET_ENDPOINT>" \
      --from-literal=access_key_id="<MINIO_ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<MINIO_ACCESS_KEY_SECRET>"
    ```

    where `tempostack-dev-minio` is the secret name.

* Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-minio
        type: s3
  ```

## OpenShift Data Foundation

### Requirements

* Deploy the [OpenShift Data Foundation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/4.10) on your cluster.

* Create a bucket via an ObjectBucketClaim.


### Installation

* Deploy the Tempo Operator to your cluster.

* Create an Object Storage secret with keys as follows:

    ```console
    kubectl create secret generic tempostack-dev-odf \
      --from-literal=bucket="<BUCKET_NAME>" \
      --from-literal=endpoint="https://s3.openshift-storage.svc" \
      --from-literal=access_key_id="<ACCESS_KEY_ID>" \
      --from-literal=access_key_secret="<ACCESS_KEY_SECRET>"
    ```

    where `tempostack-dev-odf` is the secret name. You can copy the values for `BUCKET_NAME`, `ACCESS_KEY_ID` and `ACCESS_KEY_SECRET` from your ObjectBucketClaim's accompanied secret.

* Create an instance of TempoStack by referencing the secret name and type as `s3`:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-odf
        type: s3
  ```