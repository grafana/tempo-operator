package prometheus

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestServiceMonitorWithoutTLS(t *testing.T) {
	servicemonitor := ServiceMonitor(v1alpha1.FeatureGates{}, "tempo-operator-system")

	assert.Equal(t, &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-controller-manager-metrics-monitor",
			Namespace: "tempo-operator-system",
			Labels:    manifestutils.CommonOperatorLabels(),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme:          "https",
				Port:            "https",
				Path:            "/metrics",
				BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
				TLSConfig: &monitoringv1.TLSConfig{
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						InsecureSkipVerify: true,
					},
				},
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			},
		},
	}, servicemonitor)
}

func TestServiceMonitorWithTLS(t *testing.T) {
	servicemonitor := ServiceMonitor(
		v1alpha1.FeatureGates{
			OpenShift: v1alpha1.OpenShiftFeatureGates{
				ServingCertsService: true,
			},
		},
		"tempo-operator-system",
	)

	assert.Equal(t, &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-controller-manager-metrics-monitor",
			Namespace: "tempo-operator-system",
			Labels:    manifestutils.CommonOperatorLabels(),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme:          "https",
				Port:            "https",
				Path:            "/metrics",
				BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
				TLSConfig: &monitoringv1.TLSConfig{
					CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						ServerName: "tempo-operator-controller-manager-metrics-service.tempo-operator-system.svc",
					},
				},
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			},
		},
	}, servicemonitor)
}
