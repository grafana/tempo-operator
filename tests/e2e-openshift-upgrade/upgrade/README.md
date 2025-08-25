# OpenShift Upgrade Tests

## Prerequisites
* OpenShift cluster version > 4.12
* OLM must be installed
* tempo-operator must not be installed
* File-based catalog (FBC) image for upgrade must be available.
* The following values must be provided when running the test:
  - `upgrade_fbc_image`: FBC image containing the operator version to upgrade to
  - `upgrade_operator_version`: Target operator version for upgrade
  - `upgrade_tempo_version`: Target Tempo version after upgrade
  - `upgrade_operator_csv_name`: CSV name of the target operator version

## Test Steps
* Install operators from marketplace
* Set up Minio storage required for TempoStack
* Install multitenant TempoStack with RBAC
* Install multitenant TempoMonolithic with RBAC
* Install OpenTelemetry Collector for both TempoStack and TempoMonolithic
* Create service accounts with namespace-level access
* Generate and verify traces from different namespaces with RBAC
* Verify admin access to traces from all projects
* Create upgrade catalog for Tempo Operator
* Upgrade Tempo Operator using the provided FBC image
* Assert both TempoStack and TempoMonolithic are ready after upgrade
* Verify traces and RBAC functionality still work after upgrade

## Running the OpenShift upgrade test

### Using heredoc to pass values:
```bash
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-upgrade --values - <<EOF
upgrade_fbc_image: brew.registry.redhat.io/rh-osbs/iib:988093
upgrade_operator_version: 0.16.0
upgrade_tempo_version: 2.7.2
upgrade_operator_csv_name: tempo-operator.v0.16.0-1
EOF
```

### Using values file:
Create a `values.yaml` file with the required values:
```yaml
upgrade_fbc_image: brew.registry.redhat.io/rh-osbs/iib:988093
upgrade_operator_version: 0.16.0
upgrade_tempo_version: 2.7.2
upgrade_operator_csv_name: tempo-operator.v0.16.0-1
```

Then run:
```bash
chainsaw test --config .chainsaw-openshift.yaml tests/e2e-openshift-upgrade --values values.yaml
```

## Test Features Validated
* Multitenant TempoStack and TempoMonolithic deployment
* RBAC functionality with namespace-level access control
* Trace generation and verification across multiple namespaces
* OpenTelemetry Collector integration
* Operator upgrade process using file-based catalog
* Post-upgrade functionality verification
* Admin-level access to traces from all projects 
