package crdmetrics

import (
	"context"
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

type tempoStackMetrics struct {
	client       client.Client
	observations []instancesView
}

func instanceMetricName(name string) string {
	return fmt.Sprintf("%s_%s", tempoStackMetricsPrefix, name)
}

func newTempoStackMetrics(client client.Client) *tempoStackMetrics {
	return &tempoStackMetrics{
		client: client,
	}
}

func (i *tempoStackMetrics) Setup() error {
	meter := otel.Meter(meterName)

	obs, err := newObservation(meter,
		storageBackendMetric,
		"Number of instances per storage type",
		"type",
		func(instance client.Object) (string, bool) {
			tempoStack := instance.(*v1alpha1.TempoStack)
			return string(tempoStack.Spec.Storage.Secret.Type), true
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		managedMetric,
		"Instances managed by the operator",
		"state",
		func(instance client.Object) (string, bool) {
			tempoStack := instance.(*v1alpha1.TempoStack)
			return string(tempoStack.Spec.ManagementState), true
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		jaegerUIUsage,
		"Instances with jaeger UI enabled/disabled",
		"enabled",
		func(instance client.Object) (string, bool) {
			tempoStack := instance.(*v1alpha1.TempoStack)
			return strconv.FormatBool(tempoStack.Spec.Template.QueryFrontend.JaegerQuery.Enabled), true
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	obs, err = newObservation(meter,
		multitenancy,
		"Instances with multi-tenancy mode static/openshift/disabled",
		"type",
		func(instance client.Object) (string, bool) {
			tempoStack := instance.(*v1alpha1.TempoStack)
			if tempoStack.Spec.Tenants != nil && tempoStack.Spec.Tenants.Mode != "" {
				return string(tempoStack.Spec.Tenants.Mode), true
			}
			return "disabled", true
		})
	if err != nil {
		return err
	}
	i.observations = append(i.observations, obs)

	instruments := make([]metric.Observable, 0, len(i.observations))
	for _, o := range i.observations {
		instruments = append(instruments, o.Gauge)
	}
	_, err = meter.RegisterCallback(i.callback, instruments...)
	return err
}

func (i *tempoStackMetrics) callback(ctx context.Context, observer metric.Observer) error {
	instances := &v1alpha1.TempoStackList{}
	if err := i.client.List(ctx, instances); err == nil {

		// Reset observations
		for _, o := range i.observations {
			o.reset()
		}

		for k := range instances.Items {
			tempoStack := instances.Items[k]
			for _, o := range i.observations {
				o.Record(&tempoStack)
			}
		}
	}

	// Report metrics
	for _, o := range i.observations {
		o.Report(observer)
	}

	return nil
}
