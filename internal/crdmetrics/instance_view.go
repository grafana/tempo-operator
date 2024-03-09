package crdmetrics

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type countFn func(instance client.Object) (string, bool)

// This structure contains the labels associated with the instances and a counter of the number of instances.
type instancesView struct {
	Name  string
	Label string
	Count map[string]int
	Gauge metric.Int64ObservableGauge
	KeyFn countFn
}

func (i *instancesView) reset() {
	for k := range i.Count {
		i.Count[k] = 0
	}
}

func (i *instancesView) Record(instance client.Object) {
	label, counted := i.KeyFn(instance)
	if counted {
		i.Count[label]++
	}
}

func (i *instancesView) Report(observer metric.Observer) {
	for key, count := range i.Count {
		opt := metric.WithAttributes(
			attribute.Key(i.Label).String(key),
		)
		observer.ObserveInt64(i.Gauge, int64(count), opt)
	}
}

func newObservation(meter metric.Meter, name, desc, label string, keyFn countFn) (instancesView, error) {
	observation := instancesView{
		Name:  name,
		Count: make(map[string]int),
		KeyFn: keyFn,
		Label: label,
	}

	g, err := meter.Int64ObservableGauge(instanceMetricName(name), metric.WithDescription(desc))
	if err != nil {
		return instancesView{}, err
	}

	observation.Gauge = g
	return observation, nil
}
