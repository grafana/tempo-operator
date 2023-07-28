package controllers

import (
	"context"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func createOperatorDeployment(t *testing.T, nsn types.NamespacedName) {
	labels := map[string]string{
		"app.kubernetes.io/name": "tempo-operator",
	}

	tempo := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "operator",
							Image: "operator",
						},
					},
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)
}

func TestReconcileOperator(t *testing.T) {
	nsn := types.NamespacedName{Name: "reconcile-operator", Namespace: "default"}
	createOperatorDeployment(t, nsn)

	reconciler := OperatorReconciler{
		Client: k8sClient,
	}
	err := reconciler.Reconcile(context.Background(), configv1alpha1.ProjectConfig{
		Gates: configv1alpha1.FeatureGates{
			PrometheusOperator: true,
			Observability: configv1alpha1.ObservabilityFeatureGates{
				Metrics: configv1alpha1.MetricsFeatureGates{
					CreateServiceMonitors: true,
					CreatePrometheusRules: true,
				},
			},
		},
	})
	require.NoError(t, err)

	// Check if objects of specific types were created and are managed by the operator
	listOpts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(manifestutils.CommonOperatorLabels()),
	}
	{
		list := &monitoringv1.ServiceMonitorList{}
		err = k8sClient.List(context.Background(), list, listOpts...)
		assert.NoError(t, err)
		assert.Len(t, list.Items, 1)
	}
	{
		list := &monitoringv1.PrometheusRuleList{}
		err = k8sClient.List(context.Background(), list, listOpts...)
		assert.NoError(t, err)
		assert.Len(t, list.Items, 1)
	}

	// Update config and reconcile again
	err = reconciler.Reconcile(context.Background(), configv1alpha1.ProjectConfig{
		Gates: configv1alpha1.FeatureGates{
			PrometheusOperator: true,
			Observability: configv1alpha1.ObservabilityFeatureGates{
				Metrics: configv1alpha1.MetricsFeatureGates{
					CreateServiceMonitors: false,
					CreatePrometheusRules: false,
				},
			},
		},
	})
	require.NoError(t, err)

	// Check if objects got pruned by the operator
	{
		list := &monitoringv1.ServiceMonitorList{}
		err = k8sClient.List(context.Background(), list, listOpts...)
		assert.NoError(t, err)
		assert.Len(t, list.Items, 0)
	}
	{
		list := &monitoringv1.PrometheusRuleList{}
		err = k8sClient.List(context.Background(), list, listOpts...)
		assert.NoError(t, err)
		assert.Len(t, list.Items, 0)
	}
}
