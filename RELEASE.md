# Release instructions

## Prerequisites

The release is driven from your machine with the scripts in `hack/release/`, using your own
GitHub credentials. You need:

* An authenticated [`gh`](https://cli.github.com/) (`gh auth login`), plus `go` and `make`.
* Write access to `grafana/tempo-operator`: the release branch is pushed to the repository itself.
  Commit signing has to be configured, it is required on every branch.
* Your GitHub user listed in `operators/tempo-operator/ci.yaml` of
  [k8s-operatorhub/community-operators](https://github.com/k8s-operatorhub/community-operators)
  and [redhat-openshift-ecosystem/community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod),
  otherwise the OperatorHub pull requests will not be merged automatically.

## Steps

Steps to release a new version of the Tempo Operator:

1. Checkout the main branch, and make sure you have the latest changes.
1. Confirm that `MIN_KUBERNETES_VERSION` and [`MIN_OPENSHIFT_VERSION`](https://access.redhat.com/support/policy/updates/openshift#dates) in the `Makefile` are still up-to-date, and create a PR if required.
1. Run `./hack/release/prepare-release.sh x.y.z`. This bumps `OPERATOR_VERSION`, regenerates the
   bundles and the CHANGELOG, and creates a PR with the title "[chore] Prepare release vx.y.z".
   Pass `--dry-run` to review the changes before anything is pushed.
1. Use that branch to build, deploy and, run OpenShift tests against an OpenShift cluster (see below for instructions).
1. Once the PR above is merged and available in the `main` branch, it will trigger the release workflow which will create the tag, the GitHub release, and push the images, bundle and catalog to ghcr.io.
   The workflow only runs for commits whose message contains "[chore] Prepare release v", so that
   no other change reaching `CHANGELOG.md` can cut a release, and it is a no-op if the version
   has already been released.
1. Once the release workflow finished, run `./hack/release/operatorhub-release.sh`. This takes the
   bundles from the `vx.y.z` tag and opens a PR against `k8s-operatorhub/community-operators`
   (community bundle) and `redhat-openshift-ecosystem/community-operators-prod` (openshift
   bundle). The version is read from the CHANGELOG, pass it explicitly to publish an older one.

## Running e2e tests on OpenShift
A locally installed [CRC](https://github.com/crc-org/crc) cluster can be used for testing.

Note: The e2e tests require [opentelemetry-operator](https://github.com/open-telemetry/opentelemetry-operator) and cluster monitoring operator (`crc config set enable-cluster-monitoring true`).

```bash
kubectl create namespace openshift-tempo-operator
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift OPERATOR_NAMESPACE=openshift-tempo-operator make build docker-build docker-push bundle bundle-build bundle-push olm-deploy
make e2e e2e-openshift
```

## Release Schedule
We plan to release the operator monthly, **at the end of each month**.

| Version | Release Manager                                          |
|---------| -------------------------------------------------------- |
|  0.19.0 | [Ruben Vargas](https://github.com/rubenvp8510)           |
|  0.20.0 | [Pavol Loffay](https://github.com/pavolloffay)           |
|  0.21.0 | [Benedikt Bongartz](https://github.com/frzifus)          |
|  0.22.0 | [Andreas Gerstmayr](https://github.com/andreasgerstmayr) |
