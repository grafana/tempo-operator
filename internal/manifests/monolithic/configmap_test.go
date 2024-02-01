package monolithic

import (
	"crypto/sha256"
	"fmt"
	"testing"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	tests := []struct {
		name     string
		spec     v1alpha1.TempoMonolithicSpec
		expected string
	}{
		{
			name: "memory storage",
			spec: v1alpha1.TempoMonolithicSpec{},
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
        grpc: {}
        http: {}
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "PV storage with OTLP/gRPC and OTLP/HTTP",
			spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "pv",
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
        grpc: {}
        http: {}
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "OTLP/gRPC with mTLS",
			spec: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled:    true,
								CA:         "ca",
								Cert:       "cert",
								MinVersion: string(openshiftconfigv1.VersionTLS13),
							},
						},
						HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
							Enabled: false,
						},
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
          tls:
            client_ca_file: /var/run/ca-receiver/service-ca.crt
            cert_file: /var/run/tls/receiver/tls.crt
            key_file: /var/run/tls/receiver/tls.key
            min_version: VersionTLS13
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "extra config",
			spec: v1alpha1.TempoMonolithicSpec{
				ExtraConfig: &v1alpha1.ExtraConfigSpec{
					Tempo: apiextensionsv1.JSON{Raw: []byte(`{"storage": {"trace": {"wal": {"overlay_setting": "abc"}}}}`)},
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
      overlay_setting: abc
    local:
      path: /var/tempo/blocks
distributor:
  receivers:
    otlp:
      protocols:
        grpc: {}
        http: {}
usage_report:
  reporting_enabled: false
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
					Spec: test.spec,
				},
			}
			opts.Tempo.Default()

			cfg, err := buildTempoConfig(opts)
			require.NoError(t, err)
			require.YAMLEq(t, test.expected, string(cfg))
		})
	}
}
