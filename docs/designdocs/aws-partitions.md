# AWS partitions

AWS operates several isolated partitions. They serve their services under different DNS suffixes
and identify their resources with different ARN prefixes, so a configuration that works in the
commercial partition does not work in any of the others.

| Partition | Example regions | DNS suffix | ARN prefix |
|---|---|---|---|
| `aws` (commercial) | `us-east-1`, `eu-west-1` | `amazonaws.com` | `arn:aws:` |
| `aws-us-gov` (GovCloud) | `us-gov-east-1`, `us-gov-west-1` | `amazonaws.com` | `arn:aws-us-gov:` |
| `aws-iso` (C2S) | `us-iso-east-1` | `c2s.ic.gov` | `arn:aws-iso:` |
| `aws-iso-b` (SC2S) | `us-isob-east-1` | `sc2s.sgov.gov` | `arn:aws-iso-b:` |
| `aws-iso-e` | `eu-isoe-west-1` | `cloud.adc-e.uk` | `arn:aws-iso-e:` |
| `aws-iso-f` | `us-isof-south-1` | `csp.hci.ic.gov` | `arn:aws-iso-f:` |
| `aws-eusc` (European Sovereign Cloud) | `eusc-de-east-1` | `amazonaws.eu` | `arn:aws-eusc:` |
| `aws-cn` (China) | `cn-north-1` | `amazonaws.com.cn` | `arn:aws-cn:` |

GovCloud is served under `amazonaws.com`, which is why it works with a configuration that assumes
the commercial partition. The remaining partitions do not.

The operator derives the partition from the region of the storage secret, in
[`aws_partition.go`](../../internal/manifests/manifestutils/aws_partition.go). Regions it does not
recognize fall back to the commercial partition, so that a region added by AWS in the future does
not break the reconciliation of existing stacks.

## What is derived from the partition

For the `token` and `token-cco` credential modes:

| Setting | Value |
|---|---|
| S3 endpoint (`storage.trace.s3.endpoint`) | `s3.<region>.<dns suffix>` |
| Audience of the projected service account token | `sts.<region>.<dns suffix>`, `sts.amazonaws.com` in the commercial and GovCloud partitions |
| STS endpoint of the token exchange | `https://sts.<region>.<dns suffix>`, only outside of `amazonaws.com` |
| `Resource` of the CCO `CredentialsRequest` | `arn:<partition>:s3:*:*:*`, taken from the partition of the role ARN |
| Signing region (`storage.trace.s3.region`) | only rendered when it cannot be derived from the endpoint, see below |

The `Resource` of the `CredentialsRequest` uses the partition of the role ARN rather than the one of
the region, because an IAM policy must name resources in the same partition as the role it is
attached to. The role ARN also reaches the operator earlier: it comes from the `ROLEARN` environment
variable of the operator pod, which is read before the storage secret of a given `TempoStack` is.

## Optional fields of the storage secret

The storage secret of the `token` and `token-cco` modes accepts two optional fields:

* `endpoint` overrides the auto-generated S3 endpoint, for private endpoints and for endpoints
  behind a proxy. Its scheme selects the transport: `http://` makes Tempo reach S3 over plain HTTP.
  This is required in the ISO partitions, where service control policies block HTTPS access to S3
  and transport security is handled at the network layer instead, and it is the only way to request
  it — `spec.storage.tls.enabled` configures the CA and the certificates of the connection, not
  whether TLS is used at all, and it keeps that meaning for every credential mode.
* `audience` overrides the audience of the projected service account token, for the case the IAM
  OIDC identity provider of the cluster was registered with a different one.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tempo-storage
stringData:
  bucket: my-bucket
  region: us-iso-east-1
  endpoint: "http://s3.us-iso-east-1.c2s.ic.gov"
```

## Why the signing region is rendered conditionally

Tempo does not set the signing region on the S3 client unless `storage.trace.s3.region` is
configured, and lets the minio-go client derive it from the endpoint host. That derivation
([`s3utils.GetRegionFromURL`](https://github.com/minio/minio-go/blob/master/pkg/s3utils/utils.go))
only matches hosts under `amazonaws.com` and `amazonaws.com.cn`, and returns an empty region for
anything else, which makes the client sign requests for the wrong region.

The operator therefore renders `region` whenever the endpoint is not one minio-go can parse — that
is, outside of the commercial and GovCloud partitions, and whenever the secret carries a custom
endpoint. In every other case the rendered configuration is unchanged, which matters because any
change of the configuration rolls every pod of an existing deployment.

## Why the STS endpoint is an environment variable

Tempo obtains AWS credentials through the credential chain of the minio-go client
([`fetchCreds`](https://github.com/grafana/tempo/blob/main/tempodb/backend/s3/s3.go)). The
`credentials.IAM` provider of that chain exchanges the projected service account token for
temporary credentials, and builds the endpoint of the exchange as `sts.<AWS_REGION>.amazonaws.com`
— or the global `https://sts.amazonaws.com` when `AWS_REGION` is unset — with the DNS suffix
hardcoded ([`iam_aws.go`](https://github.com/minio/minio-go/blob/master/pkg/credentials/iam_aws.go)).
The S3 configuration of Tempo has no field for it.

The only hook Tempo exposes is `TEST_IAM_ENDPOINT`, which it passes to the provider as its
`Endpoint`. The operator sets it on the Tempo containers for the partitions served under a different
DNS suffix, and sets nothing in the commercial and GovCloud partitions. Its name suggests it is
meant for tests; until Tempo grows a supported configuration field it is the only way to reach the
STS endpoint of an ISO partition.

When the exchange fails, the chain does not error out: it falls through to anonymous requests, so
the symptom is `Access Denied` on `ListObjects` rather than a credentials error.

Two related details:

* The operator sets `AWS_DEFAULT_REGION` on the Tempo pods, but the minio-go credential chain reads
  `AWS_REGION`. Setting `AWS_REGION` would not help outside of `amazonaws.com` anyway, and would
  change the behavior of existing deployments, so the explicit endpoint is used instead.
* With an endpoint outside of `amazonaws.com`, minio-go addresses buckets in path style. The ISO
  partitions accept it. Tempo exposes a `forcepathstyle` setting if this ever has to be controlled
  explicitly; the operator does not configure it.

## Verifying a deployment

There is no way to exercise the ISO partitions in CI, so the settings have to be checked on a
cluster in one of those regions:

```bash
# endpoint, insecure and region of the generated configuration
oc get cm tempo-<name> -o jsonpath='{.data.tempo\.yaml}' | grep -A 5 's3:'

# audience and role of the service account
oc get sa tempo-<name> -o jsonpath='{.metadata.annotations}'

# partition of the CredentialsRequest, for the token-cco mode
oc get credentialsrequest <name> -o jsonpath='{.spec.providerSpec.statementEntries}'

# STS endpoint of the token exchange
oc set env deploy/tempo-<name>-ingester --list | grep TEST_IAM_ENDPOINT
```
