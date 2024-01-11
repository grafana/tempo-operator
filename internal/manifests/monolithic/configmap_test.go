package monolithic

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func TestBuildConfigMap(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	cm, checksum, err := BuildConfigMap(opts)
	require.NoError(t, err)
	require.NotNil(t, cm.Data)
	require.NotNil(t, cm.Data["tempo.yaml"])
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(cm.Data["tempo.yaml"]))), checksum)
}

func TestBuildConfig(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{},
		},
	}

	tests := []struct {
		name      string
		storage   *v1alpha1.MonolithicStorageSpec
		ingestion *v1alpha1.MonolithicIngestionSpec
		expected  string
	}{
		{
			name: "memory storage",
			storage: &v1alpha1.MonolithicStorageSpec{
				Traces: v1alpha1.MonolithicTracesStorageSpec{
					Backend: "memory",
				},
			},
			expected: `
server:
  http_listen_port: 3200
storage:
  trace:
    backend: local
    wal:
      path: /var/tempo/wal
    local:
      path: /var/tempo/blocks
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "PV storage with OTLP/gRPC and OTLP/HTTP",
			storage: &v1alpha1.MonolithicStorageSpec{
				Traces: v1alpha1.MonolithicTracesStorageSpec{
					Backend: "pv",
				},
			},
			ingestion: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
						Enabled: true,
					},
					HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
						Enabled: true,
					},
				},
			},
			expected: `
server:
  http_listen_port: 3200
storage:
  trace:
    backend: local
    wal:
      path: /var/tempo/wal
    local:
      path: /var/tempo/blocks
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
        http:
usage_report:
  reporting_enabled: false
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec.Storage = test.storage
			opts.Tempo.Spec.Ingestion = test.ingestion
			cfg, err := buildTempoConfig(opts)
			require.NoError(t, err)
			require.YAMLEq(t, test.expected, string(cfg))
		})
	}
}
