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

func TestGetStorageParamsForTempoStack_S3TokenModeTransport(t *testing.T) {
	tests := []struct {
		name           string
		credentialMode v1alpha1.CredentialMode
		region         string
		endpoint       string
		tlsEnabled     bool
		wantInsecure   bool
		wantEndpoint   string
	}{
		{
			name:           "token mode without TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeToken,
			region:         "us-east-1",
			tlsEnabled:     false,
			wantInsecure:   false,
			wantEndpoint:   "s3.us-east-1.amazonaws.com",
		},
		{
			name:           "token mode with TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeToken,
			region:         "us-east-1",
			tlsEnabled:     true,
			wantInsecure:   false,
			wantEndpoint:   "s3.us-east-1.amazonaws.com",
		},
		{
			name:           "token-cco mode without TLS enabled uses HTTPS",
			credentialMode: v1alpha1.CredentialModeTokenCCO,
			region:         "us-east-1",
			tlsEnabled:     false,
			wantInsecure:   false,
			wantEndpoint:   "s3.us-east-1.amazonaws.com",
		},
		{
			// spec.storage.tls.enabled must not influence the transport of token modes.
			name:           "token mode with an http endpoint uses HTTP",
			credentialMode: v1alpha1.CredentialModeToken,
			region:         "us-iso-east-1",
			endpoint:       "http://s3.us-iso-east-1.c2s.ic.gov",
			tlsEnabled:     true,
			wantInsecure:   true,
			wantEndpoint:   "s3.us-iso-east-1.c2s.ic.gov",
		},
		{
			name:           "token-cco mode with an https endpoint uses HTTPS",
			credentialMode: v1alpha1.CredentialModeTokenCCO,
			region:         "us-iso-east-1",
			endpoint:       "https://s3.us-iso-east-1.c2s.ic.gov",
			tlsEnabled:     false,
			wantInsecure:   false,
			wantEndpoint:   "s3.us-iso-east-1.c2s.ic.gov",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string][]byte{
				"bucket":   []byte("my-bucket"),
				"region":   []byte(tt.region),
				"role_arn": []byte("arn:aws:iam::123456789012:role/my-role"),
			}
			if tt.endpoint != "" {
				data["endpoint"] = []byte(tt.endpoint)
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-secret",
					Namespace: "default",
				},
				Data: data,
			}

			s := runtime.NewScheme()
			err := scheme.AddToScheme(s)
			require.NoError(t, err)

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
			require.Equal(t, tt.wantEndpoint, params.S3.Endpoint)
		})
	}
}

func TestDiscoverS3CredentialType(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string][]byte
		wantMode v1alpha1.CredentialMode
		wantErr  bool
	}{
		{
			name: "long lived credentials",
			data: map[string][]byte{
				"bucket":            []byte("my-bucket"),
				"endpoint":          []byte("https://minio:9000"),
				"access_key_id":     []byte("abc"),
				"access_key_secret": []byte("def"),
			},
			wantMode: v1alpha1.CredentialModeStatic,
		},
		{
			name: "short lived credentials",
			data: map[string][]byte{
				"bucket":   []byte("my-bucket"),
				"region":   []byte("us-east-1"),
				"role_arn": []byte("arn:aws:iam::123456789012:role/my-role"),
			},
			wantMode: v1alpha1.CredentialModeToken,
		},
		{
			// The endpoint is optional for short lived credentials and therefore must not
			// make the secret look like it holds long lived credentials.
			name: "short lived credentials with a custom endpoint",
			data: map[string][]byte{
				"bucket":   []byte("my-bucket"),
				"region":   []byte("us-iso-east-1"),
				"role_arn": []byte("arn:aws-iso:iam::123456789012:role/my-role"),
				"endpoint": []byte("http://s3.us-iso-east-1.c2s.ic.gov"),
			},
			wantMode: v1alpha1.CredentialModeToken,
		},
		{
			name: "both credential types",
			data: map[string][]byte{
				"bucket":            []byte("my-bucket"),
				"region":            []byte("us-east-1"),
				"access_key_id":     []byte("abc"),
				"access_key_secret": []byte("def"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, errs := discoverS3CredentialType(corev1.Secret{Data: tt.data}, nil)

			if tt.wantErr {
				require.Len(t, errs, 1)
				return
			}
			require.Empty(t, errs)
			require.Equal(t, tt.wantMode, mode)
		})
	}
}

func TestGetStorageParamsForTempoStack_S3StaticModeRespectsStorageTLS(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket":            []byte("my-bucket"),
			"endpoint":          []byte("https://minio:9000"),
			"access_key_id":     []byte("key"),
			"access_key_secret": []byte("secret"),
		},
	}

	tests := []struct {
		name         string
		tlsEnabled   bool
		wantInsecure bool
	}{
		{
			name:         "static mode with TLS enabled uses HTTPS",
			tlsEnabled:   true,
			wantInsecure: false,
		},
		{
			name:         "static mode without TLS enabled uses HTTP",
			tlsEnabled:   false,
			wantInsecure: true,
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
