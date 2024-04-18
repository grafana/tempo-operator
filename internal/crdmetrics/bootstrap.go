package crdmetrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Bootstrap configures the OpenTelemetry meter provider with the Prometheus exporter.
func Bootstrap(client client.Client) error {
	exporter, err := prometheus.New(prometheus.WithRegisterer(metrics.Registry))
	if err != nil {
		return err
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	// Create metrics
	tempoStackMetrics := newTempoStackMetrics(client)
	err = tempoStackMetrics.Setup()
	return err
}
