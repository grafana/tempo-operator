package monolithic

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
)

func TestBuildServiceMonitor(t *testing.T) {
	opts := Options{
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{},
		},
	}
	sm := BuildServiceMonitor(opts)

	labels := ComponentLabels("tempo", "sample")
	require.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme: "http",
				Port:   "http",
				Path:   "/metrics",
				RelabelConfigs: []*monitoringv1.RelabelConfig{
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_label_app_kubernetes_io_instance"},
						TargetLabel:  "cluster",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_namespace", "__meta_kubernetes_service_label_app_kubernetes_io_component"},
						Separator:    ptr.To("/"),
						TargetLabel:  "job",
					},
				},
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"default"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, sm)
}

func TestBuildServiceMonitorGateway(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
					Enabled: true,
					TenantsSpec: v1alpha1.TenantsSpec{
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "dev",
								TenantID:   "dev",
							},
						},
					},
				},
			},
		},
	}
	sm := BuildServiceMonitor(opts)

	labels := ComponentLabels("gateway", "sample")
	require.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample-gateway",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme: "http",
				Port:   "internal",
				Path:   "/metrics",
				RelabelConfigs: []*monitoringv1.RelabelConfig{
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_label_app_kubernetes_io_instance"},
						TargetLabel:  "cluster",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_namespace", "__meta_kubernetes_service_label_app_kubernetes_io_component"},
						Separator:    ptr.To("/"),
						TargetLabel:  "job",
					},
				},
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"default"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, sm)
}
