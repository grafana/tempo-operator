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
	S3 S3
}

// S3 holds S3 object storage configuration options.
type S3 struct {
	Endpoint string
	Bucket   string
}

// BuildConfigMap creates configuration objects.
func BuildConfigMap(tempo v1alpha1.Microservices, params Params) (*corev1.ConfigMap, error) {
	s3Insecure := false
	s3Endpoint := params.S3.Endpoint
	if strings.HasPrefix(s3Endpoint, "http://") {
		s3Insecure = true
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "http://")
	} else if !strings.HasPrefix(s3Endpoint, "https://") {
		s3Insecure = true
	} else {
		s3Endpoint = strings.TrimPrefix(s3Endpoint, "https://")
	}

	config, err := buildConfiguration(options{
		S3: s3{
			Endpoint: s3Endpoint,
			Bucket:   params.S3.Bucket,
			Insecure: s3Insecure,
		},
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
