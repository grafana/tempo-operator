package monolithic

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
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
				JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
					Enabled:                      true,
					ServicesQueryDuration:        &metav1.Duration{Duration: time.Duration(3 * 24 * time.Hour)},
					FindTracesConcurrentRequests: 22,
				},
			},
		},
	}

	cm, annotations, err := BuildConfigMap(opts)
	require.NoError(t, err)
	require.NotNil(t, cm.Data)
	require.NotNil(t, cm.Data["tempo.yaml"])
	require.Equal(t, map[string]string{
		"tempo.grafana.com/tempoConfig.hash": fmt.Sprintf("%x", sha256.Sum256([]byte(cm.Data["tempo.yaml"]))),
	}, annotations)

	require.NotNil(t, cm.Data["tempo-query.yaml"])
	tempoQueryCfg := `
address: 0.0.0.0:7777
backend: localhost:3200
tenant_header_key: x-scope-orgid
services_query_duration: 72h0m0s
find_traces_concurrent_requests: 22
`
	require.YAMLEq(t, tempoQueryCfg, cm.Data["tempo-query.yaml"])
}

func TestBuildConfig(t *testing.T) {
	tests := []struct {
		name     string
		spec     v1alpha1.TempoMonolithicSpec
		opts     Options
		expected string
	}{
		{
			name: "memory storage",
			spec: v1alpha1.TempoMonolithicSpec{},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
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
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "OTLP/gRPC with TLS",
			spec: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled:    true,
								CA:         "ca",
								Cert:       "cert",
								MinVersion: "1.3",
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
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
          endpoint: 0.0.0.0:4317
          tls:
            client_ca_file: /var/run/ca-receiver/grpc/service-ca.crt
            cert_file: /var/run/tls/receiver/grpc/tls.crt
            key_file: /var/run/tls/receiver/grpc/tls.key
            min_version: "1.3"
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "OTLP/HTTP with TLS",
			spec: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: false,
						},
						HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled:    true,
								CA:         "ca",
								Cert:       "cert",
								MinVersion: "1.3",
							},
						},
					},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
        http:
          endpoint: 0.0.0.0:4318
          tls:
            client_ca_file: /var/run/ca-receiver/http/service-ca.crt
            cert_file: /var/run/tls/receiver/http/tls.crt
            key_file: /var/run/tls/receiver/http/tls.key
            min_version: "1.3"
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "OTLP/HTTP and OTLP/Grpc with TLS",
			spec: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled:    true,
								CA:         "ca",
								Cert:       "cert",
								MinVersion: "1.3",
							},
						},
						HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled:    true,
								CA:         "ca",
								Cert:       "cert",
								MinVersion: "1.3",
							},
						},
					},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
        http:
          endpoint: 0.0.0.0:4318
          tls:
            client_ca_file: /var/run/ca-receiver/http/service-ca.crt
            cert_file: /var/run/tls/receiver/http/tls.crt
            key_file: /var/run/tls/receiver/http/tls.key
            min_version: "1.3"
        grpc:
          endpoint: 0.0.0.0:4317
          tls:
            client_ca_file: /var/run/ca-receiver/grpc/service-ca.crt
            cert_file: /var/run/tls/receiver/grpc/tls.crt
            key_file: /var/run/tls/receiver/grpc/tls.key
            min_version: "1.3"
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "OTLP/gRPC with TLS Profile",
			spec: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled: true,
								CA:      "ca",
								Cert:    "cert",
							},
						},
						HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
							Enabled: false,
						},
					},
				},
			},
			opts: Options{
				TLSProfile: tlsprofile.TLSProfileOptions{
					MinTLSVersion: "VersionTLS12",
					Ciphers:       []string{"abc"},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
          endpoint: 0.0.0.0:4317
          tls:
            client_ca_file: /var/run/ca-receiver/grpc/service-ca.crt
            cert_file: /var/run/tls/receiver/grpc/tls.crt
            key_file: /var/run/tls/receiver/grpc/tls.key
            min_version: "1.2"
            cipher_suites: [abc]
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "S3 storage with TLS Profile",
			spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: v1alpha1.MonolithicTracesStorageBackendS3,
						S3: &v1alpha1.MonolithicTracesStorageS3Spec{
							TLS: &v1alpha1.TLSSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			opts: Options{
				TLSProfile: tlsprofile.TLSProfileOptions{
					MinTLSVersion: "VersionTLS12",
					Ciphers:       []string{"abc", "def"},
				},
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeStatic,
					S3: &manifestutils.S3{
						Endpoint: "minio",
						Bucket:   "tempo",
					},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
storage:
  trace:
    backend: s3
    wal:
      path: /var/tempo/wal
    s3:
      endpoint: minio
      bucket: tempo
      insecure: false
      tls_min_version: VersionTLS12
      tls_cipher_suites: abc,def
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "extra config",
			spec: v1alpha1.TempoMonolithicSpec{
				ExtraConfig: &v1alpha1.ExtraConfigSpec{
					Tempo: apiextensionsv1.JSON{Raw: []byte(`{"storage": {"trace": {"wal": {"overlay_setting": "abc"}}}, "server": {"http_server_read_timeout": "1m", "http_server_write_timeout": "1m"}}`)},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 1m
  http_server_write_timeout: 1m
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
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
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
usage_report:
  reporting_enabled: false
`,
		},
		{
			name: "azure federated token",
			spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: v1alpha1.MonolithicTracesStorageBackendAzure,
						Azure: &v1alpha1.MonolithicTracesObjectStorageSpec{
							Secret: "azure-sts-secret",
						},
					},
				},
			},
			opts: Options{
				StorageParams: manifestutils.StorageParams{
					CredentialMode: v1alpha1.CredentialModeToken,
					AzureStorage: &manifestutils.AzureStorage{
						Container: "minio",
					},
				},
			},
			expected: `
server:
  http_listen_port: 3200
  http_server_read_timeout: 30s
  http_server_write_timeout: 30s
internal_server:
  enable: true
  http_listen_address: 0.0.0.0
storage:
  trace:
    backend: azure
    wal:
      path: /var/tempo/wal
    azure:
      container_name: minio
      use_federated_token: true
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
usage_report:
  reporting_enabled: false
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.opts.Tempo = v1alpha1.TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sample",
					Namespace: "default",
				},
				Spec: test.spec,
			}
			test.opts.Tempo.Default(test.opts.CtrlConfig)

			cfg, err := buildTempoConfig(test.opts)
			require.NoError(t, err)
			require.YAMLEq(t, test.expected, string(cfg))
		})
	}
}
