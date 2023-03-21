package gateway

import (
	"testing"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/manifests/naming"
	"github.com/os-observability/tempo-operator/internal/tlsprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestRbacConfig (t *testing.T){
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{
				Mode: "static",
				Authorization: &v1alpha1.AuthorizationSpec{
					RoleBindings: []v1alpha1.RoleBindingsSpec{
						{
							Name: "test",
							Roles: []string{"read-write"},
							Subjects: []v1alpha1.Subject{
								{
								Name: "admin@example.com",
								Kind: v1alpha1.User,
								},
							},
						},
					},
					Roles: []v1alpha1.RoleSpec{{
						Name: "read-write",
						Resources: []string{
							"logs", "metrics", "traces",
						},
						Tenants: []string{
							"test-oidc",
						},
						Permissions: []v1alpha1.PermissionType{v1alpha1.Write, v1alpha1.Read},
					},
					},
				},
			},
		},
	}
	params := manifestutils.Params{
		StorageParams: manifestutils.StorageParams{},
		ConfigChecksum: "",
		Tempo: tempo,
		Gates: configv1alpha1.FeatureGates{},
		TLSProfile: tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantSecret{},
	}

	tenantsCfg, _, err := buildConfigFiles(options{
		Namespace:     params.Tempo.Namespace,
		Name:          params.Tempo.Name,
		BaseDomain:    params.Gates.OpenShift.BaseDomain,
		Tenants:       params.Tempo.Spec.Tenants,
		TenantSecrets: params.GatewayTenantSecret,
	})
	assert.NoError(t, err)

	secret, hash := rbacConfig(tempo, tenantsCfg)
	assert.Equal(t, "f7945d87b710f8df423b9d926d771951b0307fc8cb050f7dbc8773fff3febaa8", hash)
	assert.NotEmpty(t, secret.Data["rbac.yaml"])
}


func TestTenantsConfig(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{
				Mode: "static",
				Authorization: &v1alpha1.AuthorizationSpec{
					RoleBindings: []v1alpha1.RoleBindingsSpec{
						{
							Name: "test",
							Roles: []string{"read-write"},
							Subjects: []v1alpha1.Subject{
								{
								Name: "admin@example.com",
								Kind: v1alpha1.User,
								},
							},
						},
					},
					Roles: []v1alpha1.RoleSpec{{
						Name: "read-write",
						Resources: []string{
							"logs", "metrics", "traces",
						},
						Tenants: []string{
							"test-oidc",
						},
						Permissions: []v1alpha1.PermissionType{v1alpha1.Write, v1alpha1.Read},
					},
					},
				},
			},
		},
	}
	params := manifestutils.Params{
		StorageParams: manifestutils.StorageParams{},
		ConfigChecksum: "",
		Tempo: tempo,
		Gates: configv1alpha1.FeatureGates{},
		TLSProfile: tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantSecret{},
	}

	_, tenantsCfg, err := buildConfigFiles(options{
		Namespace:     params.Tempo.Namespace,
		Name:          params.Tempo.Name,
		BaseDomain:    params.Gates.OpenShift.BaseDomain,
		Tenants:       params.Tempo.Spec.Tenants,
		TenantSecrets: params.GatewayTenantSecret,
	})
	assert.NoError(t, err)

	secret, hash := tenantsConfig(tempo, tenantsCfg)
	assert.Equal(t, "80a845b34523484bf2ca89eaf19fa9fefbaacac1f1c12d5c3fbac7a2614b4c76", hash)
	assert.NotEmpty(t, secret.Data["tenants.yaml"])
}

func TestPatchOCPServingCerts(t *testing.T) {
	tempo := v1alpha1.TempoStack{}
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
