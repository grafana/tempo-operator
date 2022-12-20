package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestConfigmap(t *testing.T) {
	cm, err := BuildConfigMap(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Retention: v1alpha1.RetentionSpec{
				Global: v1alpha1.RetentionConfig{
					Traces: metav1.Duration{Duration: 48 * time.Hour},
				},
			},
		},
	}, Params{S3: S3{
		Endpoint: "http://minio:9000",
		Bucket:   "tempo",
	}})

	require.NoError(t, err)
	require.NotNil(t, cm.Data)
	require.NotNil(t, cm.Data["tempo.yaml"])
	require.NotNil(t, cm.Data["overrides.yaml"])

}
