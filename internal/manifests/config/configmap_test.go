package config

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

func TestConfigmap(t *testing.T) {
	cm, checksum, err := BuildConfigMap(manifestutils.Params{
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1alpha1.TempoStackSpec{
				Retention: v1alpha1.RetentionSpec{
					Global: v1alpha1.RetentionConfig{
						Traces: metav1.Duration{Duration: 48 * time.Hour},
					},
				},
			},
		},
		TLSProfile: tlsprofile.TLSProfileOptions{
			MinTLSVersion: string(openshiftconfigv1.VersionTLS13),
		},
		StorageParams: manifestutils.StorageParams{
			S3: &manifestutils.S3{
				Endpoint: "http://minio:9000",
				Bucket:   "tempo",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, cm.Data)
	require.NotNil(t, cm.Data["tempo.yaml"])
	require.NotNil(t, cm.Data["overrides.yaml"])
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(cm.Data["tempo.yaml"]))), checksum)
}
