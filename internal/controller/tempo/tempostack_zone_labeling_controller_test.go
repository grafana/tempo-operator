package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func zoneAwarePod(nodeName string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "tempo-test-ingester-0",
			Namespace:   "default",
			Labels:      map[string]string{v1alpha1.LabelZoneAwarePod: "enabled"},
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{NodeName: nodeName},
	}
}

func node(name string, labels map[string]string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func zoneReconciler(objs ...client.Object) *TempoStackZoneAwarePodReconciler {
	return &TempoStackZoneAwarePodReconciler{
		Client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build(),
		Log:    ctrl.Log.WithName("test"),
	}
}

func reconcilePod(t *testing.T, r *TempoStackZoneAwarePodReconciler) (*corev1.Pod, error) {
	t.Helper()

	nsn := types.NamespacedName{Name: "tempo-test-ingester-0", Namespace: "default"}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{}
	require.NoError(t, r.Get(context.Background(), nsn, pod))
	return pod, nil
}

func TestZoneAwarePodAnnotatedWithAvailabilityZone(t *testing.T) {
	r := zoneReconciler(
		zoneAwarePod("node-1", map[string]string{
			v1alpha1.AnnotationAvailabilityZoneLabels: "topology.kubernetes.io/zone",
		}),
		node("node-1", map[string]string{"topology.kubernetes.io/zone": "eu-west-1a"}),
	)

	pod, err := reconcilePod(t, r)
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1a", pod.Annotations[v1alpha1.AnnotationAvailabilityZone])
}

func TestZoneAwarePodMultipleTopologyKeysAreJoined(t *testing.T) {
	r := zoneReconciler(
		zoneAwarePod("node-1", map[string]string{
			v1alpha1.AnnotationAvailabilityZoneLabels: "topology.kubernetes.io/region,topology.kubernetes.io/zone",
		}),
		node("node-1", map[string]string{
			"topology.kubernetes.io/region": "eu-west-1",
			"topology.kubernetes.io/zone":   "eu-west-1a",
		}),
	)

	pod, err := reconcilePod(t, r)
	require.NoError(t, err)
	// the values are joined in the order the topology keys were configured in the CR
	assert.Equal(t, "eu-west-1_eu-west-1a", pod.Annotations[v1alpha1.AnnotationAvailabilityZone])
}

func TestZoneAwarePodNotScheduledYet(t *testing.T) {
	r := zoneReconciler(
		zoneAwarePod("", map[string]string{
			v1alpha1.AnnotationAvailabilityZoneLabels: "topology.kubernetes.io/zone",
		}),
	)

	// a later event carries the node name, therefore this must not be an error
	pod, err := reconcilePod(t, r)
	require.NoError(t, err)
	assert.NotContains(t, pod.Annotations, v1alpha1.AnnotationAvailabilityZone)
}

func TestZoneAwarePodOnNodeWithoutTopologyLabel(t *testing.T) {
	r := zoneReconciler(
		zoneAwarePod("node-1", map[string]string{
			v1alpha1.AnnotationAvailabilityZoneLabels: "topology.kubernetes.io/zone",
		}),
		node("node-1", map[string]string{}),
	)

	_, err := reconcilePod(t, r)
	require.ErrorContains(t, err, "scheduled node is missing the topology.kubernetes.io/zone label")
}

func TestZoneAwarePodWithoutTopologyKeysAnnotation(t *testing.T) {
	r := zoneReconciler(
		zoneAwarePod("node-1", nil),
		node("node-1", map[string]string{"topology.kubernetes.io/zone": "eu-west-1a"}),
	)

	_, err := reconcilePod(t, r)
	require.ErrorContains(t, err, "is missing the tempo.grafana.com/availability-zone-labels annotation")
}

func TestZoneAwarePodDeleted(t *testing.T) {
	r := zoneReconciler()

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "tempo-test-ingester-0", Namespace: "default"},
	})
	require.NoError(t, err)
}

func TestEventPodHasLabel(t *testing.T) {
	assert.True(t, eventPodHasLabel(zoneAwarePod("node-1", nil)))

	pod := zoneAwarePod("node-1", nil)
	pod.Labels = nil
	assert.False(t, eventPodHasLabel(pod))

	assert.False(t, eventPodHasLabel(node("node-1", nil)))
}
