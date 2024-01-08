package monolithic

import (
	"testing"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildTempoApiService(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}

	svc := buildTempoApiService(opts)

	labels := ComponentLabels("tempo", "sample")
	require.Equal(t, &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample-api",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       3200,
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: labels,
		},
	}, svc)
}

func TestBuildTempoIngestService(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{},
		},
	}

	tests := []struct {
		name     string
		input    *v1alpha1.MonolithicIngestionSpec
		expected []corev1.ServicePort
	}{
		{
			name:     "no ingestion ports",
			input:    nil,
			expected: []corev1.ServicePort{},
		},
		{
			name: "OTLP/gRPC",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ServicePort{
				{
					Name:       "otlp-grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       4317,
					TargetPort: intstr.FromString("otlp-grpc"),
				},
			},
		},
		{
			name: "OTLP/HTTP",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ServicePort{
				{
					Name:       "otlp-http",
					Protocol:   corev1.ProtocolTCP,
					Port:       4318,
					TargetPort: intstr.FromString("otlp-http"),
				},
			},
		},
	}

	labels := ComponentLabels("tempo", "sample")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec.Ingestion = test.input
			svc := buildTempoIngestService(opts)
			require.Equal(t, &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tempo-sample-ingest",
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

func TestBuildJaegerUIService(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}

	svc := buildJaegerUIService(opts)

	labels := ComponentLabels("tempo", "sample")
	require.Equal(t, &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample-jaegerui",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
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
			Selector: labels,
		},
	}, svc)
}
