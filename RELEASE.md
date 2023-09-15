# Release instructions

Steps to release a new version of the Tempo Operator:

1. Checkout the main branch, and make sure you have the latest changes.
1. Build, deploy and, run OpenShift tests locally against an OpenShift cluster `make e2e-openshift`. A locally installed [CRC](https://github.com/crc-org/crc) cluster can be used for testing.
1. Go to GitHub Actions Tab, In the left sidebar, choose "Prepare Release" Workflow, then push in the "Run workflow" button , select the main branch and type the version of operator to release
1. Push "Run workflow", this will trigger the process to generate the CHANGELOG and publish the images, and create the GitHub release

   Note: The e2e tests require MinIO (`make deploy-minio`) and opentelemetry-operator to be installed in the cluster.
```
IMG_PREFIX=docker.io/your_username OPERATOR_VERSION=x.y.z BUNDLE_VARIANT=openshift OPERATOR_NAMESPACE=openshift-operators make bundle docker-build docker-push bundle-build bundle-push olm-deploy
make e2e e2e-openshift
```
