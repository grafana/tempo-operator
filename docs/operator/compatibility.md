---
title: "Compatibility"
description: "The Tempo Operator supports a number of Kubernetes and Tempo releases."
lead: ""
date: 2022-06-21T08:48:45+00:00
lastmod: 2022-06-21T08:48:45+00:00
draft: false
images: []
menu:
  docs:
    parent: "operator"
weight: 100
toc: true
---

The Tempo Operator supports a number of Kubernetes and Tempo releases.

## Kubernetes

The Tempo Operator uses client-go to communicate with Kubernetes clusters. The supported Kubernetes cluster version is determined by client-go. The compatibility matrix for client-go and Kubernetes clusters can be found [here](https://github.com/kubernetes/client-go#compatibility-matrix). All additional compatibility is only best effort, or happens to still/already be supported. The currently used client-go version is "v0.25.0".

This operator was tested on Kubernetes >= v1.21.0, so at least that version is required.


## Tempo

The versions of Tempo compatible to be run with the Tempo Operator are:

* v2.0.1
