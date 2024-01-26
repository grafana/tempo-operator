package monolithic

import (
	"crypto/sha256"
	"fmt"
	"path"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	tempoStackConfig "github.com/grafana/tempo-operator/internal/manifests/config"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

type tempoReceiverTLSConfig struct {
	CAFile     string `yaml:"client_ca_file,omitempty"`
	CertFile   string `yaml:"cert_file,omitempty"`
	KeyFile    string `yaml:"key_file,omitempty"`
	MinVersion string `yaml:"min_version,omitempty"`
}

type tempoReceiverConfig struct {
	TLS tempoReceiverTLSConfig `yaml:"tls,omitempty"`
}

type tempoConfig struct {
	Server struct {
		HttpListenPort int `yaml:"http_listen_port"`
	} `yaml:"server"`

	Storage struct {
		Trace struct {
			Backend string `yaml:"backend"`
			WAL     struct {
				Path string `yaml:"path"`
			} `yaml:"wal"`
			Local struct {
				Path string `yaml:"path"`
			} `yaml:"local"`
		} `yaml:"trace"`
	} `yaml:"storage"`

	Distributor struct {
		Receivers struct {
			OTLP struct {
				Protocols struct {
					GRPC *tempoReceiverConfig `yaml:"grpc,omitempty"`
					HTTP *tempoReceiverConfig `yaml:"http,omitempty"`
				} `yaml:"protocols,omitempty"`
			} `yaml:"otlp,omitempty"`
		} `yaml:"receivers,omitempty"`
	} `yaml:"distributor,omitempty"`

	UsageReport struct {
		ReportingEnabled bool `yaml:"reporting_enabled"`
	} `yaml:"usage_report"`
}

type tempoQueryConfig struct {
	Backend         string `yaml:"backend"`
	TenantHeaderKey string `yaml:"tenant_header_key"`
}

// BuildConfigMap creates the Tempo ConfigMap for a monolithic deployment.
func BuildConfigMap(opts Options) (*corev1.ConfigMap, string, error) {
	tempo := opts.Tempo
	labels := Labels(tempo.Name)

	tempoConfig, err := buildTempoConfig(opts)
	if err != nil {
		return nil, "", err
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("", tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"tempo.yaml": string(tempoConfig),
		},
	}

	h := sha256.Sum256(tempoConfig)
	checksum := fmt.Sprintf("%x", h)

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		tempoQueryConfig, err := buildTempoQueryConfig()
		if err != nil {
			return nil, "", err
		}
		configMap.Data["tempo-query.yaml"] = string(tempoQueryConfig)
	}

	return configMap, checksum, nil
}

func buildTempoConfig(opts Options) ([]byte, error) {
	tempo := opts.Tempo

	config := tempoConfig{}
	config.Server.HttpListenPort = manifestutils.PortHTTPServer

	config.Storage.Trace.WAL.Path = "/var/tempo/wal"
	switch tempo.Spec.Storage.Traces.Backend {
	case v1alpha1.MonolithicTracesStorageBackendMemory,
		v1alpha1.MonolithicTracesStorageBackendPV:
		config.Storage.Trace.Backend = "local"
		config.Storage.Trace.Local.Path = "/var/tempo/blocks"

	default:
		return nil, fmt.Errorf("invalid storage backend: '%s'", tempo.Spec.Storage.Traces.Backend)
	}

	if tempo.Spec.Ingestion != nil {
		tls := tempoReceiverTLSConfig{}
		if tempo.Spec.Ingestion.TLS != nil && tempo.Spec.Ingestion.TLS.Enabled {
			if tempo.Spec.Ingestion.TLS.Cert != "" {
				tls.CertFile = path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSCertFilename)
				tls.KeyFile = path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSKeyFilename)
			}
			if tempo.Spec.Ingestion.TLS.CA != "" {
				tls.CAFile = path.Join(manifestutils.ReceiverTLSCADir, manifestutils.TLSCAFilename)
			}
			tls.MinVersion = tempo.Spec.Ingestion.TLS.MinVersion
		}

		if tempo.Spec.Ingestion.OTLP != nil {
			if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
				config.Distributor.Receivers.OTLP.Protocols.GRPC = &tempoReceiverConfig{
					TLS: tls,
				}
			}
			if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
				config.Distributor.Receivers.OTLP.Protocols.HTTP = &tempoReceiverConfig{
					TLS: tls,
				}
			}
		}
	}

	generatedYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	if tempo.Spec.ExtraConfig == nil || len(tempo.Spec.ExtraConfig.Tempo.Raw) == 0 {
		return generatedYaml, nil
	} else {
		return tempoStackConfig.MergeExtraConfigWithConfig(tempo.Spec.ExtraConfig.Tempo, generatedYaml)
	}
}

func buildTempoQueryConfig() ([]byte, error) {
	config := tempoQueryConfig{}
	config.Backend = fmt.Sprintf("127.0.0.1:%d", manifestutils.PortHTTPServer)
	config.TenantHeaderKey = manifestutils.TenantHeader

	return yaml.Marshal(&config)
}
