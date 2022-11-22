package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestReconcile(t *testing.T) {
	storageSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		StringData: map[string]string{
			"endpoint":          "http://minio:9000",
			"bucket":            "tempo",
			"access_key_id":     "tempo-user",
			"access_key_secret": "abcd1234",
		},
	}
	err := k8sClient.Create(context.Background(), storageSecret)
	require.NoError(t, err)

	nsn := types.NamespacedName{Name: "test", Namespace: "default"}
	tempo := &v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: v1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{},
			},
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: storageSecret.Name,
			},
		},
	}
	err = k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := MicroservicesReconciler{
		Client: k8sClient,
		Scheme: testScheme,
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	// Check if objects of specific types were created and are managed by the operator
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   nsn.Name,
			"app.kubernetes.io/managed-by": "tempo-controller",
		}),
	}
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.StatefulSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
}
