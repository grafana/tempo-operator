# Upgrade Tests

## Prerequisites
* OLM must be installed
* tempo-operator must not be installed

## Test Steps
* setup old and new catalog
* install operator in `kuttl-operator-upgrade` namespace
* install Tempo in random `kuttl-*` namespace
* generate and verify traces
* switch catalog to new catalog
* assert operator got upgraded
* verify traces are still there
