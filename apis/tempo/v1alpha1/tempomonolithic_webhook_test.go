package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMonolithicDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    *TempoMonolithic
		expected *TempoMonolithic
	}{
		{
			name: "empty spec, set memory backend and enable OTLP/gRPC",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "memory",
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
						},
					},
				},
			},
		},
		{
			name: "set default values for PV",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "pv",
						},
					},
				},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "pv",
							WAL: &MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
							PV: &MonolithicTracesStoragePVSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							GRPC: &MonolithicIngestionOTLPProtocolsGRPCSpec{
								Enabled: true,
							},
						},
					},
				},
			},
		},
		{
			name: "do not change already set values",
			input: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							WAL: &MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							// HTTP is already set, GRPC should not be enabled by webhook
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			expected: &TempoMonolithic{
				Spec: TempoMonolithicSpec{
					Storage: MonolithicStorageSpec{
						Traces: MonolithicTracesStorageSpec{
							Backend: "s3",
							WAL: &MonolithicTracesStorageWALSpec{
								Size: tenGBQuantity,
							},
						},
					},
					Ingestion: &MonolithicIngestionSpec{
						OTLP: &MonolithicIngestionOTLPSpec{
							HTTP: &MonolithicIngestionOTLPProtocolsHTTPSpec{
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
