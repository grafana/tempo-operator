# Release instructions

Steps to release a new version of the Tempo Operator:

1. Change the images tags to corresponding versions on `config/manager/controller_manager_config.yaml` the operator is usually aligned with the tempo versions. 

3. Run `make bundle USER=os-observability VERSION=x.y.z`, where `x.y.z` is the version that will be released.
5. Add the changes to the changelog, see Generating the changelog secction. Manually remove irrelevant changes like dependencies updates, refactorizations, etc..
7. Once the changes above are merged and available in `main`, tag it with the desired version, prefixed with `v`: `vx.y.z`
8. The GitHub Workflow will take it from here, creating a GitHub release with the generated artifacts (manifests) and publishing the images
9. After the release, generate a new OLM bundle (`make bundle`) and create two PRs against the `Community Operators repositories`:
   1. one for the `community-operators-prod`, used by OLM on Kubernetes. Example: [`operator-framework/community-operators-prod`](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/494)
   1. one for the `community-operators` directory, used by Operatorhub.io. Example: [`operator-framework/community-operators`](https://github.com/k8s-operatorhub/community-operators/pull/461)
10. Update release schedule table, by moving the current release manager to the end of the table with updated release version.

## Generating the changelog

For now we are using a manual process to update the changelog, execute the following command:

```bash
make changelog
```

This will give you the latest commits in a changelog format in STDOUT, copy it to CHANGELOG.md, remove irrelevant commits.