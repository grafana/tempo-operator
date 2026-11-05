package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestGetS3ParamsInsecure(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"endpoint":          []byte("http://minio:9000"),
			"bucket":            []byte("testbucket"),
			"access_key_id":     []byte("abc"),
			"access_key_secret": []byte("def"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeStatic)

	require.Len(t, errs, 0)
	require.Equal(t, "minio:9000", s3.Endpoint)
	require.True(t, s3.Insecure)
	require.Equal(t, "testbucket", s3.Bucket)
}

func TestGetS3ParamsSecure(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"endpoint":          []byte("https://minio:9000"),
			"bucket":            []byte("testbucket"),
			"access_key_id":     []byte("abc"),
			"access_key_secret": []byte("def"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeStatic)

	require.Len(t, errs, 0)
	require.Equal(t, "minio:9000", s3.Endpoint)
	require.False(t, s3.Insecure)
	require.Equal(t, "testbucket", s3.Bucket)
}

func TestGetS3Params_short_lived(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"role_arn": []byte("abc"),
			"region":   []byte("us-east-2"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeToken)

	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.S3{
		Bucket:    "testbucket",
		RoleARN:   "abc",
		Region:    "us-east-2",
		Endpoint:  "s3.us-east-2.amazonaws.com",
		Partition: manifestutils.AWSPartitionForRegion("us-east-2"),
		Audience:  "sts.amazonaws.com",
	}, s3)
}

func TestGetS3Params_short_lived_govcloud(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"role_arn": []byte("arn:aws-us-gov:iam::12345:role/test"),
			"region":   []byte("us-gov-west-1"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeToken)

	require.Len(t, errs, 0)
	// GovCloud is served under amazonaws.com, therefore nothing must be overridden.
	require.Equal(t, "s3.us-gov-west-1.amazonaws.com", s3.Endpoint)
	require.Equal(t, "aws-us-gov", s3.Partition.ID)
	require.Equal(t, "sts.amazonaws.com", s3.Audience)
	require.False(t, s3.Insecure)
}

func TestGetS3Params_short_lived_iso(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"role_arn": []byte("arn:aws-iso:iam::12345:role/test"),
			"region":   []byte("us-iso-east-1"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeToken)

	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.S3{
		Bucket:    "testbucket",
		RoleARN:   "arn:aws-iso:iam::12345:role/test",
		Region:    "us-iso-east-1",
		Endpoint:  "s3.us-iso-east-1.c2s.ic.gov",
		Partition: manifestutils.AWSPartitionForRegion("us-iso-east-1"),
		Audience:  "sts.amazonaws.com",
	}, s3)
}

func TestGetS3Params_short_lived_isob(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket": []byte("testbucket"),
			"region": []byte("us-isob-east-1"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeTokenCCO)

	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.S3{
		Bucket:    "testbucket",
		Region:    "us-isob-east-1",
		Endpoint:  "s3.us-isob-east-1.sc2s.sgov.gov",
		Partition: manifestutils.AWSPartitionForRegion("us-isob-east-1"),
		Audience:  "sts.amazonaws.com",
	}, s3)
}

func TestGetS3Params_short_lived_custom_endpoint(t *testing.T) {
	// The endpoint field of the storage secret is the only way to request plain HTTP for
	// short lived credentials.
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"region":   []byte("us-iso-east-1"),
			"endpoint": []byte("http://s3.us-iso-east-1.c2s.ic.gov"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeTokenCCO)

	require.Len(t, errs, 0)
	require.Equal(t, "s3.us-iso-east-1.c2s.ic.gov", s3.Endpoint)
	require.True(t, s3.Insecure)
	require.Equal(t, "aws-iso", s3.Partition.ID)
}

func TestGetS3Params_short_lived_custom_endpoint_secure(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"region":   []byte("us-east-2"),
			"role_arn": []byte("abc"),
			"endpoint": []byte("https://bucket.vpce-1234.s3.us-east-2.vpce.amazonaws.com"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeToken)

	require.Len(t, errs, 0)
	require.Equal(t, "bucket.vpce-1234.s3.us-east-2.vpce.amazonaws.com", s3.Endpoint)
	require.False(t, s3.Insecure)
}

func TestGetS3Params_short_lived_invalid_endpoint(t *testing.T) {
	// The scheme of the endpoint selects the transport, therefore it must be explicit:
	// a missing or unknown scheme must not silently fall back to plain HTTP.
	tests := []struct {
		name     string
		endpoint string
		wantErr  string
	}{
		{name: "no scheme", endpoint: "s3.us-iso-east-1.c2s.ic.gov", wantErr: "http:// or https:// scheme"},
		{name: "host and port without scheme", endpoint: "s3.us-iso-east-1.c2s.ic.gov:443", wantErr: "http:// or https:// scheme"},
		{name: "unsupported scheme", endpoint: "s3://s3.us-iso-east-1.c2s.ic.gov", wantErr: "http:// or https:// scheme"},
		{name: "missing host", endpoint: "https://", wantErr: "http:// or https:// scheme"},
		{name: "path", endpoint: "https://s3.us-iso-east-1.c2s.ic.gov/bucket", wantErr: "only contain a scheme, a host and an optional port"},
		{name: "query", endpoint: "https://s3.us-iso-east-1.c2s.ic.gov?a=b", wantErr: "only contain a scheme, a host and an optional port"},
		{name: "credentials", endpoint: "https://user:pass@s3.us-iso-east-1.c2s.ic.gov", wantErr: "only contain a scheme, a host and an optional port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageSecret := corev1.Secret{
				Data: map[string][]byte{
					"bucket":   []byte("testbucket"),
					"region":   []byte("us-iso-east-1"),
					"endpoint": []byte(tt.endpoint),
				},
			}

			_, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeTokenCCO)

			require.Len(t, errs, 1)
			require.Contains(t, errs[0].Detail, tt.wantErr)
		})
	}
}

func TestGetS3Params_short_lived_endpoint_scheme(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		wantEndpoint string
		wantInsecure bool
	}{
		{name: "http", endpoint: "http://s3.us-iso-east-1.c2s.ic.gov", wantEndpoint: "s3.us-iso-east-1.c2s.ic.gov", wantInsecure: true},
		{name: "https", endpoint: "https://s3.us-iso-east-1.c2s.ic.gov", wantEndpoint: "s3.us-iso-east-1.c2s.ic.gov", wantInsecure: false},
		{name: "uppercase scheme", endpoint: "HTTPS://s3.us-iso-east-1.c2s.ic.gov", wantEndpoint: "s3.us-iso-east-1.c2s.ic.gov", wantInsecure: false},
		{name: "port and trailing slash", endpoint: "http://s3.us-iso-east-1.c2s.ic.gov:8080/", wantEndpoint: "s3.us-iso-east-1.c2s.ic.gov:8080", wantInsecure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageSecret := corev1.Secret{
				Data: map[string][]byte{
					"bucket":   []byte("testbucket"),
					"region":   []byte("us-iso-east-1"),
					"endpoint": []byte(tt.endpoint),
				},
			}

			s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeTokenCCO)

			require.Len(t, errs, 0)
			require.Equal(t, tt.wantEndpoint, s3.Endpoint)
			require.Equal(t, tt.wantInsecure, s3.Insecure)
		})
	}
}

func TestGetS3Params_short_lived_audience_override(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"region":   []byte("us-iso-east-1"),
			"audience": []byte("openshift"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil, v1alpha1.CredentialModeTokenCCO)

	require.Len(t, errs, 0)
	require.Equal(t, "openshift", s3.Audience)
}

func TestGetGCSParams_short_lived(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucketname":        []byte("testbucket"),
			"iam_sa":            []byte("abc"),
			"iam_sa_project_id": []byte("rrrr"),
			"key.json":          []byte("{\"type\": \"external_account\", \"credential_source\": {\"file\": \"/var/run/secrets/storage/serviceaccount/token\"}}"),
		},
	}
	gcs, errs := getGCSParams(storageSecret, nil, v1alpha1.CredentialModeToken)

	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.GCS{
		Bucket:            "testbucket",
		IAMServiceAccount: "abc",
		ProjectID:         "rrrr",
		Audience:          "openshift",
	}, gcs)
}

func TestGetGCSParams_long_lived(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucketname": []byte("testbucket"),
			"key.json":   []byte("creds"),
		},
	}

	gcs, errs := getGCSParams(storageSecret, nil, v1alpha1.CredentialModeStatic)

	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.GCS{
		Bucket: "testbucket",
	}, gcs)
}

func TestGetGCSParams_both_tokens(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucketname":        []byte("testbucket"),
			"key.json":          []byte("creds"),
			"iam_sa":            []byte("abc"),
			"iam_sa_project_id": []byte("rrrr"),
		},
	}

	_, errs := discoverGCSCredentialType(storageSecret, nil)
	require.Len(t, errs, 1)
}
