package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
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

type mockK8SVersionRoundTripper struct{}

func (m *mockK8SVersionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	info := version.Info{
		Major:        "1",
		Minor:        "22",
		GitVersion:   "v1.22.0",
		GitCommit:    "abc123",
		GitTreeState: "clean",
		BuildDate:    "2025-07-24T00:00:00Z",
		GoVersion:    "go1.25.9",
		Compiler:     "gc",
		Platform:     "linux/amd64",
	}

	infoJson, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBuffer(infoJson)),
	}
	return resp, nil
}

func TestReconcileOperator(t *testing.T) {
	nsn := types.NamespacedName{Name: "reconcile-operator", Namespace: "default"}
	createOperatorDeployment(t, nsn)

	reconciler := OperatorReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		Config: &rest.Config{
			Transport: &mockK8SVersionRoundTripper{},
		},
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

// TestSetupOperatorControllerWithManager verifies that the controller watching the EndpointSlice
// of the Kubernetes API server can be registered. The controller has no For() object, which the
// controller-runtime builder only rejects at runtime.
func TestSetupOperatorControllerWithManager(t *testing.T) {
	mgr, err := ctrl.NewManager(testCfg, ctrl.Options{
		Scheme:  testScheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	require.NoError(t, err)

	reconciler := &OperatorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: mgr.GetConfig(),
	}
	require.NoError(t, reconciler.SetupWithManager(mgr, configv1alpha1.ProjectConfig{}))
}
