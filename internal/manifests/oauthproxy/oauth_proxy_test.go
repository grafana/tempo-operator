package oauthproxy

import (
	"fmt"

	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lib/proxy"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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

	customImage := "custom_image/special_oauth_proxy:99"

	tests := []struct {
		name          string
		expectedImage string
		expectedArgs  []string
		tempo         v1alpha1.TempoStack
	}{
		{
			name:          " no SAR",
			expectedImage: customImage,
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
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
									Enabled: true,
								},
							},
						},
					},
				},
			},
		},
		{
			name:          "SAR defined",
			expectedImage: customImage,
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
								Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
									Enabled: true,
									SAR:     "{\"namespace\":\"app-dev\",\"resource\":\"services\",\"resourceName\":\"proxy\",\"verb\":\"get\"}",
								},
							},
						},
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
						OauthProxy: customImage,
					},
				},
			}
			params.Tempo = test.tempo
			replicas := int32(1)
			container := oAuthProxyContainer(params.Tempo.Name,
				naming.Name(manifestutils.QueryFrontendComponentName, params.Tempo.Name),
				params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication,
				customImage,
			)
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
					Name:      getTLSSecretNameForFrontendService(test.tempo.Name),
				},

					{
						MountPath: oauthProxySecretMountPath,
						Name:      cookieSecretName(test.tempo.Name),
					},
				},
				Resources: manifestutils.Resources(test.tempo, manifestutils.QueryFrontendComponentName, &replicas),
				Env:       proxy.ReadProxyVarsFromEnv(),
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Scheme: corev1.URISchemeHTTPS,
							Path:   healthPath,
							Port:   intstr.FromString(manifestutils.OAuthProxyPortName),
						},
					},
					InitialDelaySeconds: oauthReadinessProbeInitialDelaySeconds,
					TimeoutSeconds:      oauthReadinessProbeTimeoutSeconds,
				},
				SecurityContext: manifestutils.TempoContainerSecurityContext(),
			}
			assert.Equal(t, expected, container)
		})
	}
}

func TestOAuthProxyServiceAccount(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testoauthsecret",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	service := OAuthServiceAccount(tempo.ObjectMeta)

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
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "tempoi"),
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
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	params := manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				OauthProxy: defaultImage,
			},
		},
		Tempo: tempo,
	}

	PatchDeploymentForOauthProxy(
		params.Tempo.ObjectMeta,
		params.CtrlConfig,
		params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication,
		params.Tempo.Spec.Images,
		dep)

	assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
	assert.Equal(t, "oauth-proxy", dep.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, naming.Name(manifestutils.QueryFrontendComponentName, tempo.Name), dep.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, 4, len(dep.Spec.Template.Spec.Volumes))

}

func TestPatchStatefulSetForOauthProxy(t *testing.T) {
	labels := manifestutils.ComponentLabels("query-frontend", "test")
	annotations := manifestutils.CommonAnnotations("")
	defaultImage := "myrepo/oauth_proxy:1.1"

	statefulSet := &v1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.QueryFrontendComponentName, "tempoi"),
			Namespace: "project1",
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
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
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Authentication: &v1alpha1.JaegerQueryAuthenticationSpec{
							Enabled: true,
						},
					},
				},
			},
		},
	}

	params := manifestutils.Params{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				OauthProxy: defaultImage,
			},
		},
		Tempo: tempo,
	}

	PatchStatefulSetForOauthProxy(
		params.Tempo.ObjectMeta,
		params.Tempo.Spec.Template.QueryFrontend.JaegerQuery.Authentication,
		params.CtrlConfig,
		statefulSet)

	assert.Equal(t, 2, len(statefulSet.Spec.Template.Spec.Containers))
	assert.Equal(t, "oauth-proxy", statefulSet.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, "tempo-test-serviceaccount", statefulSet.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, 4, len(statefulSet.Spec.Template.Spec.Volumes))

}

func TestPatchQueryFrontEndService(t *testing.T) {
	ports := []corev1.ServicePort{
		{
			Name:       manifestutils.JaegerGRPCQuery,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerGRPCQuery,
			TargetPort: intstr.FromString(manifestutils.JaegerGRPCQuery),
		},
		{
			Name:       manifestutils.JaegerUIPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerUI,
			TargetPort: intstr.FromString(manifestutils.JaegerUIPortName),
		},
		{
			Name:       manifestutils.JaegerMetricsPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       manifestutils.PortJaegerMetrics,
			TargetPort: intstr.FromString(manifestutils.JaegerMetricsPortName),
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.JaegerUIComponentName, "test"),
			Namespace: "ns-test",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	}

	PatchQueryFrontEndService(service, "test")

	newPorts := append([]corev1.ServicePort{}, ports...)

	assert.Equal(t, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.JaegerUIComponentName, "test"),
			Namespace: "ns-test",
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": "test-ui-oauth-proxy-tls",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: append(newPorts, corev1.ServicePort{
				Name:       manifestutils.OAuthProxyPortName,
				Port:       manifestutils.OAuthProxyPort,
				TargetPort: intstr.FromString(manifestutils.OAuthProxyPortName),
			}),
		},
	}, service)
}

func TestPatchRouteForOauthProxy(t *testing.T) {
	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.JaegerUIComponentName, "test"),
			Namespace: "test-ns",
		},
		Spec: routev1.RouteSpec{
			Host: "localhost",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "Xservice",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("targetPort"),
			},
			TLS: &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough},
		},
	}
	PatchRouteForOauthProxy(route)

	assert.Equal(t, &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkingv1.SchemeGroupVersion.String(),
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.JaegerUIComponentName, "test"),
			Namespace: "test-ns",
		},
		Spec: routev1.RouteSpec{
			Host: "localhost",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "Xservice",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(manifestutils.OAuthProxyPortName),
			},
			TLS: &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt},
		},
	}, route)

}

func TestAddServiceAccountAnnotations(t *testing.T) {
	serviceAccounnt := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.DefaultServiceAccountName("test"),
			Namespace: "test-ns",
		},
	}
	AddServiceAccountAnnotations(serviceAccounnt, "my-route")
	assert.Equal(t, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.DefaultServiceAccountName("test"),
			Namespace: "test-ns",
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.primary": `{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"my-route"}}`,
			},
		},
	}, serviceAccounnt)
}

func TestOAuthCookieSessionSecret(t *testing.T) {
	secret, err := OAuthCookieSessionSecret(metav1.ObjectMeta{
		Name:      "test",
		Namespace: "test-ns",
	})

	assert.NoError(t, err)
	assert.Equal(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cookieSecretName("test"),
			Labels:    manifestutils.ComponentLabels(manifestutils.QueryFrontendOauthProxyComponentName, "test"),
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			// Override this, because is random data, so we need to force to match
			sessionSecretKey: secret.Data[sessionSecretKey],
		},
	}, secret)
}
