package queryfrontend

import (
	"fmt"
	"testing"

	"github.com/operator-framework/operator-lib/proxy"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/memberlist"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func TestOauthProxyContainer(t *testing.T) {

	defaultImage := "myrepo/oauth_proxy:1.1"
	customImage := "custom_image/special_oauth_proxy:99"

	tests := []struct {
		name          string
		expectedImage string
		expectedArgs  []string
		tempo         v1alpha1.TempoStack
	}{
		{
			name:          "default image, no SAR",
			expectedImage: defaultImage,
			expectedArgs: []string{
				fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
				fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
				fmt.Sprintf("--openshift-service-account=%s", naming.Name(manifestutils.QueryFrontendComponentName, "test")),
				"--provider=openshift",
				fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
				fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
				fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
			},
			tempo: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "project1",
				},
			},
		},
		{
			name:          "default image, SAR defined",
			expectedImage: defaultImage,
			expectedArgs: []string{
				fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
				fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
				fmt.Sprintf("--openshift-service-account=%s", naming.Name(manifestutils.QueryFrontendComponentName, "test2")),
				"--provider=openshift",
				fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
				fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
				fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
				"--openshift-sar={\"namespace\":\"app-dev\",\"resource\":\"services\",\"resourceName\":\"proxy\",\"verb\":\"get\"}",
			},
			tempo: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test2",
					Namespace: "project1",
				},
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Ingress: v1alpha1.IngressSpec{
									Security: v1alpha1.IngressSecuritySpec{
										SAR: "{\"namespace\":\"app-dev\",\"resource\":\"services\",\"resourceName\":\"proxy\",\"verb\":\"get\"}",
									},
								},
							},
						},
					},
				},
			},
		},

		{
			name:          "set custom image",
			expectedImage: customImage,
			expectedArgs: []string{
				fmt.Sprintf("--cookie-secret-file=%s/%s", oauthProxySecretMountPath, sessionSecretKey),
				fmt.Sprintf("--https-address=:%d", manifestutils.OAuthProxyPort),
				fmt.Sprintf("--openshift-service-account=%s", naming.Name(manifestutils.QueryFrontendComponentName, "test3")),
				"--provider=openshift",
				fmt.Sprintf("--tls-cert=%s/tls.crt", tlsProxyPath),
				fmt.Sprintf("--tls-key=%s/tls.key", tlsProxyPath),
				fmt.Sprintf("--upstream=http://localhost:%d", manifestutils.PortJaegerUI),
			},
			tempo: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test3",
					Namespace: "project1",
				},
				Spec: v1alpha1.TempoStackSpec{
					Images: configv1alpha1.ImagesSpec{
						OauthProxy: customImage,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := manifestutils.Params{
				CtrlConfig: configv1alpha1.ProjectConfig{
					DefaultImages: configv1alpha1.ImagesSpec{
						OauthProxy: defaultImage,
					},
				},
			}
			params.Tempo = test.tempo
			container := oAuthProxyContainer(params)
			expected := corev1.Container{
				Image: test.expectedImage,
				Name:  "oauth-proxy",
				Args:  test.expectedArgs,
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: manifestutils.OAuthProxyPort,
						Name:          manifestutils.OAuthProxyPortName,
					},
				},
				VolumeMounts: []corev1.VolumeMount{{
					MountPath: tlsProxyPath,
					Name:      getTLSSecretNameForFrontendService(test.tempo),
				},

					{
						MountPath: oauthProxySecretMountPath,
						Name:      cookieSecretName(test.tempo),
					},
				},
				Resources: resources(test.tempo),
				Env:       proxy.ReadProxyVarsFromEnv(),
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Scheme: corev1.URISchemeHTTPS,
							Path:   healthPath,
							Port:   intstr.FromString(manifestutils.OAuthProxyPortName),
						},
					},
					InitialDelaySeconds: 15,
					TimeoutSeconds:      5,
				},
			}
			assert.Equal(t, expected, container)
		})
	}
}

func TestOAuthProxyService(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testoauthsecret",
			Namespace: "project1",
		},
	}

	service := oauthProxyService(tempo)

	assert.Equal(t,
		naming.Name(manifestutils.QueryFrontendOauthProxyComponentName, "testoauthsecret"), service.Name)

	assert.Equal(t,
		map[string]string{
			"service.beta.openshift.io/serving-cert-secret-name": getTLSSecretNameForFrontendService(tempo),
		}, service.Annotations)
}

func TestOAuthProxyServiceAccount(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testoauthsecret",
			Namespace: "project1",
		},
	}

	service := oauthServiceAccount(tempo)

	assert.Equal(t,
		naming.Name(manifestutils.QueryFrontendComponentName, "testoauthsecret"), service.Name)

	assert.Equal(t,
		map[string]string{
			"serviceaccounts.openshift.io/oauth-redirectreference.primary": `{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"tempo-testoauthsecret-query-frontend"}}`,
		}, service.Annotations)
}

func TestPatchDeploymentForOauthProxy(t *testing.T) {
	labels := manifestutils.ComponentLabels("query-frontend", "test")
	annotations := manifestutils.CommonAnnotations("")
	defaultImage := "myrepo/oauth_proxy:1.1"

	dep := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "test"),
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      k8slabels.Merge(labels, memberlist.GossipSelector),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "tempo-test-serviceaccount",
					Affinity:           manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:1.5.0",
							Env:   []corev1.EnvVar{},
							Args: []string{
								"-target=query-frontend",
								"-config.file=/conf/tempo-query-frontend.yaml",
								"-mem-ballast-size-mbs=1024",
								"-log.level=info",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          manifestutils.HttpPortName,
									ContainerPort: manifestutils.PortHTTPServer,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          manifestutils.GrpcPortName,
									ContainerPort: manifestutils.PortGRPCServer,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: manifestutils.TempoReadinessProbe(false),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      manifestutils.ConfigVolumeName,
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      manifestutils.TmpStorageVolumeName,
									MountPath: manifestutils.TmpTempoStoragePath,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(90, resource.BinarySI),
									corev1.ResourceMemory: *resource.NewQuantity(107374184, resource.BinarySI),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(27, resource.BinarySI),
									corev1.ResourceMemory: *resource.NewQuantity(32212256, resource.BinarySI),
								},
							},
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: manifestutils.ConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: naming.Name("", "test"),
									},
								},
							},
						},
						{
							Name: manifestutils.TmpStorageVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test3",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{},
	}

	params := manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				OauthProxy: defaultImage,
			},
		},
		Tempo: tempo,
	}

	patchDeploymentForOauthProxy(params, dep)

	assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name), dep.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, 4, len(dep.Spec.Template.Spec.Volumes))

}
