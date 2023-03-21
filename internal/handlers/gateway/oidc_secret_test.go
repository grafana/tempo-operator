package gateway

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/status"
)

func TestGetTenantSecrets(t *testing.T) {
	tt := []struct {
		name        string
		clientID    *string
		tempo       *v1alpha1.TempoStack
		expected    []*manifestutils.GatewayTenantSecret
		expectedErr error
	}{
		{
			name: "missing secret",
			tempo: &v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "ups",
								OIDC: &v1alpha1.OIDCSpec{
									Secret: &v1alpha1.TenantSecretSpec{
										Name: "does-not-exist",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: &status.DegradedError{
				Message: fmt.Sprintf("Missing secrets for tenant %s", "ups"),
				Reason:  v1alpha1.ReasonMissingGatewayTenantSecret,
				Requeue: true,
			},
		},
		{
			name:     "invalid secret content",
			clientID: func(s string) *string { return &s }(""),
			tempo: &v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "ups",
								OIDC: &v1alpha1.OIDCSpec{
									Secret: &v1alpha1.TenantSecretSpec{
										Name: "does-not-exist",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: &status.DegradedError{
				Message: fmt.Sprintf("Missing secrets for tenant %s", "ups"),
				Reason:  v1alpha1.ReasonMissingGatewayTenantSecret,
				Requeue: true,
			},
		},
		{
			name:     "works as expected",
			clientID: func(s string) *string { return &s }("7b3834c6-9d3b-4db9-ac6b-ccefda2a1db3"),
			tempo: &v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "ups",
								OIDC: &v1alpha1.OIDCSpec{
									Secret: &v1alpha1.TenantSecretSpec{
										Name: "exist",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: nil,
			expected: []*manifestutils.GatewayTenantSecret{
				{
					TenantName:   "ups",
					ClientID:     "7b3834c6-9d3b-4db9-ac6b-ccefda2a1db3",
					ClientSecret: "super-secret",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tc.tempo.Namespace = strings.Replace(tc.name, " ", "-", -1)
			_ = createNamespace(t, tc.tempo.Namespace)

			if tc.clientID != nil {
				nsn := types.NamespacedName{Name: "exist", Namespace: tc.tempo.Namespace}
				_ = createTenantSecret(t, nsn, *tc.clientID)
			}
			got, err := GetTenantSecrets(context.Background(), k8sClient, tc.tempo)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func createTenantSecret(t *testing.T, nsn types.NamespacedName, clientID string) *corev1.Secret {
	tenantSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		StringData: map[string]string{
			"clientID":     clientID,
			"clientSecret": "super-secret",
		},
	}
	err := k8sClient.Create(context.Background(), tenantSecret)
	require.NoError(t, err)
	return tenantSecret
}

func createNamespace(t *testing.T, name string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := k8sClient.Create(context.Background(), ns)
	require.NoError(t, err)
	return ns
}

func Test_extractSecret(t *testing.T) {
	_, err := extractSecret(&corev1.Secret{}, "test")
	assert.Error(t, err)
}
