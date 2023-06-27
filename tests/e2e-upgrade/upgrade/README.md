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
