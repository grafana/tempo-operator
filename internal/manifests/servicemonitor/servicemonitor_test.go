package servicemonitor

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/certrotation"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildServiceMonitors(t *testing.T) {
	objects := BuildServiceMonitors(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{},
	}})

	labels := manifestutils.ComponentLabels(manifestutils.CompactorComponentName, "test")
	assert.Len(t, objects, 5)
	assert.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-compactor",
			Namespace: "project1",
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
				MatchNames: []string{"project1"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, objects[0])
}

func TestBuildServiceMonitorsTLS(t *testing.T) {
	objects := BuildServiceMonitors(manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				HTTPEncryption: true,
			},
		},
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "project1",
			},
			Spec: v1alpha1.TempoStackSpec{},
		},
	})

	labels := manifestutils.ComponentLabels(manifestutils.CompactorComponentName, "test")
	assert.Len(t, objects, 5)
	assert.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-compactor",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme: "https",
				Port:   "http",
				Path:   "/metrics",
				TLSConfig: &monitoringv1.TLSConfig{
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						CA: monitoringv1.SecretOrConfigMap{
							ConfigMap: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "tempo-test-ca-bundle",
								},
								Key: certrotation.CAFile,
							},
						},
						Cert: monitoringv1.SecretOrConfigMap{
							Secret: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "tempo-test-compactor-mtls",
								},
								Key: corev1.TLSCertKey,
							},
						},
						KeySecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test-compactor-mtls",
							},
							Key: corev1.TLSPrivateKeyKey,
						},
						ServerName: "tempo-test-compactor.project1.svc.cluster.local",
					},
				},
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
				MatchNames: []string{"project1"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, objects[0])
}

func TestBuildGatewayServiceMonitor(t *testing.T) {
	objects := BuildServiceMonitors(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
		},
	}})

	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, "test")
	assert.Len(t, objects, 6)
	assert.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-gateway",
			Namespace: "project1",
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
				MatchNames: []string{"project1"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, objects[5])
}

func TestBuildGatewayServiceMonitorsTLS(t *testing.T) {
	objects := BuildServiceMonitors(manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				HTTPEncryption: true,
			},
		},
		Tempo: v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "project1",
			},
			Spec: v1alpha1.TempoStackSpec{
				Template: v1alpha1.TempoTemplateSpec{
					Gateway: v1alpha1.TempoGatewaySpec{
						Enabled: true,
					},
				},
				Tenants: &v1alpha1.TenantsSpec{
					Mode: v1alpha1.ModeOpenShift,
				},
			},
		},
	})

	labels := manifestutils.ComponentLabels(manifestutils.GatewayComponentName, "test")
	assert.Len(t, objects, 6)
	assert.Equal(t, &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-gateway",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Scheme: "https",
				Port:   "internal",
				Path:   "/metrics",
				TLSConfig: &monitoringv1.TLSConfig{
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						CA: monitoringv1.SecretOrConfigMap{
							ConfigMap: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "tempo-test-ca-bundle",
								},
								Key: certrotation.CAFile,
							},
						},
						Cert: monitoringv1.SecretOrConfigMap{
							Secret: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "tempo-test-gateway-mtls",
								},
								Key: corev1.TLSCertKey,
							},
						},
						KeySecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "tempo-test-gateway-mtls",
							},
							Key: corev1.TLSPrivateKeyKey,
						},
						ServerName: "tempo-test-gateway.project1.svc.cluster.local",
					},
				},
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
				MatchNames: []string{"project1"},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}, objects[5])
}
