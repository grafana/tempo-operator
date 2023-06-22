package manifestutils

import (
	"net"

	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

// PatchTracingJaegerEnv adds configures jaeger-sdk via environment variables if
// operand observability settings exist.
func PatchTracingJaegerEnv(tempo v1alpha1.TempoStack, pod corev1.PodTemplateSpec) (corev1.PodTemplateSpec, error) {
	if tempo.Spec.Observability.Tracing.SamplingFraction == "" {
		return pod, nil
	}
	host, port, err := net.SplitHostPort(tempo.Spec.Observability.Tracing.JaegerAgentEndpoint)
	if err != nil {
		return corev1.PodTemplateSpec{}, err
	}

	container := corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "JAEGER_AGENT_HOST",
				Value: host,
			},
			{
				Name:  "JAEGER_AGENT_PORT",
				Value: port,
			},
			{
				Name:  "JAEGER_SAMPLER_TYPE",
				Value: "const",
			},
			{
				Name:  "JAEGER_SAMPLER_PARAM",
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
