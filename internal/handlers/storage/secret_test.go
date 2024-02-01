package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

	s3, errs := getS3Params(storageSecret, nil)
	require.Len(t, errs, 0)
	require.Equal(t, "minio:9000", s3.Endpoint)
	require.False(t, s3.Insecure)
	require.Equal(t, "testbucket", s3.Bucket)
}
