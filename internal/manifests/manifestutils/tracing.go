package manifestutils

import (
	"fmt"
	"net/url"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

// PatchTracingEnvConfiguration configures OTEL SDK via environment variables if
// operand observability settings exist.
func PatchTracingEnvConfiguration(tempo v1alpha1.TempoStack, pod corev1.PodTemplateSpec) (corev1.PodTemplateSpec, error) {
	if tempo.Spec.Observability.Tracing.SamplingFraction == "" {
		return pod, nil
	}
	_, err := url.ParseRequestURI(tempo.Spec.Observability.Tracing.OTLPHttpEndpoint)
	if err != nil {
		return corev1.PodTemplateSpec{}, fmt.Errorf("invalid OTLP/http endpoint: %v", err)
	}

	container := corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "OTEL_TRACES_EXPORTER",
				Value: "otlp",
			},
			{
				Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
				Value: tempo.Spec.Observability.Tracing.OTLPHttpEndpoint,
			},
			{
				Name:  "OTEL_TRACES_SAMPLER",
				Value: "parentbased_traceidratio",
			},
			{
				Name:  "OTEL_TRACES_SAMPLER_ARG",
				Value: tempo.Spec.Observability.Tracing.SamplingFraction,
			},
		},
	}

	for i := range pod.Spec.Containers {
		if err := mergo.Merge(&pod.Spec.Containers[i], container, mergo.WithAppendSlice); err != nil {
			return corev1.PodTemplateSpec{}, err
		}
	}

	return pod, mergo.Merge(&pod.Annotations, map[string]string{
		"sidecar.opentelemetry.io/inject": "true",
	})
}
