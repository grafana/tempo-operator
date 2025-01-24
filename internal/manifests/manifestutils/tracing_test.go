package manifestutils

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func Test_PatchTracingJaegerEnv(t *testing.T) {
	tt := []struct {
		name       string
		inputTempo v1alpha1.TempoStack
		inputPod   corev1.PodTemplateSpec
		expectPod  corev1.PodTemplateSpec
		expectErr  error
	}{
		{
			name: "valid settings",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction: "1.0",
							OTLPHTTPEndpoint: "http://collector:4318",
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
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: "http://collector:4318",
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
							OTLPHTTPEndpoint: "---invalid----",
						},
					},
				},
			},
			inputPod:  corev1.PodTemplateSpec{},
			expectPod: corev1.PodTemplateSpec{},
			expectErr: &url.Error{
				Op:  "parse",
				URL: "---invalid----",
				Err: errors.New("invalid URI for request"),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pod, err := PatchTracingJaegerEnv(tc.inputTempo, tc.inputPod)
			require.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectPod, pod)
		})
	}
}
