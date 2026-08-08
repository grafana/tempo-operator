package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAzureShortLiveTokenAnnotation(t *testing.T) {
	annotations := AzureShortLiveTokenAnnotation(AzureStorage{
		TenantID: "test-tenant",
		ClientID: "test-client",
	})

	assert.Equal(t, "test-client", annotations["azure.workload.identity/client-id"])
	assert.Equal(t, "test-tenant", annotations["azure.workload.identity/tenant-id"])
}

func TestCommonAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		expected map[string]string
	}{
		{
			name: "config checksum only",
			params: Params{
				ConfigChecksum: "abc123",
			},
			expected: map[string]string{
				"tempo.grafana.com/config.hash": "abc123",
			},
		},
		{
			name: "config checksum and cert hash annotations",
			params: Params{
				ConfigChecksum: "abc123",
				CertHashAnnotations: map[string]string{
					"tempo.grafana.com/cert-hash-distributor": "hash1",
					"tempo.grafana.com/cert-hash-gateway":     "hash2",
				},
			},
			expected: map[string]string{
				"tempo.grafana.com/config.hash":           "abc123",
				"tempo.grafana.com/cert-hash-distributor": "hash1",
				"tempo.grafana.com/cert-hash-gateway":     "hash2",
			},
		},
		{
			name: "nil cert hash annotations",
			params: Params{
				ConfigChecksum:      "abc123",
				CertHashAnnotations: nil,
			},
			expected: map[string]string{
				"tempo.grafana.com/config.hash": "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommonAnnotations(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCertificateHashAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		certSecrets map[string]*corev1.Secret
		expected    map[string]string
	}{
		{
			name: "multiple certificate secrets",
			certSecrets: map[string]*corev1.Secret{
				"distributor": {
					Data: map[string][]byte{
						"tls.crt": []byte("cert-data-1"),
						"tls.key": []byte("key-data-1"),
					},
				},
				"gateway": {
					Data: map[string][]byte{
						"tls.crt": []byte("cert-data-2"),
						"tls.key": []byte("key-data-2"),
					},
				},
			},
			expected: map[string]string{
				"tempo.grafana.com/cert-hash-distributor": "5bfa3a8a7c8831ac5c8585c3ea537304c56ef80008dafc6b48683466750e2130", // sha256 of "cert-data-1"
				"tempo.grafana.com/cert-hash-gateway":     "ab3befe7203e6aeddd203ef9d5262ebf0c4f78633e89ecc0c8fe4b142f1912f5", // sha256 of "cert-data-2"
			},
		},
		{
			name: "nil secret",
			certSecrets: map[string]*corev1.Secret{
				"distributor": nil,
			},
			expected: map[string]string{},
		},
		{
			name: "secret without tls.crt",
			certSecrets: map[string]*corev1.Secret{
				"distributor": {
					Data: map[string][]byte{
						"tls.key": []byte("key-data-1"),
					},
				},
			},
			expected: map[string]string{},
		},
		{
			name:        "empty secrets map",
			certSecrets: map[string]*corev1.Secret{},
			expected:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CertificateHashAnnotations(tt.certSecrets)
			assert.Equal(t, tt.expected, result)
		})
	}
}
