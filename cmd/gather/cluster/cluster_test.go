package cluster

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/cmd/gather/config"
)

func TestGetOwnerResources(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, monitoringv1.AddToScheme(scheme))
	owner := &tempov1alpha1.TempoStack{
		TypeMeta:   metav1.TypeMeta{Kind: "TempoStack"},
		ObjectMeta: metav1.ObjectMeta{UID: "stack-uid"},
	}

	for _, tt := range []struct {
		name   string
		list   client.ObjectList
		object client.Object
	}{
		{name: "services", list: &corev1.ServiceList{}, object: &corev1.Service{}},
		{name: "service monitors", list: &monitoringv1.ServiceMonitorList{}, object: &monitoringv1.ServiceMonitor{}},
		{name: "pod monitors", list: &monitoringv1.PodMonitorList{}, object: &monitoringv1.PodMonitor{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			owned := tt.object.DeepCopyObject().(client.Object)
			owned.SetName("owned")
			owned.SetNamespace("test")
			owned.SetLabels(map[string]string{"app.kubernetes.io/managed-by": "tempo-operator"})
			owned.SetOwnerReferences([]metav1.OwnerReference{{Kind: owner.Kind, UID: owner.UID}})
			other := owned.DeepCopyObject().(client.Object)
			other.SetName("other-owner")
			other.SetOwnerReferences([]metav1.OwnerReference{{Kind: owner.Kind, UID: "other-uid"}})
			unmanaged := owned.DeepCopyObject().(client.Object)
			unmanaged.SetName("unmanaged")
			unmanaged.SetLabels(nil)
			kubernetesClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(owned, other, unmanaged).Build()
			c := NewCluster(&config.Config{KubernetesClient: kubernetesClient})
			resources, err := c.getOwnerResources(tt.list.DeepCopyObject().(client.ObjectList), owner)
			require.NoError(t, err)
			require.Len(t, resources, 1)
			require.Equal(t, "owned", resources[0].GetName())
		})
	}
}
