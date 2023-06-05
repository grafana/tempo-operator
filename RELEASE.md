# Release instructions

Steps to release a new version of the Tempo Operator:

1. Change the images tags to corresponding versions on `config/manager/controller_manager_config.yaml` the operator is usually aligned with the tempo versions. 
1. Run `make bundle USER=os-observability OPERATOR_VERSION=x.y.z`, where `x.y.z` is the version that will be released
1. Build, deploy and, run OpenShift tests locally against an OpenShift cluster `make e2e-openshift`. A locally installed [CRC](https://github.com/crc-org/crc) cluster can be used for testing.

   Note: The e2e tests require MinIO (`make deploy-minio`) and opentelemetry-operator to be installed in the cluster.
```
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift OPERATOR_NAMESPACE=openshift-operators make bundle docker-build docker-push bundle-build bundle-push olm-deploy
make e2e e2e-openshift
```
1. Add the changes to the changelog, see Generating the changelog section.
1. Send a PR with the changes
1. Once the changes above are merged and available in the `main` branch, tag it with the desired version, prefixed with `v`: `vx.y.z` (e.g. `git tag v0.1.0 && git push origin v0.1.0`)
1. The GitHub Workflow will take it from here, creating a GitHub release with the generated artifacts (manifests) and publishing the images
1. After the release, generate a new OLM bundle (`make bundle`) and create two PRs against the `Community Operators repositories`:
   1. one for the `community-operators-prod`, used by OLM on Kubernetes. Example: [`operator-framework/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/494)
   1. one for the `community-operators` directory, used by Operatorhub.io. Example: [`operator-framework/community-operators`](https://github.com/k8s-operatorhub/community-operators/pull/461)

## Generating the changelog

We use the chloggen to generate the changelog, simply run the following to generate the Changelog:

```bash
OPERATOR_VERSION=x.y.z make chlog-update
```

This will delete all entries (other than the template) in the `.chloggen` directory and create a populated Changelog.md entry.

## Test OLM Upgrade
Install latest released version:
```
IMG_PREFIX=ghcr.io/grafana/tempo-operator OPERATOR_VERSION=old_xyz OPERATOR_NAMESPACE=openshift-operators make olm-deploy
```

Build and push operator and bundle image to a container registry:
```
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift make bundle docker-build docker-push bundle-build bundle-push
```

Upgrade to new version:
```
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z OPERATOR_NAMESPACE=openshift-operators make olm-upgrade
```
