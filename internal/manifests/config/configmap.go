package config

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

// Params holds configuration parameters.
type Params struct {
	S3 S3Options
}

// BuildConfigMap creates configuration objects.
func BuildConfigMap(tempo v1alpha1.Microservices, params Params) (*corev1.ConfigMap, error) {
	params.S3.Insecure = false
	if strings.HasPrefix(params.S3.Endpoint, "http://") {
		params.S3.Insecure = true
		params.S3.Endpoint = strings.TrimPrefix(params.S3.Endpoint, "http://")
	} else if !strings.HasPrefix(params.S3.Endpoint, "https://") {
		params.S3.Insecure = true
	} else {
		params.S3.Endpoint = strings.TrimPrefix(params.S3.Endpoint, "https://")
	}

	config, err := buildConfiguration(Options{
		S3:              params.S3,
		GlobalRetention: tempo.Spec.Retention.Global.Traces.String(),
		MemberList: []string{
			manifestutils.Name("gossip-ring", tempo.Name),
		},
		QueryFrontendDiscovery: fmt.Sprintf("%s:9095", manifestutils.Name("query-frontend-discovery", tempo.Name)),
	})
	if err != nil {
		return nil, err
	}

	labels := manifestutils.ComponentLabels("config", tempo.Name)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   manifestutils.Name("", tempo.Name),
			Labels: labels,
		},
		Data: map[string]string{
			"tempo.yaml": string(config),
		},
	}
	if tempo.Spec.Components.QueryFrontend != nil && tempo.Spec.Components.QueryFrontend.JaegerQuery.Enabled {
		configMap.Data["tempo-query.yaml"] = "backend: 127.0.0.1:3100\n"
	}

	return configMap, nil
}
