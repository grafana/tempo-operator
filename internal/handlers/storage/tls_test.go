package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestGetCAConfigMapKey(t *testing.T) {
	path := field.NewPath("spec", "storage", "tls", "caName")

	tests := []struct {
		name        string
		data        map[string]string
		expectedKey string
		expectErr   bool
	}{
		{
			name:        "service-ca.crt key",
			data:        map[string]string{manifestutils.TLSCAFilename: "cert-data"},
			expectedKey: manifestutils.TLSCAFilename,
		},
		{
			name:        "ca.crt key (backwards compatibility)",
			data:        map[string]string{manifestutils.StorageTLSCAFilename: "cert-data"},
			expectedKey: manifestutils.StorageTLSCAFilename,
		},
		{
			name:        "ca-bundle.crt key (OpenShift injected CA bundle)",
			data:        map[string]string{manifestutils.OpenshiftTrustedCABundleFilename: "cert-data"},
			expectedKey: manifestutils.OpenshiftTrustedCABundleFilename,
		},
		{
			name:        "service-ca.crt takes precedence over ca-bundle.crt",
			data:        map[string]string{manifestutils.TLSCAFilename: "cert-data", manifestutils.OpenshiftTrustedCABundleFilename: "cert-data"},
			expectedKey: manifestutils.TLSCAFilename,
		},
		{
			name:      "no recognized key",
			data:      map[string]string{"other-key": "cert-data"},
			expectErr: true,
		},
		{
			name:      "empty configmap",
			data:      map[string]string{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ca"},
				Data:       tt.data,
			}
			key, errs := getCAConfigMapKey(cm, path)
			if tt.expectErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
				assert.Equal(t, tt.expectedKey, key)
			}
		})
	}
}
