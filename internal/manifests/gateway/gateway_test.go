package gateway

import (
	"net"
	"reflect"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/tlsprofile"
)

func TestRbacConfig(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{
				Mode: "static",
				Authorization: &v1alpha1.AuthorizationSpec{
					RoleBindings: []v1alpha1.RoleBindingsSpec{
						{
							Name:  "test",
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
		StorageParams:       manifestutils.StorageParams{},
		ConfigChecksum:      "",
		Tempo:               tempo,
		Gates:               configv1alpha1.FeatureGates{},
		TLSProfile:          tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantOIDCSecret{},
	}

	cfgOpts := newOptions(params.Tempo, params.Gates.OpenShift.BaseDomain, params.GatewayTenantSecret, params.GatewayTenantsData)
	tenantsCfg, _, err := buildConfigFiles(cfgOpts)
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
							Name:  "test",
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
		StorageParams:       manifestutils.StorageParams{},
		ConfigChecksum:      "",
		Tempo:               tempo,
		Gates:               configv1alpha1.FeatureGates{},
		TLSProfile:          tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantOIDCSecret{},
	}

	cfgOpts := newOptions(params.Tempo, params.Gates.OpenShift.BaseDomain, params.GatewayTenantSecret, params.GatewayTenantsData)
	_, tenantsCfg, err := buildConfigFiles(cfgOpts)
	assert.NoError(t, err)

	secret, hash := tenantsConfig(tempo, tenantsCfg)
	assert.Equal(t, "80a845b34523484bf2ca89eaf19fa9fefbaacac1f1c12d5c3fbac7a2614b4c76", hash)
	assert.NotEmpty(t, secret.Data["tenants.yaml"])
}

func TestBuildGateway_openshift(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.OpenShift,
				Authentication: []v1alpha1.AuthenticationSpec{
					{
						TenantName: "dev",
						TenantID:   "abcd1",
					},
				},
			},
		},
	}
	objects, err := BuildGateway(manifestutils.Params{
		Tempo: tempo,
		Gates: configv1alpha1.FeatureGates{
			OpenShift: configv1alpha1.OpenShiftFeatureGates{
				ServingCertsService: true,
				OpenShiftRoute:      true,
				BaseDomain:          "domain",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 9, len(objects))
	obj := getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&appsv1.Deployment{}))
	require.NotNil(t, obj)
	dep, ok := obj.(*appsv1.Deployment)
	require.True(t, ok)
	assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
	assert.Equal(t, "opa", dep.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, "tempo-simplest-gateway", dep.Spec.Template.Spec.ServiceAccountName)

	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&corev1.ServiceAccount{}))
	require.NotNil(t, obj)

	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&rbacv1.ClusterRole{}))
	require.NotNil(t, obj)
	clusterRole, ok := obj.(*rbacv1.ClusterRole)
	require.True(t, ok)
	assert.Equal(t, 2, len(clusterRole.Rules))

	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&rbacv1.ClusterRoleBinding{}))
	require.NotNil(t, obj)
	clusterRoleBinding, ok := obj.(*rbacv1.ClusterRoleBinding)
	require.True(t, ok)
	require.Equal(t, 1, len(clusterRoleBinding.Subjects))
	require.Equal(t, "ServiceAccount", clusterRoleBinding.Subjects[0].Kind)
	require.Equal(t, "tempo-simplest-gateway", clusterRoleBinding.Subjects[0].Name)

	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&routev1.Route{}))
	require.NotNil(t, obj)
	route, ok := obj.(*routev1.Route)
	require.True(t, ok)
	require.Equal(t, "Service", route.Spec.To.Kind)
	require.Equal(t, "tempo-simplest-gateway", route.Spec.To.Name)

	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway-cabundle", reflect.TypeOf(&corev1.ConfigMap{}))
	require.NotNil(t, obj)
	caConfigMap, ok := obj.(*corev1.ConfigMap)
	require.True(t, ok)
	require.Equal(t, map[string]string{
		"service.beta.openshift.io/inject-cabundle": "true",
	}, caConfigMap.Annotations)
}

func getObjectByTypeAndName(objects []client.Object, name string, t reflect.Type) client.Object { // nolint: unparam
	for _, o := range objects {
		objType := reflect.TypeOf(o)
		if o.GetName() == name && objType == t {
			return o
		}
	}
	return nil
}

func TestPatchTracing(t *testing.T) {
	tt := []struct {
		name       string
		inputTempo v1alpha1.TempoStack
		inputPod   corev1.PodTemplateSpec
		expectPod  corev1.PodTemplateSpec
		expectErr  error
	}{
		{
			name: "valid settings",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction:    "1.0",
							JaegerAgentEndpoint: "agent:1234",
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Args: []string{
								"--abc",
							},
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com":                    "true",
						"sidecar.opentelemetry.io/inject": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
							Args: []string{
								"--abc",
								"--internal.tracing.endpoint=agent:1234",
								"--internal.tracing.endpoint-type=agent",
								"--internal.tracing.sampling-fraction=1.0",
							},
						},
					},
				},
			},
		},
		{
			name: "no sampling param",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction: "",
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"existing.com": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "first",
						},
					},
				},
			},
		},
		{
			name: "invalid agent address",
			inputTempo: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
							SamplingFraction:    "0.5",
							JaegerAgentEndpoint: "---invalid----",
						},
					},
				},
			},
			inputPod:  corev1.PodTemplateSpec{},
			expectPod: corev1.PodTemplateSpec{},
			expectErr: &net.AddrError{
				Addr: "---invalid----",
				Err:  "missing port in address",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pod, err := patchTracing(tc.inputTempo, tc.inputPod)
			require.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectPod, pod)
		})
	}
}
