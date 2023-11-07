# Release instructions

Steps to release a new version of the Tempo Operator:

1. Checkout the main branch, and make sure you have the latest changes.
1. Go to GitHub Actions Tab, In the left sidebar, choose "Prepare Release" Workflow, then push in the "Run workflow" button , select the main branch and type the version of operator to release
1. Push "Run workflow", this will trigger the process to generate the CHANGELOG and generate the bundle, this will create a PR with the title "Prepare Release vx.y.z`"
1. Once the PR is created, use that branch to build, deploy and, run OpenShift tests locally against an OpenShift cluster `make e2e-openshift`. A locally installed [CRC](https://github.com/crc-org/crc) cluster can be used for testing.
1. Once the PR above are merged and available in the `main` branch, tag it with the desired version, prefixed with `v`: `vx.y.z` (e.g. `git tag v0.1.0 && git push origin v0.1.0`)

   Note: The e2e tests require MinIO (`make deploy-minio`) and opentelemetry-operator to be installed in the cluster.
```
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift OPERATOR_NAMESPACE=openshift-operators make bundle docker-build docker-push bundle-build bundle-push olm-deploy
make e2e e2e-openshift
```
