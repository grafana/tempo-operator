package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestGetAzureParams(t *testing.T) {
	tests := []struct {
		name           string
		mode           v1alpha1.CredentialMode
		secret         corev1.Secret
		expectedError  bool
		expectedConfig *manifestutils.AzureStorage
	}{
		{
			name: "static token",
			mode: v1alpha1.CredentialModeStatic,
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
					"account_key":  []byte("key"),
				},
			},
			expectedConfig: &manifestutils.AzureStorage{
				Container: "tempo",
			},
		},
		{
			name: "short live token",
			mode: v1alpha1.CredentialModeToken,
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
					"client_id":    []byte("client"),
					"tenant_id":    []byte("tenant"),
				},
			},
			expectedConfig: &manifestutils.AzureStorage{
				Container: "tempo",
				TenantID:  "tenant",
				ClientID:  "client",
				Audience:  manifestutils.AzureDefaultAudience,
			},
		},
		{
			name: "missing short live token",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
					"tenant_id":    []byte("tenant"),
				},
			},
			mode:          v1alpha1.CredentialModeToken,
			expectedError: true,
		},
		{
			name: "missing static token",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
				},
			},
			mode:          v1alpha1.CredentialModeStatic,
			expectedError: true,
		},
		{
			name: "other audience",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
					"client_id":    []byte("client"),
					"tenant_id":    []byte("tenant"),
					"audience":     []byte("other"),
				},
			},
			mode: v1alpha1.CredentialModeToken,
			expectedConfig: &manifestutils.AzureStorage{
				Container: "tempo",
				TenantID:  "tenant",
				ClientID:  "client",
				Audience:  "other",
			},
		},
		{
			name:          "unsupported CCO",
			mode:          v1alpha1.CredentialModeTokenCCO,
			expectedError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storagePath := field.NewPath("spec", "storage")
			config, err := getAzureParams(test.secret, storagePath, test.mode)
			if test.expectedError {
				assert.Len(t, err, 1)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectedConfig, config)
			}
		})
	}
}

func TestDiscoverAzureCredentialType(t *testing.T) {
	tests := []struct {
		name          string
		expectedMode  v1alpha1.CredentialMode
		secret        corev1.Secret
		expectedError bool
	}{
		{
			name: "Secret contains Azure Credential Type",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"account_name": []byte("account"),
					"account_key":  []byte("key"),
				},
			},
			expectedMode: v1alpha1.CredentialModeStatic,
		},
		{
			name: "Secret contains Azure Credential Type",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"tenant_id":    []byte("tempo-tenant"),
					"client_id":    []byte("xxxx"),
					"account_name": []byte("account"),
				},
			},
			expectedMode: v1alpha1.CredentialModeToken,
		},
		{
			name: "Error, both fields mixed",
			secret: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("tempo"),
					"tenant_id":    []byte("tempo-tenant"),
					"client_id":    []byte("xxxx"),
					"account_name": []byte("account"),
					"account_key":  []byte("key"),
				},
			},
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storagePath := field.NewPath("spec", "storage")
			mode, err := discoverAzureCredentialType(tt.secret, storagePath)
			if tt.expectedError {
				assert.Len(t, err, 1)
			} else {
				assert.Equal(t, tt.expectedMode, mode)
			}
		})
	}
}
