package servicemonitor

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/internal/certrotation"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

// BuildServiceMonitors creates ServiceMonitor objects.
func BuildServiceMonitors(params manifestutils.Params) []client.Object {
	// Create one ServiceMonitor instance per monitored service.
	// Each tempo component has its own TLS certificate, therefore we need separate
	// ServiceMonitor instances for each component.
	monitors := []client.Object{
		buildServiceMonitor(params, manifestutils.CompactorComponentName, manifestutils.HttpPortName),
		buildServiceMonitor(params, manifestutils.DistributorComponentName, manifestutils.HttpPortName),
		buildServiceMonitor(params, manifestutils.IngesterComponentName, manifestutils.HttpPortName),
		buildServiceMonitor(params, manifestutils.QuerierComponentName, manifestutils.HttpPortName),
		buildServiceMonitor(params, manifestutils.QueryFrontendComponentName, manifestutils.HttpPortName),
	}

	if params.Tempo.Spec.Template.Gateway.Enabled {
		monitors = append(monitors, buildServiceMonitor(params, manifestutils.GatewayComponentName, manifestutils.GatewayInternalHttpPortName))
	}

	return monitors
}

func buildServiceMonitor(params manifestutils.Params, component string, port string) *monitoringv1.ServiceMonitor {
	labels := manifestutils.ComponentLabels(component, params.Tempo.Name)
	return NewServiceMonitor(params.Tempo.Namespace, params.Tempo.Name, labels, params.CtrlConfig.Gates.HTTPEncryption, component, port)
}

// NewServiceMonitor creates a ServiceMonitor.
func NewServiceMonitor(
	namespace string,
	name string,
	labels labels.Set,
	tls bool,
	component string,
	port string,
) *monitoringv1.ServiceMonitor {
	scheme := "http"
	var tlsConfig *monitoringv1.TLSConfig

	if tls {
		scheme = "https"
		serverName := naming.ServiceFqdn(namespace, name, component)

		tlsConfig = &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				CA: monitoringv1.SecretOrConfigMap{
					ConfigMap: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: naming.SigningCABundleName(name),
						},
						Key: certrotation.CAFile,
					},
				},
				Cert: monitoringv1.SecretOrConfigMap{
					Secret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: naming.TLSSecretName(component, name),
						},
						Key: corev1.TLSCertKey,
					},
				},
				KeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: naming.TLSSecretName(component, name),
					},
					Key: corev1.TLSPrivateKeyKey,
				},
				// E.g. tempo-simplest-compactor.tempo-operator-system.svc.cluster.local
				ServerName: serverName,
			},
		}
	}

	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       monitoringv1.ServiceMonitorsKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      naming.Name(component, name),
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme:    scheme,
				Port:      port,
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
						Separator:    ptr.To("/"),
						TargetLabel:  "job",
					},
				},
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}
}
