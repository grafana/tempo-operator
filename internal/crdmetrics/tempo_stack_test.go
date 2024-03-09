package crdmetrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

const (
	AgentSideCar   = "Sidecar"
	AgentDaemonSet = "Daemonset"
)

type expectedMetric struct {
	name   string
	labels []attribute.KeyValue
	value  int64
}

func assertLabelAndValues(t *testing.T, name string, metrics metricdata.ResourceMetrics, expectedAttrs []attribute.KeyValue, expectedValue int64) {
	var matchingMetric metricdata.Metrics
	found := false
	for _, sm := range metrics.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				matchingMetric = m
				found = true
				break
			}
		}
	}

	assert.True(t, found, "Metric %s not found", name)

	gauge, ok := matchingMetric.Data.(metricdata.Gauge[int64])
	assert.True(t, ok,
		"Metric %s doesn't have expected type %T, got %T", metricdata.Gauge[int64]{}, matchingMetric.Data,
	)

	expectedAttrSet := attribute.NewSet(expectedAttrs...)
	var matchingDP metricdata.DataPoint[int64]
	found = false
	for _, dp := range gauge.DataPoints {
		if expectedAttrSet.Equals(&dp.Attributes) {
			matchingDP = dp
			found = true
			break
		}
	}

	assert.True(t, found, "Metric %s doesn't have expected attributes %v", expectedAttrs)
	assert.Equal(t, expectedValue, matchingDP.Value,
		"Metric %s doesn't have expected value %d, got %d", name, expectedValue, matchingDP.Value)
}

func newTempoStackInstance(nsn types.NamespacedName, managedState v1alpha1.ManagementStateType,
	storage v1alpha1.ObjectStorageSecretType) v1alpha1.TempoStack {
	return v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			ManagementState: managedState,
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Type: storage,
				},
			},
		},
	}
}

func newExpectedMetric(name string, keyPair attribute.KeyValue, value int64) expectedMetric {
	return expectedMetric{
		name: instanceMetricName(name),
		labels: []attribute.KeyValue{
			keyPair,
		},
		value: value,
	}
}

func TestValueObservedMetrics(t *testing.T) {
	s := scheme.Scheme

	// Add jaeger to schema
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.TempoStack{}, &v1alpha1.TempoStackList{})

	// Create tempo instances
	tempoGCS := newTempoStackInstance(types.NamespacedName{
		Name:      "my-tempo-gcs",
		Namespace: "test",
	}, v1alpha1.ManagementStateManaged, v1alpha1.ObjectStorageSecretGCS)

	tempoS3 := newTempoStackInstance(types.NamespacedName{
		Name:      "my-jaeger-s3",
		Namespace: "test",
	}, v1alpha1.ManagementStateManaged, v1alpha1.ObjectStorageSecretS3)

	tempoOtherS3 := newTempoStackInstance(types.NamespacedName{
		Name:      "my-jaeger-other-s3",
		Namespace: "test",
	}, v1alpha1.ManagementStateManaged, v1alpha1.ObjectStorageSecretS3)

	tempoUnmanaged := newTempoStackInstance(types.NamespacedName{
		Name:      "my-jaeger-azure-unmanaged",
		Namespace: "test",
	}, v1alpha1.ManagementStateUnmanaged, v1alpha1.ObjectStorageSecretAzure)

	tempoUnmanaged2 := newTempoStackInstance(types.NamespacedName{
		Name:      "my-jaeger-azure-unmanaged-2",
		Namespace: "test",
	}, v1alpha1.ManagementStateUnmanaged, v1alpha1.ObjectStorageSecretAzure)

	objs := []runtime.Object{
		&tempoGCS,
		&tempoS3,
		&tempoOtherS3,
		&tempoUnmanaged,
		&tempoUnmanaged2,
	}
	expected := []expectedMetric{
		newExpectedMetric(managedMetric, attribute.String("state", string(v1alpha1.ManagementStateManaged)), 3),
		newExpectedMetric(managedMetric, attribute.String("state", string(v1alpha1.ManagementStateUnmanaged)), 2),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretGCS)), 1),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretS3)), 2),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretAzure)), 2),
	}

	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	instancesObservedValue := newTempoStackMetrics(cl)
	err := instancesObservedValue.Setup()
	require.NoError(t, err)

	metrics := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &metrics)
	require.NoError(t, err)

	for _, e := range expected {
		assertLabelAndValues(t, e.name, metrics, e.labels, e.value)
	}

	// Test deleting GCS storage
	err = cl.Delete(context.Background(), &tempoGCS)
	require.NoError(t, err)

	// Reset measurement batches
	err = provider.ForceFlush(context.Background())
	require.NoError(t, err)
	metrics = metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &metrics)
	require.NoError(t, err)

	// Set new numbers
	expected = []expectedMetric{
		newExpectedMetric(managedMetric, attribute.String("state", string(v1alpha1.ManagementStateManaged)), 2),
		newExpectedMetric(managedMetric, attribute.String("state", string(v1alpha1.ManagementStateUnmanaged)), 2),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretGCS)), 0),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretS3)), 2),
		newExpectedMetric(storageBackendMetric, attribute.String("type", string(v1alpha1.ObjectStorageSecretAzure)), 2),
	}
	for _, e := range expected {
		assertLabelAndValues(t, e.name, metrics, e.labels, e.value)
	}
}
