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
	TLS      tempoReceiverTLSConfig `yaml:"tls,omitempty"`
	Endpoint string                 `yaml:"endpoint,omitempty"`
}

type tempoLocalConfig struct {
	Path string `yaml:"path"`
}
type tempoS3Config struct {
	Endpoint      string `yaml:"endpoint"`
	Insecure      bool   `yaml:"insecure"`
	Bucket        string `yaml:"bucket"`
	TLSCAPath     string `yaml:"tls_ca_path,omitempty"`
	TLSCertPath   string `yaml:"tls_cert_path,omitempty"`
	TLSKeyPath    string `yaml:"tls_key_path,omitempty"`
	TLSMinVersion string `yaml:"tls_min_version,omitempty"`
}
type tempoAzureConfig struct {
	ContainerName string `yaml:"container_name"`
}
type tempoGCSConfig struct {
	BucketName string `yaml:"bucket_name"`
}

type tempoConfig struct {
	MultitenancyEnabled bool `yaml:"multitenancy_enabled,omitempty"`

	Server struct {
		HTTPListenAddress string `yaml:"http_listen_address,omitempty"`
		HttpListenPort    int    `yaml:"http_listen_port,omitempty"`
		GRPCListenAddress string `yaml:"grpc_listen_address,omitempty"`
	} `yaml:"server"`

	InternalServer struct {
		Enable            bool   `yaml:"enable,omitempty"`
		HTTPListenAddress string `yaml:"http_listen_address,omitempty"`
	} `yaml:"internal_server"`

	Storage struct {
		Trace struct {
			Backend string `yaml:"backend"`
			WAL     struct {
				Path string `yaml:"path"`
			} `yaml:"wal"`
			Local *tempoLocalConfig `yaml:"local,omitempty"`
			S3    *tempoS3Config    `yaml:"s3,omitempty"`
			Azure *tempoAzureConfig `yaml:"azure,omitempty"`
			GCS   *tempoGCSConfig   `yaml:"gcs,omitempty"`
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
func BuildConfigMap(opts Options) (*corev1.ConfigMap, map[string]string, error) {
	tempo := opts.Tempo
	extraAnnotations := map[string]string{}
	labels := ComponentLabels(manifestutils.TempoConfigName, tempo.Name)

	tempoConfig, err := buildTempoConfig(opts)
	if err != nil {
		return nil, nil, err
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.TempoConfigName, tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"tempo.yaml": string(tempoConfig),
		},
	}

	h := sha256.Sum256(tempoConfig)
	extraAnnotations["tempo.grafana.com/tempoConfig.hash"] = fmt.Sprintf("%x", h)

	if tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled {
		tempoQueryConfig, err := buildTempoQueryConfig()
		if err != nil {
			return nil, nil, err
		}
		configMap.Data["tempo-query.yaml"] = string(tempoQueryConfig)
	}

	return configMap, extraAnnotations, nil
}

func configureReceiverTLS(tlsSpec *v1alpha1.TLSSpec) tempoReceiverTLSConfig {
	tlsCfg := tempoReceiverTLSConfig{}
	if tlsSpec != nil && tlsSpec.Enabled {
		if tlsSpec.Cert != "" {
			tlsCfg.CertFile = path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSCertFilename)
			tlsCfg.KeyFile = path.Join(manifestutils.ReceiverTLSCertDir, manifestutils.TLSKeyFilename)
		}
		if tlsSpec.CA != "" {
			tlsCfg.CAFile = path.Join(manifestutils.ReceiverTLSCADir, manifestutils.TLSCAFilename)
		}
		tlsCfg.MinVersion = tlsSpec.MinVersion
	}
	return tlsCfg
}

func buildTempoConfig(opts Options) ([]byte, error) {
	tempo := opts.Tempo

	config := tempoConfig{}
	config.MultitenancyEnabled = tempo.Spec.Multitenancy != nil && tempo.Spec.Multitenancy.Enabled
	config.Server.HttpListenPort = manifestutils.PortHTTPServer
	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		// all connections to tempo must go via gateway
		config.Server.HTTPListenAddress = "localhost"
		config.Server.GRPCListenAddress = "localhost"
	}

	// The internal server is required because if the gateway is enabled,
	// the Tempo API will listen on localhost only,
	// and then Kubernetes cannot reach the health check endpoint.
	config.InternalServer.Enable = true
	config.InternalServer.HTTPListenAddress = "0.0.0.0"

	// The internal server is required because if the gateway is enabled,
	// the Tempo API will listen on localhost only,
	// and then Kubernetes cannot reach the health check endpoint.
	config.InternalServer.Enable = true
	config.InternalServer.HTTPListenAddress = "0.0.0.0"

	if tempo.Spec.Storage != nil {
		config.Storage.Trace.WAL.Path = "/var/tempo/wal"
		switch tempo.Spec.Storage.Traces.Backend {
		case v1alpha1.MonolithicTracesStorageBackendMemory,
			v1alpha1.MonolithicTracesStorageBackendPV:
			config.Storage.Trace.Backend = "local"
			config.Storage.Trace.Local = &tempoLocalConfig{}
			config.Storage.Trace.Local.Path = "/var/tempo/blocks"

		case v1alpha1.MonolithicTracesStorageBackendS3:
			config.Storage.Trace.Backend = "s3"
			config.Storage.Trace.S3 = &tempoS3Config{}
			config.Storage.Trace.S3.Endpoint = opts.StorageParams.S3.Endpoint
			config.Storage.Trace.S3.Insecure = opts.StorageParams.S3.Insecure
			config.Storage.Trace.S3.Bucket = opts.StorageParams.S3.Bucket
			if tempo.Spec.Storage.Traces.S3 != nil && tempo.Spec.Storage.Traces.S3.TLS != nil && tempo.Spec.Storage.Traces.S3.TLS.Enabled {
				if tempo.Spec.Storage.Traces.S3.TLS.CA != "" {
					config.Storage.Trace.S3.TLSCAPath = path.Join(manifestutils.StorageTLSCADir, opts.StorageParams.S3.TLS.CAFilename)
				}
				if tempo.Spec.Storage.Traces.S3.TLS.Cert != "" {
					config.Storage.Trace.S3.TLSCertPath = path.Join(manifestutils.StorageTLSCertDir, manifestutils.TLSCertFilename)
					config.Storage.Trace.S3.TLSKeyPath = path.Join(manifestutils.StorageTLSCertDir, manifestutils.TLSKeyFilename)
				}
				config.Storage.Trace.S3.TLSMinVersion = tempo.Spec.Storage.Traces.S3.TLS.MinVersion
			}

		case v1alpha1.MonolithicTracesStorageBackendAzure:
			config.Storage.Trace.Backend = "azure"
			config.Storage.Trace.Azure = &tempoAzureConfig{}
			config.Storage.Trace.Azure.ContainerName = opts.StorageParams.AzureStorage.Container

		case v1alpha1.MonolithicTracesStorageBackendGCS:
			config.Storage.Trace.Backend = "gcs"
			config.Storage.Trace.GCS = &tempoGCSConfig{}
			config.Storage.Trace.GCS.BucketName = opts.StorageParams.GCS.Bucket

		default:
			return nil, fmt.Errorf("invalid storage backend: '%s'", tempo.Spec.Storage.Traces.Backend)
		}
	}

	if tempo.Spec.Ingestion != nil {
		if tempo.Spec.Ingestion.OTLP != nil {
			if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
				config.Distributor.Receivers.OTLP.Protocols.GRPC = &tempoReceiverConfig{
					TLS: configureReceiverTLS(tempo.Spec.Ingestion.OTLP.GRPC.TLS),
				}
				if tempo.Spec.Multitenancy.IsGatewayEnabled() {
					// all connections to tempo must go via gateway
					config.Distributor.Receivers.OTLP.Protocols.GRPC.Endpoint = fmt.Sprintf("localhost:%d", manifestutils.PortOtlpGrpcServer)
				}
			}

			if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
				config.Distributor.Receivers.OTLP.Protocols.HTTP = &tempoReceiverConfig{
					TLS: configureReceiverTLS(tempo.Spec.Ingestion.OTLP.HTTP.TLS),
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
