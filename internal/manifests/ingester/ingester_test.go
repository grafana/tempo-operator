package ingester

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildIngester(t *testing.T) {
	objects, err := BuildIngester(v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: "test-storage-secret",
			},
		},
	})
	require.NoError(t, err)

	labels := manifestutils.ComponentLabels("ingester", "test")
	assert.Equal(t, 2, len(objects))
	assert.Equal(t, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-ingester",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: k8slabels.Merge(labels, map[string]string{"tempo-gossip-member": "true"}),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Args: []string{"-target=ingester",
								"-config.file=/conf/tempo.yaml",
								"--storage.trace.s3.secret_key=$(S3_SECRET_KEY)",
								"--storage.trace.s3.access_key=$(S3_ACCESS_KEY)"},
							Env: []corev1.EnvVar{
								{
									Name: "S3_SECRET_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "access_key_secret",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-storage-secret",
											},
										},
									},
								}, {
									Name: "S3_ACCESS_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "access_key_id",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-storage-secret",
											},
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http-memberlist",
									ContainerPort: 7946,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "http",
									ContainerPort: portHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "grpc",
									ContainerPort: portGRPCServer,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: configVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tempo-test",
									},
								},
							},
						},
					},
				},
			},
		},
	}, objects[0])
	assert.Equal(t, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-test-ingester",
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       portHTTPServer,
					TargetPort: intstr.FromString("http"),
				},
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       portGRPCServer,
					TargetPort: intstr.FromString("grpc"),
				},
			},
			Selector: labels,
		},
	}, objects[1])
}
