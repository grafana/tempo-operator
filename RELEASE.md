# Release instructions

Steps to release a new version of the Tempo Operator:

1. Checkout the main branch, and make sure you have the latest changes.
1. Go to GitHub Actions Tab, In the left sidebar, choose "Prepare Release" Workflow, then push in the "Run workflow" button , select the main branch and type the version of operator to release
1. Push "Run workflow", this will trigger the process to generate the CHANGELOG and generate the bundle, this will create a PR with the title "Prepare Release vx.y.z`"
1. Once the PR is created, use that branch to build, deploy and, run OpenShift tests against an OpenShift cluster (see below for instructions).
1. Once the PR above are merged and available in the `main` branch, it will trigger the release workflow which will create the tag and the GitHub release.

## Running e2e tests on OpenShift
A locally installed [CRC](https://github.com/crc-org/crc) cluster can be used for testing.

Note: The e2e tests require [opentelemetry-operator](https://github.com/open-telemetry/opentelemetry-operator) to be installed in the cluster.

```
kubectl create namespace tempo-operator-system
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift OPERATOR_NAMESPACE=tempo-operator-system make bundle docker-build docker-push bundle-build bundle-push olm-deploy
make e2e e2e-openshift
```

## Release Schedule
We plan to release the operator monthly, **at the end of each month**.

| Release Date | Version | Release Manager                                          |
| ------------ | ------- | -------------------------------------------------------- |
| Jan 2024     | 0.8.0   | [Andreas Gerstmayr](https://github.com/andreasgerstmayr) |
| Feb 2024     | 0.9.0   | [Benedikt Bongartz](https://github.com/frzifus)          |
| Mar 2024     | 0.10.0  | [Israel Blancas](https://github.com/iblancasa)           |
| Apr 2024     | 0.11.0  | [Ruben Vargas](https://github.com/rubenvp8510)           |
| May 2024     | 0.12.0  | [Pavol Loffay](https://github.com/pavolloffay)           |
