package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
)

func TestPatchOCPServingCerts(t *testing.T) {
	tempo := v1alpha1.Microservices{}
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
	expected.Spec.Template.Spec.Containers[0].Args = append(expected.Spec.Template.Spec.Containers[0].Args,
		[]string{
			"--tls.server.cert-file=/etc/tempo-gateway/serving-certs/tls.crt",
			"--tls.server.key-file=/etc/tempo-gateway/serving-certs/tls.key",
		}...)
	expected.Spec.Template.Spec.Containers[0].VolumeMounts = append(expected.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "serving-certs",
			ReadOnly:  true,
			MountPath: "/etc/tempo-gateway/serving-certs",
		})

	got, err := patchOCPServingCerts(tempo, dep)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}
