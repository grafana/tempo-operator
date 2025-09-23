package monolithic

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestBuildServices(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
		},
	}

	tests := []struct {
		name                      string
		input                     v1alpha1.TempoMonolithicSpec
		addServiceCertAnnotations bool
		expected                  []client.Object
	}{
		{
			name:  "no ingestion ports, no jaeger ui",
			input: v1alpha1.TempoMonolithicSpec{},
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "tempo-sample",
						Namespace:   "default",
						Labels:      ComponentLabels("tempo", "sample"),
						Annotations: map[string]string{},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:       "http",
							Protocol:   corev1.ProtocolTCP,
							Port:       3200,
							TargetPort: intstr.FromString("http"),
						}},
						Selector: ComponentLabels("tempo", "sample"),
					},
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
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "tempo-sample",
						Namespace:   "default",
						Labels:      ComponentLabels("tempo", "sample"),
						Annotations: map[string]string{},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
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
						Selector: ComponentLabels("tempo", "sample"),
					},
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
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "tempo-sample",
						Namespace:   "default",
						Labels:      ComponentLabels("tempo", "sample"),
						Annotations: map[string]string{},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
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
						Selector: ComponentLabels("tempo", "sample"),
					},
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
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "tempo-sample",
						Namespace:   "default",
						Labels:      ComponentLabels("tempo", "sample"),
						Annotations: map[string]string{},
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
						Selector: ComponentLabels("tempo", "sample"),
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-sample-jaegerui",
						Namespace: "default",
						Labels:    ComponentLabels("jaegerui", "sample"),
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "jaeger-grpc",
								Protocol:   corev1.ProtocolTCP,
								Port:       16685,
								TargetPort: intstr.FromString("jaeger-grpc"),
							},
							{
								Name:       "jaeger-ui",
								Protocol:   corev1.ProtocolTCP,
								Port:       16686,
								TargetPort: intstr.FromString("jaeger-ui"),
							},
							{
								Name:       "jaeger-metrics",
								Protocol:   corev1.ProtocolTCP,
								Port:       16687,
								TargetPort: intstr.FromString("jaeger-metrics"),
							},
						},
						Selector: ComponentLabels("tempo", "sample"),
					},
				},
			},
		},
		{
			name: "enable gateway, OTLP/gRPC, and JaegerUI",
			input: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
				JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
					Enabled: true,
				},
				Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
					Enabled: true,
					TenantsSpec: v1alpha1.TenantsSpec{
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "dev",
								TenantID:   "dev",
							},
						},
					},
				},
			},
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-sample-gateway",
						Namespace: "default",
						Labels:    ComponentLabels("gateway", "sample"),
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "tempo-sample-gateway-serving-cert",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "public",
								Protocol:   corev1.ProtocolTCP,
								Port:       8080,
								TargetPort: intstr.FromString("public"),
							},
							{
								Name:       "internal",
								Protocol:   corev1.ProtocolTCP,
								Port:       8081,
								TargetPort: intstr.FromString("internal"),
							},
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
								TargetPort: intstr.FromString("grpc-public"),
							},
						},
						Selector: ComponentLabels("tempo", "sample"),
					},
				},
			},
		},
		{
			name: "add service cert annotation",
			input: v1alpha1.TempoMonolithicSpec{
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			addServiceCertAnnotations: true,
			expected: []client.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apps/v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-sample",
						Namespace: "default",
						Labels:    ComponentLabels("tempo", "sample"),
						Annotations: map[string]string{
							"service.beta.openshift.io/serving-cert-secret-name": "tempo-sample-serving-cert",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
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
						Selector: ComponentLabels("tempo", "sample"),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec = test.input
			opts.useServiceCertsOnReceiver = test.addServiceCertAnnotations
			svcs := BuildServices(opts)
			require.Equal(t, test.expected, svcs)
		})
	}
}
