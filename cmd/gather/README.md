# Tempo Operator Must-Gather

The Tempo Operator `must-gather` tool is designed to collect comprehensive information about Tempo components within an OpenShift cluster. This utility extends the functionality of [OpenShift must-gather](https://github.com/openshift/must-gather) by specifically targeting and retrieving data related to the Tempo Operator, helping in diagnostics and troubleshooting.

Note that you can use this utility too to gather information about the objects deployed by the Tempo Operator if you don't use OpenShift.

## What is a Must-Gather?

The `must-gather` tool is a utility that collects logs, cluster information, and resource configurations related to a specific operator or application in an OpenShift cluster. It helps cluster administrators and developers diagnose issues by providing a snapshot of the cluster's state related to the targeted component. More information [in the official documentation](https://docs.openshift.com/container-platform/4.16/support/gathering-cluster-data.html).

## Usage

First, you will need to build and push the image:
```sh
make container-must-gather container-must-gather-push
```

To run the must-gather tool for the Tempo Operator, use one of the following commands, depending on how you want to source the image and the namespace where the operator is deployed.

### Using the image from the Operator deployment

If you want to use the image in a running cluster, you need to run the following command:

```sh
oc adm must-gather --image=<must-gather-image> -- /usr/bin/must-gather --operator-namespace tempo-operator-system
```

### Using it as a CLI

You only need to build and run:
```sh
make must-gather
./bin/must-gather --help
```

This is the recommended way to do it if you are not using OpenShift.