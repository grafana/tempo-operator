package config

import (
	"crypto/sha256"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const tenantOverridesMountPath = "/conf/overrides.yaml"
const tempoConfigKey = "tempo.yaml"
const tempoQueryFrontendConfigKey = "tempo-query-frontend.yaml"
const tempoQueryConfigKey = "tempo-query.yaml"
const overridesConfigKey = "overrides.yaml"

// BuildConfigMap builds the tempo configuration file and the tenant-specific overrides configuration.
// It returns a ConfigMap containing both configuration files and the checksum of the main configuration file
// (the tenant-specific configuration gets reloaded automatically, therefore no checksum is required).
func BuildConfigMap(params manifestutils.Params) (*corev1.ConfigMap, string, error) {
	tempo := params.Tempo

	config, err := buildConfiguration(params)
	if err != nil {
		return nil, "", err
	}

	overridesConfig, err := buildTenantOverrides(tempo)
	if err != nil {
		return nil, "", err
	}

	frontendConfig, err := buildQueryFrontEndConfig(params)
	if err != nil {
		return nil, "", err
	}

	if params.Tempo.Spec.ExtraConfig != nil {
		// For we only support tempo for now.
		config, err = MergeExtraConfigWithConfig(params.Tempo.Spec.ExtraConfig.Tempo, config)
		if err != nil {
			return nil, "", err
		}

		// Is the same tempo config with certain TLS fields disabled.
		frontendConfig, err = MergeExtraConfigWithConfig(params.Tempo.Spec.ExtraConfig.Tempo, frontendConfig)
		if err != nil {
			return nil, "", err
		}
	}

	labels := manifestutils.ComponentLabels("config", tempo.Name)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name("", tempo.Name),
			Namespace: tempo.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			tempoConfigKey:              string(config),
			tempoQueryFrontendConfigKey: string(frontendConfig),
			overridesConfigKey:          string(overridesConfig),
		},
	}
	if tempo.Spec.Template.QueryFrontend.JaegerQuery.Enabled {
		tempoQueryConfig, err := buildTempoQueryConfig(params)
		if err != nil {
			return nil, "", err
		}
		configMap.Data[tempoQueryConfigKey] = string(tempoQueryConfig)
	}

	// We only need to hash the main ConfigMap, the per-tenant overrides
	// is reloaded by tempo without requiring a restart
	h := sha256.Sum256(config)
	checksum := fmt.Sprintf("%x", h)

	return configMap, checksum, nil
}
