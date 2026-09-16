package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestGetStorageParamsForTempoStack_S3TokenModeAlwaysHTTPS(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket":   []byte("my-bucket"),
			"region":   []byte("us-east-1"),
			"role_arn": []byte("arn:aws:iam::123456789012:role/my-role"),
		},
	}

	tests := []struct {
		name           string
		credentialMode v1alpha1.CredentialMode
		tlsEnabled     bool
		wantInsecure   bool
	}{
		{
			name:           "token mode without TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeToken,
			tlsEnabled:     false,
			wantInsecure:   false,
		},
		{
			name:           "token mode with TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeToken,
			tlsEnabled:     true,
			wantInsecure:   false,
		},
		{
			name:           "token-cco mode without TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeTokenCCO,
			tlsEnabled:     false,
			wantInsecure:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			err := scheme.AddToScheme(s)
			require.NoError(t, err)

			cl := fake.NewClientBuilder().WithScheme(s).WithObjects(secret.DeepCopy()).Build()

			tempo := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.TempoStackSpec{
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name:           "storage-secret",
							Type:           v1alpha1.ObjectStorageSecretS3,
							CredentialMode: tt.credentialMode,
						},
						TLS: v1alpha1.TLSSpec{
							Enabled: tt.tlsEnabled,
						},
					},
				},
			}

			params, errs := GetStorageParamsForTempoStack(context.Background(), cl, tempo)
			require.Empty(t, errs)
			require.Equal(t, tt.wantInsecure, params.S3.Insecure)
		})
	}
}

func TestGetStorageParamsForTempoStack_S3StaticMode(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		tlsEnabled   bool
		wantInsecure bool
	}{
		{
			name:         "https endpoint without TLS enabled uses HTTPS",
			endpoint:     "https://minio:9000",
			tlsEnabled:   false,
			wantInsecure: false,
		},
		{
			name:         "https endpoint with TLS enabled uses HTTPS",
			endpoint:     "https://minio:9000",
			tlsEnabled:   true,
			wantInsecure: false,
		},
		{
			name:         "http endpoint without TLS enabled uses HTTP",
			endpoint:     "http://minio:9000",
			tlsEnabled:   false,
			wantInsecure: true,
		},
		{
			name:         "http endpoint with TLS enabled uses HTTPS",
			endpoint:     "http://minio:9000",
			tlsEnabled:   true,
			wantInsecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			err := scheme.AddToScheme(s)
			require.NoError(t, err)

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"bucket":            []byte("my-bucket"),
					"endpoint":          []byte(tt.endpoint),
					"access_key_id":     []byte("key"),
					"access_key_secret": []byte("secret"),
				},
			}

			cl := fake.NewClientBuilder().WithScheme(s).WithObjects(secret).Build()

			tempo := v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.TempoStackSpec{
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name:           "storage-secret",
							Type:           v1alpha1.ObjectStorageSecretS3,
							CredentialMode: v1alpha1.CredentialModeStatic,
						},
						TLS: v1alpha1.TLSSpec{
							Enabled: tt.tlsEnabled,
						},
					},
				},
			}

			params, errs := GetStorageParamsForTempoStack(context.Background(), cl, tempo)
			require.Empty(t, errs)
			require.Equal(t, tt.wantInsecure, params.S3.Insecure)
		})
	}
}
