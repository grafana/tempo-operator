package webhooks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func TestMonolithicDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    *v1alpha1.TempoMonolithic
		expected *v1alpha1.TempoMonolithic
	}{
		{
			name: "empty spec, set memory backend and enable OTLP/gRPC and OTLP/HTTP",
			input: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{},
			},
			expected: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Storage: &v1alpha1.MonolithicStorageSpec{
						Traces: v1alpha1.MonolithicTracesStorageSpec{
							Backend: "memory",
						},
					},
					Ingestion: &v1alpha1.MonolithicIngestionSpec{
						OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
							GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
				},
			},
		},
		{
			name: "set default values for PV",
			input: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Storage: &v1alpha1.MonolithicStorageSpec{
						Traces: v1alpha1.MonolithicTracesStorageSpec{
							Backend: "pv",
						},
					},
				},
			},
			expected: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Storage: &v1alpha1.MonolithicStorageSpec{
						Traces: v1alpha1.MonolithicTracesStorageSpec{
							Backend: "pv",
							WAL: &v1alpha1.MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
							PV: &v1alpha1.MonolithicTracesStoragePVSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &v1alpha1.MonolithicIngestionSpec{
						OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
							GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
							HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
				},
			},
		},
		{
			name: "do not change already set values",
			input: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Storage: &v1alpha1.MonolithicStorageSpec{
						Traces: v1alpha1.MonolithicTracesStorageSpec{
							Backend: "s3",
							WAL: &v1alpha1.MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &v1alpha1.MonolithicIngestionSpec{
						OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
							// GRPC is explicitly disabled and should not be enabled by webhook
							GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: false,
							},
							HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			expected: &v1alpha1.TempoMonolithic{
				Spec: v1alpha1.TempoMonolithicSpec{
					Storage: &v1alpha1.MonolithicStorageSpec{
						Traces: v1alpha1.MonolithicTracesStorageSpec{
							Backend: "s3",
							WAL: &v1alpha1.MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &v1alpha1.MonolithicIngestionSpec{
						OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
							GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: false,
							},
							HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Default()
			assert.Equal(t, test.expected, test.input)
		})
	}
}
