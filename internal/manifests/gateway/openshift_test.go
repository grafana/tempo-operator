package gateway

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func TestPatchOPAContainer(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeOpenShift,
				Authentication: []v1alpha1.AuthenticationSpec{
					{
						TenantName: "dev",
						TenantID:   "abcd1",
					},
					{
						TenantName: "prod",
						TenantID:   "abcd2",
					},
				},
			},
		},
	}
	dep, err := patchOCPOPAContainer(manifestutils.Params{Tempo: tempo}, &appsv1.Deployment{})
	require.NoError(t, err)
	require.Equal(t, 1, len(dep.Spec.Template.Spec.Containers))
	assert.Equal(t, []string{
		"--log.level=warn",
		"--web.listen=:8082", "--web.internal.listen=:8083",
		"--web.healthchecks.url=http://localhost:8082",
		"--opa.package=tempostack",
		"--opa.ssar",
		"--openshift.mappings=dev=tempo.grafana.com",
		"--openshift.mappings=prod=tempo.grafana.com",
	}, dep.Spec.Template.Spec.Containers[0].Args)
}

func TestPatchOCPServingCerts(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
	}
	dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "data",
						},
					},
					Containers: []corev1.Container{
						{
							Args: []string{"--help"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name: "data",
								},
							},
						},
					},
				},
			},
		},
	}
	expected := dep.DeepCopy()
	expected.Spec.Template.Spec.Volumes = append(expected.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "serving-certs",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: naming.Name("gateway-tls", tempo.Name),
			},
		},
	})
	expected.Spec.Template.Spec.Volumes = append(expected.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "cabundle",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: naming.Name("gateway-cabundle", tempo.Name),
				},
			},
		},
	})
	expected.Spec.Template.Spec.Containers[0].Args = append(expected.Spec.Template.Spec.Containers[0].Args,
		[]string{
			"--tls.server.cert-file=/etc/tempo-gateway/serving-certs/tls.crt",
			"--tls.server.key-file=/etc/tempo-gateway/serving-certs/tls.key",
			"--tls.healthchecks.server-ca-file=/etc/tempo-gateway/cabundle/service-ca.crt",
			fmt.Sprintf("--tls.healthchecks.server-name=tempo-%s-gateway.%s.svc.cluster.local", tempo.Name, tempo.Namespace),
			"--web.healthchecks.url=https://localhost:8080",
			"--tls.client-auth-type=NoClientCert",
		}...)
	expected.Spec.Template.Spec.Containers[0].VolumeMounts = append(expected.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "serving-certs",
			ReadOnly:  true,
			MountPath: "/etc/tempo-gateway/serving-certs",
		})
	expected.Spec.Template.Spec.Containers[0].VolumeMounts = append(expected.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "cabundle",
			ReadOnly:  true,
			MountPath: "/etc/tempo-gateway/cabundle",
		})

	got, err := patchOCPServingCerts(tempo, dep)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}
