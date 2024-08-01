package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
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
					Query:      &QuerySpec{},
					Management: "Managed",
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
					Query:      &QuerySpec{},
					Management: "Managed",
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
					Query:      &QuerySpec{},
					Management: "Unmanaged",
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
					Query:      &QuerySpec{},
					Management: "Unmanaged",
				},
			},
		},
		{
			name: "enable oauth when feature gate is enabled",
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
				ObjectMeta: v1.ObjectMeta{
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
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
						ServicesQueryDuration: &defaultServicesDuration,
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
					},
					Management: "Managed",
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
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
						ServicesQueryDuration: &defaultServicesDuration,
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
					Management: "Managed",
				},
			},
		},
		{
			name: "no touch oauth when feature gate is disabled (true case)",
			input: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
						ServicesQueryDuration: &defaultServicesDuration,
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: true,
							SAR:     "{\"namespace\": \"testns\", \"resource\": \"pods\", \"verb\": \"get\"}",
						},
					},
					Management: "Managed",
				},
			},
		},
		{
			name: "no touch jaeger ui oauth when feature gate is disabled (false case)",
			input: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
						ServicesQueryDuration: &defaultServicesDuration,
					},
					Query: &QuerySpec{
						Authentication: &OAuthAuthenticationSpec{
							Enabled: false,
						},
					},
					Management: "Managed",
				},
			},
		},
		{
			name: "define custom duration for services list",
			input: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						ServicesQueryDuration: &v1.Duration{Duration: time.Duration(100 * 100)},
					},
				},
			},
			expected: &TempoMonolithic{
				ObjectMeta: v1.ObjectMeta{
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
						ServicesQueryDuration: &v1.Duration{Duration: time.Duration(100 * 100)},
					},
					Query:      &QuerySpec{},
					Management: "Managed",
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
