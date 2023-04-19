package servicemonitor

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/certrotation"
	"github.com/os-observability/tempo-operator/internal/manifests/gateway"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

// BuildServiceMonitors creates ServiceMonitor objects.
func BuildServiceMonitors(params manifestutils.Params) []client.Object {
	// Create one ServiceMonitor instance per monitored service.
	// Each tempo component has its own TLS certificate, therefore we need separate
	// ServiceMonitor instances for each component.
	monitors := []client.Object{
		buildTempoComponentServiceMonitor(params, manifestutils.CompactorComponentName),
		buildTempoComponentServiceMonitor(params, manifestutils.DistributorComponentName),
		buildTempoComponentServiceMonitor(params, manifestutils.IngesterComponentName),
		buildTempoComponentServiceMonitor(params, manifestutils.QuerierComponentName),
		buildTempoComponentServiceMonitor(params, manifestutils.QueryFrontendComponentName),
	}

	if params.Tempo.Spec.Template.Gateway.Enabled {
		monitors = append(monitors, buildGatewayServiceMonitor(params.Tempo))
	}

	return monitors
}

func buildTempoComponentServiceMonitor(params manifestutils.Params, component string) *monitoringv1.ServiceMonitor {
	tempo := params.Tempo
	scheme := "http"
	var tlsConfig *monitoringv1.TLSConfig

	if params.Gates.HTTPEncryption {
		scheme = "https"
		serverName := naming.ServiceFqdn(tempo.Namespace, tempo.Name, component)

		tlsConfig = &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				CA: monitoringv1.SecretOrConfigMap{
					ConfigMap: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: naming.SigningCABundleName(tempo.Name),
						},
						Key: certrotation.CAFile,
					},
				},
				Cert: monitoringv1.SecretOrConfigMap{
					Secret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: naming.Name(tempo.Name, component),
						},
						Key: corev1.TLSCertKey,
					},
				},
				KeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: naming.Name(tempo.Name, component),
					},
					Key: corev1.TLSPrivateKeyKey,
				},
				// E.g. tempo-simplest-compactor-http.tempo-operator.svc.cluster.local
				ServerName: serverName,
			},
		}
	}

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tempo.Namespace,
			Name:      naming.Name(component, tempo.Name),
			Labels:    manifestutils.CommonLabels(tempo.Name),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme:    scheme,
				Port:      manifestutils.HttpPortName,
				Path:      "/metrics",
				TLSConfig: tlsConfig,
				// Custom relabel configs to be compatible with predefined Tempo dashboards:
				// https://grafana.com/docs/tempo/latest/operations/monitoring/#dashboards
				RelabelConfigs: []*monitoringv1.RelabelConfig{
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_label_app_kubernetes_io_instance"},
						TargetLabel:  "cluster",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_namespace", "__meta_kubernetes_service_label_app_kubernetes_io_component"},
						Separator:    "/",
						TargetLabel:  "job",
					},
				},
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{tempo.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: manifestutils.ComponentLabels(component, tempo.Name),
			},
		},
	}
}

func buildGatewayServiceMonitor(tempo v1alpha1.TempoStack) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tempo.Namespace,
			Name:      naming.Name(manifestutils.GatewayComponentName, tempo.Name),
			Labels:    manifestutils.CommonLabels(tempo.Name),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port: gateway.InternalPortName,
				// Custom relabel configs to be compatible with predefined Tempo dashboards:
				// https://grafana.com/docs/tempo/latest/operations/monitoring/#dashboards
				RelabelConfigs: []*monitoringv1.RelabelConfig{
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_label_app_kubernetes_io_instance"},
						TargetLabel:  "cluster",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_namespace", "__meta_kubernetes_service_label_app_kubernetes_io_component"},
						Separator:    "/",
						TargetLabel:  "job",
					},
				},
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{tempo.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: manifestutils.ComponentLabels(manifestutils.GatewayComponentName, tempo.Name),
			},
		},
	}
}
