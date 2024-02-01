# Upgrade Tests

## Prerequisites
* OLM must be installed
* tempo-operator must not be installed
* catalog image of the current sources must be available at `localregistry:5000/tempo-operator-catalog:v100.0.0`

## Test Steps
* setup old and new catalog
* install operator in `kuttl-operator-upgrade` namespace
* install Tempo in random `kuttl-*` namespace
* generate and verify traces
* switch catalog to new catalog
* assert operator got upgraded
* verify traces are still there

## Running the upgrade test with minikube
```
# specify a container registry with push permissions
export IMG_PREFIX=docker.io/${USER}

minikube start
make olm-install

export OPERATOR_VERSION=100.0.0
export LATEST_TAG=$(git describe --tags --abbrev=0)
export BUNDLE_IMGS=ghcr.io/grafana/tempo-operator/tempo-operator-bundle:${LATEST_TAG},${IMG_PREFIX}/tempo-operator-bundle:v${OPERATOR_VERSION}
make bundle docker-build docker-push bundle-build bundle-push catalog-build catalog-push

sed -i "s@localregistry:5000@${IMG_PREFIX}@g" tests/e2e-upgrade/upgrade/10-setup-olm.yaml
kubectl-kuttl test --config kuttl-test-upgrade.yaml --skip-delete
```
