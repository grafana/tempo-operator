package monolithic

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildTempoService(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}

	tests := []struct {
		name     string
		input    v1alpha1.TempoMonolithicSpec
		expected []corev1.ServicePort
	}{
		{
			name:  "no ingestion ports, no jaeger ui",
			input: v1alpha1.TempoMonolithicSpec{},
			expected: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
		{
			name: "ingest OTLP/gRPC",
			input: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
			},
			expected: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "otlp-grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       4317,
					TargetPort: intstr.FromString("otlp-grpc"),
				},
			},
		},
		{
			name: "ingest OTLP/HTTP",
			input: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
							Enabled: true,
						},
					},
				},
			},
			expected: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "otlp-http",
					Protocol:   corev1.ProtocolTCP,
					Port:       4318,
					TargetPort: intstr.FromString("otlp-http"),
				},
			},
		},
		{
			name: "enable JaegerUI",
			input: v1alpha1.TempoMonolithicSpec{
				JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
					Enabled: true,
				},
			},
			expected: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "jaeger-grpc",
					Port:       16685,
					TargetPort: intstr.FromString("jaeger-grpc"),
				},
				{
					Name:       "jaeger-ui",
					Port:       16686,
					TargetPort: intstr.FromString("jaeger-ui"),
				},
				{
					Name:       "jaeger-metrics",
					Port:       16687,
					TargetPort: intstr.FromString("jaeger-metrics"),
				},
			},
		},
	}

	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, "sample")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec = test.input
			svc := BuildTempoService(opts)
			require.Equal(t, &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-sample",
					Namespace: "default",
					Labels:    labels,
				},
				Spec: corev1.ServiceSpec{
					Ports:    test.expected,
					Selector: labels,
				},
			}, svc)
		})
	}
}
