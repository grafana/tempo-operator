package config

import (
	"crypto/sha256"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
	"github.com/os-observability/tempo-operator/internal/tlsprofile"
)

const tenantOverridesMountPath = "/conf/overrides.yaml"

// Params holds configuration parameters.
type Params struct {
	AzureStorage   AzureStorage
	GCS            GCS
	S3             S3
	HTTPEncryption bool
	GRPCEncryption bool
	TLSProfile     tlsprofile.TLSProfileOptions
}

// AzureStorage holds AzureStorage object storage configuration options.
type AzureStorage struct {
	Container string
}

// GCS holds Google Cloud Storage object storage configuration options.
type GCS struct {
	Bucket string
}

// S3 holds S3 object storage configuration options.
type S3 struct {
	Endpoint string
	Bucket   string
	Insecure bool
}

// BuildConfigMap builds the tempo configuration file and the tenant-specific overrides configuration.
// It returns a ConfigMap containing both configuration files and the checksum of the main configuration file
// (the tenant-specific configuration gets reloaded automatically, therefore no checksum is required).
func BuildConfigMap(tempo v1alpha1.TempoStack, params Params) (*corev1.ConfigMap, string, error) {
	config, err := buildConfiguration(tempo, params)
	if err != nil {
		return nil, "", err
	}

	overridesConfig, err := buildTenantOverrides(tempo)
	if err != nil {
		return nil, "", err
	}

	labels := manifestutils.ComponentLabels("config", tempo.Name)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   naming.Name("", tempo.Name),
			Labels: labels,
		},
		Data: map[string]string{
			"tempo.yaml":     string(config),
			"overrides.yaml": string(overridesConfig),
		},
	}
	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		tempoQueryConfig, err := buildTempoQueryConfig(tempo, params)
		if err != nil {
			return nil, "", err
		}
		configMap.Data["tempo-query.yaml"] = string(tempoQueryConfig)
	}

	// We only need to hash the main ConfigMap, the per-tenant overrides
	// is reloaded by tempo without requiring a restart
	h := sha256.Sum256(config)
	checksum := fmt.Sprintf("%x", h)

	return configMap, checksum, nil
}
