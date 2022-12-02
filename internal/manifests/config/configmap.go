package config

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

const tenantOverridesMountPath = "/conf/overrides.yaml"

// Params holds configuration parameters.
type Params struct {
	S3 S3
}

// S3 holds S3 object storage configuration options.
type S3 struct {
	Endpoint string
	Bucket   string
}

func BuildConfigMap(tempo v1alpha1.Microservices, params Params) (*corev1.ConfigMap, error) {
	config, err := buildConfiguration(tempo, params)
	if err != nil {
		return nil, err
	}

	overridesConfig, err := buildTenantOverrides(tempo)
	if err != nil {
		return nil, err
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
	if tempo.Spec.Components.QueryFrontend != nil && tempo.Spec.Components.QueryFrontend.JaegerQuery.Enabled {
		configMap.Data["tempo-query.yaml"] = "backend: 127.0.0.1:3100\n"
	}

	return configMap, nil
}
