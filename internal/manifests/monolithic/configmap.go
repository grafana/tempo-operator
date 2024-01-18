package monolithic

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

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
					GRPC *interface{} `yaml:"grpc,omitempty"`
					HTTP *interface{} `yaml:"http,omitempty"`
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

	if tempo.Spec.Ingestion != nil && tempo.Spec.Ingestion.OTLP != nil {
		if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
			var i interface{}
			config.Distributor.Receivers.OTLP.Protocols.GRPC = &i
		}
		if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
			var i interface{}
			config.Distributor.Receivers.OTLP.Protocols.HTTP = &i
		}
	}

	if tempo.Spec.ExtraConfig == nil || len(tempo.Spec.ExtraConfig.Tempo.Raw) == 0 {
		return yaml.Marshal(config)
	} else {
		return overlayJson(config, tempo.Spec.ExtraConfig.Tempo.Raw)
	}
}

func overlayJson(config tempoConfig, overlay []byte) ([]byte, error) {
	// mergo.Merge requires that both variables have the same type
	generatedCfg := make(map[string]interface{})
	overlayCfg := make(map[string]interface{})

	// Convert tempoConfig{} to map[string]interface{}
	generatedYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(generatedYaml, &generatedCfg); err != nil {
		return nil, err
	}

	// Unmarshal overlay of type []byte to map[string]interface{}
	if err := json.Unmarshal(overlay, &overlayCfg); err != nil {
		return nil, err
	}

	// Override generated config with extra config
	err = mergo.Merge(&generatedCfg, overlayCfg, mergo.WithOverride)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(generatedCfg)
}

func buildTempoQueryConfig() ([]byte, error) {
	config := tempoQueryConfig{}
	config.Backend = fmt.Sprintf("127.0.0.1:%d", manifestutils.PortHTTPServer)
	config.TenantHeaderKey = manifestutils.TenantHeader

	return yaml.Marshal(&config)
}
