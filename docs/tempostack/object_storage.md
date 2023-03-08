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

Tempo Operator supports [Minio](https://min.io/) for TempoStack object storage.

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

* Create an instance of TempoStack by referencing the secret name:

  ```yaml
  spec:
    storage:
      secret:
        name: tempostack-dev-minio
  ```