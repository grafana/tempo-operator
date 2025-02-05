package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func Test_PatchTracing(t *testing.T) {
	tt := []struct {
		name       string
		inputTempo v1alpha1.TempoStack
		inputPod   corev1.PodTemplateSpec
		expectPod  corev1.PodTemplateSpec
		expectErr  string
	}{
		{
			name: "valid settings",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction: "1.0",
							OTLPHttpEndpoint: "http://collector:1234",
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Env: []corev1.EnvVar{
								{
									Name:  "EXISTING_VAR",
									Value: "1234",
								},
							},
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com":                    "true",
						"sidecar.opentelemetry.io/inject": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Env: []corev1.EnvVar{
								{
									Name:  "EXISTING_VAR",
									Value: "1234",
								},
								{
									Name:  "OTEL_TRACES_EXPORTER",
									Value: "otlp",
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:1234",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER",
									Value: "parentbased_traceidratio",
								},
								{
									Name:  "OTEL_TRACES_SAMPLER_ARG",
									Value: "1.0",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no sampling param",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction: "",
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Env: []corev1.EnvVar{
								{
									Name:  "EXISTING_VAR",
									Value: "1234",
								},
							},
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Env: []corev1.EnvVar{
								{
									Name:  "EXISTING_VAR",
									Value: "1234",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "invalid agent address",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction: "0.5",
							OTLPHttpEndpoint: "---invalid----",
						},
					},
				},
			},
			inputPod:  corev1.PodTemplateSpec{},
			expectPod: corev1.PodTemplateSpec{},
			expectErr: "invalid OTLP/http endpoint: parse \"---invalid----\": invalid URI for request",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pod, err := PatchTracingEnvConfiguration(tc.inputTempo, tc.inputPod)
			if err != nil {
				require.EqualError(t, err, tc.expectErr)
			}
			assert.Equal(t, tc.expectPod, pod)
		})
	}
}
