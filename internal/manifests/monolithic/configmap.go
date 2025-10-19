package monolithic

import (
	"crypto/sha256"
	"fmt"
	"path"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	tempoStackConfig "github.com/grafana/tempo-operator/internal/manifests/config"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
)

type tempoReceiverTLSConfig struct {
	CAFile       string   `yaml:"client_ca_file,omitempty"`
	CertFile     string   `yaml:"cert_file,omitempty"`
	KeyFile      string   `yaml:"key_file,omitempty"`
	MinVersion   string   `yaml:"min_version,omitempty"`
	CipherSuites []string `yaml:"cipher_suites,omitempty"`
}

type tempoReceiverConfig struct {
	TLS      tempoReceiverTLSConfig `yaml:"tls,omitempty"`
	Endpoint string                 `yaml:"endpoint,omitempty"`
}

type tempoLocalConfig struct {
	Path string `yaml:"path"`
}
type tempoS3Config struct {
	Endpoint        string `yaml:"endpoint"`
	Insecure        bool   `yaml:"insecure"`
	Bucket          string `yaml:"bucket"`
	TLSCAPath       string `yaml:"tls_ca_path,omitempty"`
	TLSCertPath     string `yaml:"tls_cert_path,omitempty"`
	TLSKeyPath      string `yaml:"tls_key_path,omitempty"`
	TLSMinVersion   string `yaml:"tls_min_version,omitempty"`
	TLSCipherSuites string `yaml:"tls_cipher_suites,omitempty"`
}
type tempoAzureConfig struct {
	ContainerName     string `yaml:"container_name"`
	UseFederatedToken bool   `yaml:"use_federated_token,omitempty"`
}
type tempoGCSConfig struct {
	BucketName string `yaml:"bucket_name"`
}
type tempoHTTPTLSConfig struct {
	CertFile       string `yaml:"cert_file,omitempty"`
	KeyFile        string `yaml:"key_file,omitempty"`
	ClientAuthType string `yaml:"client_auth_type,omitempty"`
	ClientCAFile   string `yaml:"client_ca_file,omitempty"`
}
type tempoConfig struct {
	MultitenancyEnabled bool `yaml:"multitenancy_enabled,omitempty"`

	Server struct {
		HTTPListenAddress      string              `yaml:"http_listen_address,omitempty"`
		HttpListenPort         int                 `yaml:"http_listen_port,omitempty"`
		GRPCListenAddress      string              `yaml:"grpc_listen_address,omitempty"`
		HttpServerReadTimeout  time.Duration       `yaml:"http_server_read_timeout,omitempty"`
		HttpServerWriteTimeout time.Duration       `yaml:"http_server_write_timeout,omitempty"`
		HTTPTLSConfig          *tempoHTTPTLSConfig `yaml:"http_tls_config,omitempty"`
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
	Address                      string        `yaml:"address"`
	Backend                      string        `yaml:"backend"`
	TenantHeaderKey              string        `yaml:"tenant_header_key"`
	ServicesQueryDuration        time.Duration `yaml:"services_query_duration"`
	FindTracesConcurrentRequests int           `yaml:"find_traces_concurrent_requests"`
	TLSEnabled                   bool          `yaml:"tls_enabled,omitempty"`
	TLSCertPath                  *string       `yaml:"tls_cert_path,omitempty"`
	TLSKeyPath                   *string       `yaml:"tls_key_path,omitempty"`
	TLSCAPath                    *string       `yaml:"tls_ca_path,omitempty"`
	TLSInsecureSkipVerify        *bool         `yaml:"tls_insecure_skip_verify,omitempty"`
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
			APIVersion: corev1.SchemeGroupVersion.String(),
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

		enableTLS := tempo.Spec.Multitenancy.IsGatewayEnabled() && opts.CtrlConfig.Gates.HTTPEncryption
		tempoQueryConfig, err := buildTempoQueryConfig(tempo.Spec.JaegerUI, enableTLS)
		if err != nil {
			return nil, nil, err
		}
		configMap.Data["tempo-query.yaml"] = string(tempoQueryConfig)
	}

	return configMap, extraAnnotations, nil
}

func configureReceiverTLS(tlsSpec *v1alpha1.TLSSpec, tlsProfile tlsprofile.TLSProfileOptions, caCertDir, certDir string) (tempoReceiverTLSConfig, error) {
	tlsCfg := tempoReceiverTLSConfig{}
	if tlsSpec != nil && tlsSpec.Enabled {
		if tlsSpec.Cert != "" {
			tlsCfg.CertFile = path.Join(certDir, manifestutils.TLSCertFilename)
			tlsCfg.KeyFile = path.Join(certDir, manifestutils.TLSKeyFilename)
		}
		if tlsSpec.CA != "" {
			tlsCfg.CAFile = path.Join(caCertDir, manifestutils.TLSCAFilename)
		}
		if tlsSpec.MinVersion != "" {
			tlsCfg.MinVersion = tlsSpec.MinVersion
		} else if tlsProfile.MinTLSVersion != "" {
			var err error
			tlsCfg.MinVersion, err = tlsProfile.MinVersionShort()
			if err != nil {
				return tempoReceiverTLSConfig{}, err
			}
		}
		tlsCfg.CipherSuites = tlsProfile.Ciphers
	}
	return tlsCfg, nil
}

func buildTempoConfig(opts Options) ([]byte, error) {
	tempo := opts.Tempo

	config := tempoConfig{}
	config.MultitenancyEnabled = tempo.Spec.Multitenancy != nil && tempo.Spec.Multitenancy.Enabled
	config.Server.HttpListenPort = manifestutils.PortHTTPServer
	config.Server.HttpServerReadTimeout = opts.Tempo.Spec.Timeout.Duration
	config.Server.HttpServerWriteTimeout = opts.Tempo.Spec.Timeout.Duration
	if tempo.Spec.Multitenancy.IsGatewayEnabled() {
		// We need this to scrap metrics.
		config.Server.HTTPListenAddress = "0.0.0.0"
		// all connections to tempo must go via gateway
		config.Server.GRPCListenAddress = "localhost"

		if opts.CtrlConfig.Gates.HTTPEncryption {
			config.Server.HTTPTLSConfig = &tempoHTTPTLSConfig{
				CertFile:       path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename),
				KeyFile:        path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename),
				ClientCAFile:   path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename),
				ClientAuthType: "RequireAndVerifyClientCert",
			}
		}

	}

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
			if opts.StorageParams.CredentialMode == v1alpha1.CredentialModeStatic {
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
					if tempo.Spec.Storage.Traces.S3.TLS.MinVersion != "" {
						config.Storage.Trace.S3.TLSMinVersion = tempo.Spec.Storage.Traces.S3.TLS.MinVersion
					} else if opts.TLSProfile.MinTLSVersion != "" {
						config.Storage.Trace.S3.TLSMinVersion = opts.TLSProfile.MinTLSVersion
					}
					config.Storage.Trace.S3.TLSCipherSuites = opts.TLSProfile.TLSCipherSuites()
				}
			} else if opts.StorageParams.CredentialMode == v1alpha1.CredentialModeToken || opts.StorageParams.CredentialMode == v1alpha1.CredentialModeTokenCCO {
				config.Storage.Trace.S3.Bucket = opts.StorageParams.S3.Bucket
				config.Storage.Trace.S3.Endpoint = fmt.Sprintf("s3.%s.amazonaws.com", opts.StorageParams.S3.Region)
			}

		case v1alpha1.MonolithicTracesStorageBackendAzure:
			config.Storage.Trace.Backend = "azure"
			config.Storage.Trace.Azure = &tempoAzureConfig{}
			config.Storage.Trace.Azure.ContainerName = opts.StorageParams.AzureStorage.Container
			config.Storage.Trace.Azure.UseFederatedToken = opts.StorageParams.CredentialMode == v1alpha1.CredentialModeToken

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
			// It seems like the gateway try to report grpc using mTLS even if only HTTP encryption is enabled.
			if tempo.Spec.Multitenancy.IsGatewayEnabled() && opts.CtrlConfig.Gates.HTTPEncryption {
				config.Distributor.Receivers.OTLP.Protocols.GRPC = &tempoReceiverConfig{
					TLS: tempoReceiverTLSConfig{
						CertFile: path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename),
						CAFile:   path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename),
						KeyFile:  path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename),
					},
				}
			} else {
				if tempo.Spec.Ingestion.OTLP.GRPC != nil && tempo.Spec.Ingestion.OTLP.GRPC.Enabled {
					receiverTLS, err := configureReceiverTLS(tempo.Spec.Ingestion.OTLP.GRPC.TLS, opts.TLSProfile,
						manifestutils.ReceiverGRPCTLSCADir, manifestutils.ReceiverGRPCTLSCertDir)

					if err != nil {
						return nil, err
					}

					config.Distributor.Receivers.OTLP.Protocols.GRPC = &tempoReceiverConfig{
						TLS: receiverTLS,
					}

					if tempo.Spec.Multitenancy.IsGatewayEnabled() {
						// all connections to tempo must go via gateway
						config.Distributor.Receivers.OTLP.Protocols.GRPC.Endpoint = fmt.Sprintf("localhost:%d", manifestutils.PortOtlpGrpcServer)
					} else {
						config.Distributor.Receivers.OTLP.Protocols.GRPC.Endpoint = fmt.Sprintf("0.0.0.0:%d", manifestutils.PortOtlpGrpcServer)
					}
				}
			}

			if tempo.Spec.Ingestion.OTLP.HTTP != nil && tempo.Spec.Ingestion.OTLP.HTTP.Enabled {
				receiverTLS, err := configureReceiverTLS(tempo.Spec.Ingestion.OTLP.HTTP.TLS,
					opts.TLSProfile, manifestutils.ReceiverHTTPTLSCADir, manifestutils.ReceiverHTTPTLSCertDir)
				if err != nil {
					return nil, err
				}

				config.Distributor.Receivers.OTLP.Protocols.HTTP = &tempoReceiverConfig{
					TLS: receiverTLS,
				}

				if tempo.Spec.Multitenancy.IsGatewayEnabled() {
					// all connections to tempo must go via gateway
					config.Distributor.Receivers.OTLP.Protocols.HTTP.Endpoint = fmt.Sprintf("localhost:%d", manifestutils.PortOtlpHttp)
				} else {
					config.Distributor.Receivers.OTLP.Protocols.HTTP.Endpoint = fmt.Sprintf("0.0.0.0:%d", manifestutils.PortOtlpHttp)
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

func buildTempoQueryConfig(jaegerUISpec *v1alpha1.MonolithicJaegerUISpec, enableTLS bool) ([]byte, error) {
	config := tempoQueryConfig{}
	config.Address = fmt.Sprintf("0.0.0.0:%d", manifestutils.PortTempoGRPCQuery)
	config.Backend = fmt.Sprintf("localhost:%d", manifestutils.PortHTTPServer)
	config.TenantHeaderKey = manifestutils.TenantHeader
	config.ServicesQueryDuration = jaegerUISpec.ServicesQueryDuration.Duration
	config.FindTracesConcurrentRequests = jaegerUISpec.FindTracesConcurrentRequests
	if enableTLS {
		config.TLSEnabled = true
		config.TLSCertPath = ptr.To(path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSCertFilename))
		config.TLSKeyPath = ptr.To(path.Join(manifestutils.TempoInternalTLSCertDir, manifestutils.TLSKeyFilename))
		config.TLSCAPath = ptr.To(path.Join(manifestutils.TempoInternalTLSCADir, manifestutils.TLSCAFilename))
		config.TLSInsecureSkipVerify = ptr.To(false)
	}
	return yaml.Marshal(&config)
}
