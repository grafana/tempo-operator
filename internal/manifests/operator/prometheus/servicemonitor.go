package prometheus

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// ServiceMonitor creates a ServiceMonitor which scrapes the operator metrics.
func ServiceMonitor(featureGates configv1alpha1.FeatureGates, namespace string) *monitoringv1.ServiceMonitor {
	var tlsConfig *monitoringv1.TLSConfig

	if featureGates.OpenShift.ServingCertsService {
		tlsConfig = &monitoringv1.TLSConfig{
			CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName: fmt.Sprintf("tempo-operator-controller-manager-metrics-service.%s.svc", namespace),
			},
		}
	} else {
		tlsConfig = &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				// kube-rbac-proxy uses a self-signed cert by default
				InsecureSkipVerify: true,
			},
		}
	}

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "tempo-operator-controller-manager-metrics-monitor",
			Labels:    manifestutils.CommonOperatorLabels(),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme:          "https",
				Port:            "https",
				Path:            "/metrics",
				BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
				TLSConfig:       tlsConfig,
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			},
		},
	}
}
