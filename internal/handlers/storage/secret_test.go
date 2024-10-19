package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

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

	s3, errs := getS3Params(storageSecret, nil)
	require.Len(t, errs, 0)
	require.Equal(t, "minio:9000", s3.LongLived.Endpoint)
	require.True(t, s3.Insecure)
	require.Equal(t, "testbucket", s3.LongLived.Bucket)
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

	s3, errs := getS3Params(storageSecret, nil)
	require.Len(t, errs, 0)
	require.Equal(t, "minio:9000", s3.LongLived.Endpoint)
	require.False(t, s3.Insecure)
	require.Equal(t, "testbucket", s3.LongLived.Bucket)
}

func TestGetS3Params_short_lived(t *testing.T) {
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"bucket":   []byte("testbucket"),
			"role_arn": []byte("abc"),
			"region":   []byte("rrrr"),
		},
	}

	s3, errs := getS3Params(storageSecret, nil)
	require.Len(t, errs, 0)
	require.Equal(t, &manifestutils.S3ShortLived{
		Bucket:  "testbucket",
		RoleARN: "abc",
		Region:  "rrrr",
	}, s3.ShortLived)
}
