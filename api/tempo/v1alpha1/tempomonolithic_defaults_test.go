package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

var (
	twentyGBQuantity = resource.MustParse("20Gi")
)

func TestMonolithicDefault(t *testing.T) {
	tests := []struct {
		name       string
		ctrlConfig configv1alpha1.ProjectConfig
		input      *TempoMonolithic
		expected   *TempoMonolithic
	}{
		{
			name: "empty spec, set memory backend and enable OTLP/gRPC and OTLP/HTTP",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "memory",
							Size:    &twoGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "pv backend, set 10Gi default pv size",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "pv",
						},
					},
				},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "pv",
							Size:    &tenGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "do not change already set values",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							// GRPC is explicitly disabled and should not be enabled by webhook
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: false,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					Management: "Unmanaged",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: false,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					Management: "Unmanaged",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "enable jaeger ui oauth when feature gate is enabled",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OauthProxy: configv1alpha1.OauthProxyFeatureGates{
							DefaultEnabled: true,
						},
					},
				},
			},
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled:     true,
							Termination: TLSRouteTerminationTypeEdge,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   true,
							SAR:       "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
							Resources: &corev1.ResourceRequirements{},
						},
						ServicesQueryDuration:        &defaultServicesDuration,
						FindTracesConcurrentRequests: 2,
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "no touch jaeger ui oauth when feature gate is enabled and user specified false value explicit",
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OauthProxy: configv1alpha1.OauthProxyFeatureGates{
							DefaultEnabled: true,
						},
					},
				},
			},
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled: false,
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled:     true,
							Termination: TLSRouteTerminationTypeEdge,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   false,
							SAR:       "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
							Resources: &corev1.ResourceRequirements{},
						},
						ServicesQueryDuration:        &defaultServicesDuration,
						FindTracesConcurrentRequests: 2,
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "no touch jaeger ui oauth when feature gate is disabled (true case)",
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled:     true,
							Termination: TLSRouteTerminationTypeEdge,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   true,
							SAR:       "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
							Resources: &corev1.ResourceRequirements{},
						},
						ServicesQueryDuration:        &defaultServicesDuration,
						FindTracesConcurrentRequests: 2,
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "no touch jaeger ui oauth when feature gate is disabled (false case)",
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   false,
							Resources: &corev1.ResourceRequirements{},
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled:     true,
							Termination: TLSRouteTerminationTypeEdge,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   false,
							SAR:       "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
							Resources: &corev1.ResourceRequirements{},
						},
						ServicesQueryDuration:        &defaultServicesDuration,
						FindTracesConcurrentRequests: 2,
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "define custom duration for services list, timeout and find traces",
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled: true,
						},
						ServicesQueryDuration:        &metav1.Duration{Duration: time.Duration(100 * 100)},
						FindTracesConcurrentRequests: 40,
					},
					Timeout: metav1.Duration{Duration: time.Hour},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							Size:    &twentyGBQuantity,
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					JaegerUI: &MonolithicJaegerUISpec{
						Enabled: true,
						Route: &MonolithicJaegerUIRouteSpec{
							Enabled:     true,
							Termination: TLSRouteTerminationTypeEdge,
						},
						Authentication: &JaegerQueryAuthenticationSpec{
							Enabled:   false,
							SAR:       "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
							Resources: &corev1.ResourceRequirements{},
						},
						ServicesQueryDuration:        &metav1.Duration{Duration: time.Duration(100 * 100)},
						FindTracesConcurrentRequests: 40,
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Hour},
					Query:      &MonolithicQuerySpec{},
				},
			},
		},
		{
			name: "query defined",
			input: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "memory",
							Size:    &twoGBQuantity,
						},
					},
					Query: &MonolithicQuerySpec{
						RBAC: RBACSpec{
							Enabled: true,
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Spec: TempoMonolithicSpec{
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
					Storage: &MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "memory",
							Size:    &twoGBQuantity,
						},
					},
					Management: "Managed",
					Timeout:    metav1.Duration{Duration: time.Second * 30},
					Query: &MonolithicQuerySpec{
						RBAC: RBACSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Default(test.ctrlConfig)
			assert.Equal(t, test.expected, test.input)
		})
	}
}
