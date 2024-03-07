package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	twentyGBQuantity = resource.MustParse("20Gi")
)

func TestMonolithicDefault(t *testing.T) {
	tests := []struct {
		name     string
		input    *TempoMonolithic
		expected *TempoMonolithic
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
