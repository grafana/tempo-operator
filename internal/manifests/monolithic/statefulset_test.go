package monolithic

import (
	"testing"
	"time"

	"github.com/operator-framework/operator-lib/proxy"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestStatefulsetMemoryStorage(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1Gi"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("3Gi"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{"tempo.grafana.com/tempoConfig.hash": "abc"})
	require.NoError(t, err)

	labels := ComponentLabels(manifestutils.TempoMonolithComponentName, "sample")

	proxyEnv := proxy.ReadProxyVarsFromEnv()
	proxyEnv = append(proxyEnv, corev1.EnvVar{
		Name:  "GOMEMLIMIT",
		Value: "3435973836",
	})

	require.Equal(t, &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-sample",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"tempo.grafana.com/tempoConfig.hash": "abc",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "tempo-sample",
					Affinity:           manifestutils.DefaultAffinity(labels),
					Containers: []corev1.Container{
						{
							Name:  "tempo",
							Image: "docker.io/grafana/tempo:x.y.z",
							Env:   proxyEnv,
							Args: []string{
								"-config.file=/conf/tempo.yaml",
								"-mem-ballast-size-mbs=1024",
								"-log.level=info",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tempo-conf",
									MountPath: "/conf",
									ReadOnly:  true,
								},
								{
									Name:      "tempo-storage",
									MountPath: "/var/tempo",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 3200,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "tempo-internal",
									ContainerPort: 3101,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "otlp-grpc",
									ContainerPort: 4317,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Scheme: corev1.URISchemeHTTP,
										Path:   "/ready",
										Port:   intstr.FromString("tempo-internal"),
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
							},
							SecurityContext: manifestutils.TempoContainerSecurityContext(),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1Gi"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("3Gi"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tempo-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tempo-sample-config",
									},
								},
							},
						},
						{
							Name: "tempo-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
					},
				},
			},
		},
	}, sts)
}

func TestStatefulsetPVStorage(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "pv",
						Size:    &tenGBQuantity,
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, []corev1.VolumeMount{
		{
			Name:      "tempo-conf",
			MountPath: "/conf",
			ReadOnly:  true,
		},
		{
			Name:      "tempo-storage",
			MountPath: "/var/tempo",
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	require.Equal(t, []corev1.Volume{
		{
			Name: "tempo-conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-config",
					},
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	require.Equal(t, []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tempo-storage",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: tenGBQuantity,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		},
	}, sts.Spec.VolumeClaimTemplates)
}

func TestStatefulsetS3TLSStorage(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "s3",
						Size:    &tenGBQuantity,
						S3: &v1alpha1.MonolithicTracesStorageS3Spec{
							MonolithicTracesObjectStorageSpec: v1alpha1.MonolithicTracesObjectStorageSpec{
								Secret: "storage-secret",
							},
							TLS: &v1alpha1.TLSSpec{
								Enabled: true,
								Cert:    "custom-cert",
								CA:      "custom-ca",
							},
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, []corev1.VolumeMount{
		{
			Name:      "tempo-conf",
			MountPath: "/conf",
			ReadOnly:  true,
		},
		{
			Name:      "tempo-storage",
			MountPath: "/var/tempo",
		},
		{
			Name:      "custom-ca",
			MountPath: "/var/run/tls/storage/ca",
			ReadOnly:  true,
		},
		{
			Name:      "custom-cert",
			MountPath: "/var/run/tls/storage/cert",
			ReadOnly:  true,
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	require.Equal(t, []corev1.EnvVar{
		{
			Name: "S3_SECRET_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "access_key_secret",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "storage-secret",
					},
				},
			},
		},
		{
			Name: "S3_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "access_key_id",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "storage-secret",
					},
				},
			},
		},
	}, sts.Spec.Template.Spec.Containers[0].Env)

	require.Equal(t, []string{
		"-config.file=/conf/tempo.yaml",
		"-mem-ballast-size-mbs=1024",
		"-log.level=info",
		"--storage.trace.s3.secret_key=$(S3_SECRET_KEY)",
		"--storage.trace.s3.access_key=$(S3_ACCESS_KEY)",
	}, sts.Spec.Template.Spec.Containers[0].Args)

	require.Equal(t, []corev1.Volume{
		{
			Name: "tempo-conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-config",
					},
				},
			},
		},
		{
			Name: "custom-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "custom-ca",
					},
				},
			},
		},
		{
			Name: "custom-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "custom-cert",
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)

	require.Equal(t, []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tempo-storage",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: tenGBQuantity,
					},
				},
				VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
			},
		},
	}, sts.Spec.VolumeClaimTemplates)
}

func TestStatefulsetReceiverTLS(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
							TLS: &v1alpha1.TLSSpec{
								Enabled: true,
								CA:      "custom-ca",
								Cert:    "custom-cert",
							},
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, []corev1.VolumeMount{
		{
			Name:      "tempo-conf",
			MountPath: "/conf",
			ReadOnly:  true,
		},
		{
			Name:      "tempo-storage",
			MountPath: "/var/tempo",
		},
		{
			Name:      "custom-ca",
			MountPath: "/var/run/ca-receiver/grpc",
			ReadOnly:  true,
		},
		{
			Name:      "custom-cert",
			MountPath: "/var/run/tls/receiver/grpc",
			ReadOnly:  true,
		},
	}, sts.Spec.Template.Spec.Containers[0].VolumeMounts)

	require.Equal(t, []corev1.Volume{
		{
			Name: "tempo-conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-config",
					},
				},
			},
		},
		{
			Name: "tempo-storage",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			Name: "custom-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "custom-ca",
					},
				},
			},
		},
		{
			Name: "custom-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "custom-cert",
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes)
}

func TestStatefulsetPorts(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		input    *v1alpha1.MonolithicIngestionSpec
		expected []corev1.ContainerPort
	}{
		{
			name:  "no ingestion ports",
			input: nil,
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "tempo-internal",
					ContainerPort: 3101,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		{
			name: "OTLP/gRPC",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "tempo-internal",
					ContainerPort: 3101,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "otlp-grpc",
					ContainerPort: 4317,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		{
			name: "OTLP/HTTP",
			input: &v1alpha1.MonolithicIngestionSpec{
				OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
					HTTP: &v1alpha1.MonolithicIngestionOTLPProtocolsHTTPSpec{
						Enabled: true,
					},
				},
			},
			expected: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 3200,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "tempo-internal",
					ContainerPort: 3101,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "otlp-http",
					ContainerPort: 4318,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec.Ingestion = test.input
			sts, err := BuildTempoStatefulset(opts, map[string]string{})
			require.NoError(t, err)
			require.Equal(t, test.expected, sts.Spec.Template.Spec.Containers[0].Ports)
		})
	}
}

func TestStatefulsetSchedulingRules(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				MonolithicSchedulerSpec: v1alpha1.MonolithicSchedulerSpec{
					NodeSelector: map[string]string{
						"key1": "value1",
					},
					Tolerations: []corev1.Toleration{{
						Key:      "example",
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					}},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      "topology.kubernetes.io/zone",
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											"eu-west-1",
											"eu-west-2",
										},
									}},
								}},
							},
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, map[string]string{
		"key1": "value1",
	}, sts.Spec.Template.Spec.NodeSelector)

	require.Equal(t, []corev1.Toleration{{
		Key:      "example",
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	}}, sts.Spec.Template.Spec.Tolerations)

	require.Equal(t, &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "topology.kubernetes.io/zone",
						Operator: corev1.NodeSelectorOpIn,
						Values: []string{
							"eu-west-1",
							"eu-west-2",
						},
					}},
				}},
			},
		},
	}, sts.Spec.Template.Spec.Affinity)
}

func TestStatefulsetCustomServiceAccount(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no custom serviceaccount",
			input:    "",
			expected: "tempo-sample",
		},
		{
			name:     "custom serviceaccount",
			input:    "custom-sa",
			expected: "custom-sa",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts.Tempo.Spec.ServiceAccount = test.input
			sts, err := BuildTempoStatefulset(opts, map[string]string{})
			require.NoError(t, err)
			require.Equal(t, test.expected, sts.Spec.Template.Spec.ServiceAccountName)
		})
	}
}

func TestStatefulsetGateway(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				TempoGateway:    "quay.io/observatorium/api:x.y.z",
				TempoGatewayOpa: "quay.io/observatorium/opa-openshift:x.y.z",
			},
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Timeout: metav1.Duration{Duration: time.Second * 5},
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				// Fix nil pointer dereference
				Query: &v1alpha1.MonolithicQuerySpec{},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
				JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
					Enabled: true,
				},
				Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
					Enabled: true,
					TenantsSpec: v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeOpenShift,
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "dev",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfa",
							},
							{
								TenantName: "prod",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfb",
							},
						},
					},
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1Gi"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("3Gi"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, corev1.Container{
		Name:  "tempo-gateway",
		Image: "quay.io/observatorium/api:x.y.z",
		Env:   proxy.ReadProxyVarsFromEnv(),
		Args: []string{
			"--web.listen=0.0.0.0:8080",
			"--web.internal.listen=0.0.0.0:8081",
			"--traces.tenant-header=x-scope-orgid",
			"--traces.tempo.endpoint=http://localhost:3200",
			"--traces.write-timeout=5s",
			"--rbac.config=/etc/tempo-gateway/rbac/rbac.yaml",
			"--tenants.config=/etc/tempo-gateway/tenants/tenants.yaml",
			"--log.level=info",
			"--grpc.listen=0.0.0.0:8090",
			"--traces.write.otlpgrpc.endpoint=localhost:4317",
			"--traces.write.otlphttp.endpoint=http://localhost:4318",
			"--traces.read.endpoint=http://localhost:16686",
			"--tls.server.cert-file=/etc/tempo-gateway/serving-cert/tls.crt",
			"--tls.server.key-file=/etc/tempo-gateway/serving-cert/tls.key",
			"--tls.healthchecks.server-ca-file=/etc/tempo-gateway/serving-ca/service-ca.crt",
			"--tls.healthchecks.server-name=tempo-sample-gateway.default.svc.cluster.local",
			"--web.healthchecks.url=https://localhost:8080",
			"--tls.client-auth-type=NoClientCert",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "public",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "internal",
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "grpc-public",
				ContainerPort: 8090,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Port:   intstr.FromString("internal"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   2,
			PeriodSeconds:    30,
			FailureThreshold: 10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/ready",
					Port:   intstr.FromString("internal"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   1,
			PeriodSeconds:    5,
			FailureThreshold: 12,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "gateway-rbac",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/rbac",
			},
			{
				Name:      "gateway-tenants",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/tenants",
			},
			{
				Name:      "tempo-sample-serving-cabundle",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/serving-ca",
			},
			{
				Name:      "tempo-sample-gateway-serving-cert",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/serving-cert",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1Gi"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3Gi"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
	}, sts.Spec.Template.Spec.Containers[3])

	require.Equal(t, []corev1.Volume{
		{
			Name: "gateway-rbac",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-gateway",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "rbac.yaml",
							Path: "rbac.yaml",
						},
					},
				},
			},
		},
		{
			Name: "gateway-tenants",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "tempo-sample-gateway",
					Items: []corev1.KeyToPath{
						{
							Key:  "tenants.yaml",
							Path: "tenants.yaml",
						},
					},
				},
			},
		},
		{
			Name: "tempo-sample-serving-cabundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-serving-cabundle",
					},
				},
			},
		},
		{
			Name: "tempo-sample-gateway-serving-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "tempo-sample-gateway-serving-cert",
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes[3:])

	require.Equal(t, corev1.Container{
		Name:  "tempo-gateway-opa",
		Image: "quay.io/observatorium/opa-openshift:x.y.z",
		Args: []string{
			"--log.level=warn",
			"--web.listen=:8082",
			"--web.internal.listen=:8083",
			"--web.healthchecks.url=http://localhost:8082",
			"--opa.package=tempomonolithic",
			"--opa.ssar",
			"--openshift.mappings=dev=tempo.grafana.com",
			"--openshift.mappings=prod=tempo.grafana.com",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "public",
				ContainerPort: 8082,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "opa-metrics",
				ContainerPort: 8083,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Port:   intstr.FromInt(8083),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   2,
			PeriodSeconds:    30,
			FailureThreshold: 10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/ready",
					Port:   intstr.FromInt(8083),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   1,
			PeriodSeconds:    5,
			FailureThreshold: 12,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1Gi"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3Gi"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
	}, sts.Spec.Template.Spec.Containers[4])
}

func TestStatefulsetGatewayRBAC(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				TempoGateway:    "quay.io/observatorium/api:x.y.z",
				TempoGatewayOpa: "quay.io/observatorium/opa-openshift:x.y.z",
			},
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Timeout: metav1.Duration{Duration: time.Second * 5},
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				Ingestion: &v1alpha1.MonolithicIngestionSpec{
					OTLP: &v1alpha1.MonolithicIngestionOTLPSpec{
						GRPC: &v1alpha1.MonolithicIngestionOTLPProtocolsGRPCSpec{
							Enabled: true,
						},
					},
				},
				JaegerUI: &v1alpha1.MonolithicJaegerUISpec{
					Enabled: false,
				},
				Query: &v1alpha1.MonolithicQuerySpec{
					RBAC: v1alpha1.RBACSpec{
						Enabled: true,
					},
				},
				Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
					Enabled: true,
					TenantsSpec: v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeOpenShift,
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								TenantName: "dev",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfa",
							},
							{
								TenantName: "prod",
								TenantID:   "1610b0c3-c509-4592-a256-a1871353dbfb",
							},
						},
					},
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1Gi"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("3Gi"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, corev1.Container{
		Name:  "tempo-gateway",
		Image: "quay.io/observatorium/api:x.y.z",
		Env:   proxy.ReadProxyVarsFromEnv(),
		Args: []string{
			"--web.listen=0.0.0.0:8080",
			"--web.internal.listen=0.0.0.0:8081",
			"--traces.tenant-header=x-scope-orgid",
			"--traces.tempo.endpoint=http://localhost:3200",
			"--traces.write-timeout=5s",
			"--rbac.config=/etc/tempo-gateway/rbac/rbac.yaml",
			"--tenants.config=/etc/tempo-gateway/tenants/tenants.yaml",
			"--log.level=info",
			"--grpc.listen=0.0.0.0:8090",
			"--traces.write.otlpgrpc.endpoint=localhost:4317",
			"--traces.write.otlphttp.endpoint=http://localhost:4318",
			"--traces.query-rbac=true",
			"--tls.server.cert-file=/etc/tempo-gateway/serving-cert/tls.crt",
			"--tls.server.key-file=/etc/tempo-gateway/serving-cert/tls.key",
			"--tls.healthchecks.server-ca-file=/etc/tempo-gateway/serving-ca/service-ca.crt",
			"--tls.healthchecks.server-name=tempo-sample-gateway.default.svc.cluster.local",
			"--web.healthchecks.url=https://localhost:8080",
			"--tls.client-auth-type=NoClientCert",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "public",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "internal",
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "grpc-public",
				ContainerPort: 8090,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Port:   intstr.FromString("internal"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   2,
			PeriodSeconds:    30,
			FailureThreshold: 10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/ready",
					Port:   intstr.FromString("internal"),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   1,
			PeriodSeconds:    5,
			FailureThreshold: 12,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "gateway-rbac",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/rbac",
			},
			{
				Name:      "gateway-tenants",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/tenants",
			},
			{
				Name:      "tempo-sample-serving-cabundle",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/serving-ca",
			},
			{
				Name:      "tempo-sample-gateway-serving-cert",
				ReadOnly:  true,
				MountPath: "/etc/tempo-gateway/serving-cert",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1Gi"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3Gi"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
		SecurityContext: manifestutils.TempoContainerSecurityContext(),
	}, sts.Spec.Template.Spec.Containers[1])

	require.Equal(t, []corev1.Volume{
		{
			Name: "gateway-rbac",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-gateway",
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "rbac.yaml",
							Path: "rbac.yaml",
						},
					},
				},
			},
		},
		{
			Name: "gateway-tenants",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "tempo-sample-gateway",
					Items: []corev1.KeyToPath{
						{
							Key:  "tenants.yaml",
							Path: "tenants.yaml",
						},
					},
				},
			},
		},
		{
			Name: "tempo-sample-serving-cabundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "tempo-sample-serving-cabundle",
					},
				},
			},
		},
		{
			Name: "tempo-sample-gateway-serving-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "tempo-sample-gateway-serving-cert",
				},
			},
		},
	}, sts.Spec.Template.Spec.Volumes[2:])

	require.Equal(t, corev1.Container{
		Name:  "tempo-gateway-opa",
		Image: "quay.io/observatorium/opa-openshift:x.y.z",
		Args: []string{
			"--log.level=warn",
			"--web.listen=:8082",
			"--web.internal.listen=:8083",
			"--web.healthchecks.url=http://localhost:8082",
			"--opa.package=tempomonolithic",
			"--opa.ssar",
			"--opa.matcher=kubernetes_namespace_name",
			"--openshift.mappings=dev=tempo.grafana.com",
			"--openshift.mappings=prod=tempo.grafana.com",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "public",
				ContainerPort: 8082,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "opa-metrics",
				ContainerPort: 8083,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/live",
					Port:   intstr.FromInt(8083),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   2,
			PeriodSeconds:    30,
			FailureThreshold: 10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/ready",
					Port:   intstr.FromInt(8083),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			TimeoutSeconds:   1,
			PeriodSeconds:    5,
			FailureThreshold: 12,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1Gi"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3Gi"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		},
	}, sts.Spec.Template.Spec.Containers[2])
}

func TestStatefulsetCustomStorageClass(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend:          "pv",
						Size:             &tenGBQuantity,
						StorageClassName: ptr.To("custom-storage-class"),
					},
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, ptr.To("custom-storage-class"), sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
}

func TestStatefulsetPodSecurityContext(t *testing.T) {
	opts := Options{
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:x.y.z",
			},
		},
		Tempo: v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: v1alpha1.TempoMonolithicSpec{
				Storage: &v1alpha1.MonolithicStorageSpec{
					Traces: v1alpha1.MonolithicTracesStorageSpec{
						Backend: "memory",
					},
				},
				PodSecurityContext: &corev1.PodSecurityContext{
					RunAsUser:  ptr.To(int64(10001)),
					RunAsGroup: ptr.To(int64(10001)),
				},
			},
		},
	}
	sts, err := BuildTempoStatefulset(opts, map[string]string{})
	require.NoError(t, err)

	require.Equal(t, &corev1.PodSecurityContext{
		RunAsUser:  ptr.To(int64(10001)),
		RunAsGroup: ptr.To(int64(10001)),
	}, sts.Spec.Template.Spec.SecurityContext)
}
