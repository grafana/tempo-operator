package gateway

import (
	"fmt"

	"net"
	"reflect"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
	"github.com/grafana/tempo-operator/internal/tlsprofile"
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
		CtrlConfig:          configv1alpha1.ProjectConfig{},
		TLSProfile:          tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantOIDCSecret{},
	}

	cfgOpts := NewConfigOptions(
		params.Tempo.Namespace,
		params.Tempo.Name,
		"",
		"",
		"tempostack",
		*params.Tempo.Spec.Tenants,
		params.GatewayTenantSecret,
		params.GatewayTenantsData,
	)
	secret, hash, err := NewRBACConfigMap(cfgOpts, "", "", map[string]string{})
	assert.NoError(t, err)
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
		CtrlConfig:          configv1alpha1.ProjectConfig{},
		TLSProfile:          tlsprofile.TLSProfileOptions{},
		GatewayTenantSecret: []*manifestutils.GatewayTenantOIDCSecret{},
	}

	cfgOpts := NewConfigOptions(
		params.Tempo.Namespace,
		params.Tempo.Name,
		"",
		"",
		"tempostack",
		*params.Tempo.Spec.Tenants,
		params.GatewayTenantSecret,
		params.GatewayTenantsData,
	)
	secret, hash, err := NewTenantsSecret(cfgOpts, "", "", map[string]string{})
	assert.NoError(t, err)
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
					Ingress: v1alpha1.IngressSpec{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.RouteSpec{
							Termination: v1alpha1.TLSRouteTerminationTypePassthrough,
						},
					},
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeOpenShift,
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
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
					OpenShiftRoute:      true,
					BaseDomain:          "domain",
				},
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
	assert.Equal(t, "tempo-gateway-opa", dep.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, "tempo-simplest-gateway", dep.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/ready",
				Port:   intstr.FromString(manifestutils.GatewayInternalHttpPortName),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		TimeoutSeconds:   1,
		PeriodSeconds:    5,
		FailureThreshold: 12,
	}, dep.Spec.Template.Spec.Containers[0].ReadinessProbe)
	assert.Equal(t, &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/live",
				Port:   intstr.FromString(manifestutils.GatewayInternalHttpPortName),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		TimeoutSeconds:   2,
		PeriodSeconds:    30,
		FailureThreshold: 10,
	}, dep.Spec.Template.Spec.Containers[0].LivenessProbe)

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

func TestPatchTraceReadEndpoint(t *testing.T) {
	tt := []struct {
		name        string
		inputParams manifestutils.Params
		inputPod    corev1.PodTemplateSpec
		expectPod   corev1.PodTemplateSpec
		expectErr   error
	}{
		{
			name: "with trace read endpoint",
			inputParams: manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name",
						Namespace: "default",
					},
					Spec: v1alpha1.TempoStackSpec{
						Template: v1alpha1.TempoTemplateSpec{
							QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
								JaegerQuery: v1alpha1.JaegerQuerySpec{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: containerNameTempoGateway,
							Args: []string{
								"--abc",
							},
						},
						{
							Name: "second",
							Args: []string{
								"--xyz",
							},
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: containerNameTempoGateway,
							Args: []string{
								"--abc",
								"--traces.read.endpoint=http://tempo-name-query-frontend.default.svc.cluster.local:16686",
							},
						},
						{
							Name: "second",
							Args: []string{
								"--xyz",
							},
						},
					},
				},
			},
		},
		{
			name: "without trace read endpoint",
			inputParams: manifestutils.Params{
				Tempo: v1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name",
						Namespace: "default",
					},
					Spec: v1alpha1.TempoStackSpec{
						Template: v1alpha1.TempoTemplateSpec{
							QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
								JaegerQuery: v1alpha1.JaegerQuerySpec{
									Enabled: false,
								},
							},
						},
					},
				},
			},
			inputPod: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: containerNameTempoGateway,
							Args: []string{
								"--abc",
							},
						},
					},
				},
			},
			expectPod: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: containerNameTempoGateway,
							Args: []string{
								"--abc",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pod, err := patchTraceReadEndpoint(tc.inputParams, tc.inputPod)
			require.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectPod, pod)
		})
	}
}

func TestTLSParameters(t *testing.T) {
	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
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
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						Enabled: true,
					},
				},
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
		},
	}

	// test with TLS
	objects, err := BuildGateway(manifestutils.Params{
		Tempo: tempo,
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				HTTPEncryption: true,
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					BaseDomain: "domain",
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 4, len(objects))
	obj := getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&appsv1.Deployment{}))
	require.NotNil(t, obj)

	dep, ok := obj.(*appsv1.Deployment)
	require.True(t, ok)

	args := dep.Spec.Template.Spec.Containers[0].Args
	assert.Contains(t, args, fmt.Sprintf("--tls.internal.server.key-file=%s/tls.key", manifestutils.TempoInternalTLSCertDir))
	assert.Contains(t, args, fmt.Sprintf("--tls.internal.server.cert-file=%s/tls.crt", manifestutils.TempoInternalTLSCertDir))
	assert.Contains(t, args, fmt.Sprintf("--traces.tls.key-file=%s/tls.key", manifestutils.TempoInternalTLSCertDir))
	assert.Contains(t, args, fmt.Sprintf("--traces.tls.cert-file=%s/tls.crt", manifestutils.TempoInternalTLSCertDir))
	assert.Contains(t, args, fmt.Sprintf("--traces.tls.ca-file=%s/service-ca.crt", manifestutils.TempoInternalTLSCADir))
	assert.Contains(t, args, fmt.Sprintf("--traces.tempo.endpoint=https://%s:%d",
		naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName), manifestutils.PortHTTPServer))
	assert.Equal(t, corev1.URISchemeHTTPS, dep.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme)

	// test without TLS
	objects, err = BuildGateway(manifestutils.Params{
		Tempo: tempo,
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				HTTPEncryption: false,
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					BaseDomain: "domain",
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 4, len(objects))
	obj = getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&appsv1.Deployment{}))
	require.NotNil(t, obj)

	dep, ok = obj.(*appsv1.Deployment)
	require.True(t, ok)

	args = dep.Spec.Template.Spec.Containers[0].Args
	assert.NotContains(t, args, fmt.Sprintf("--tls.internal.server.key-file=%s/tls.key", manifestutils.TempoInternalTLSCertDir))
	assert.NotContains(t, args, fmt.Sprintf("--tls.internal.server.cert-file=%s/tls.crt", manifestutils.TempoInternalTLSCertDir))
	assert.NotContains(t, args, fmt.Sprintf("--traces.tls.key-file=%s/tls.key", manifestutils.TempoInternalTLSCertDir))
	assert.NotContains(t, args, fmt.Sprintf("--traces.tls.cert-file=%s/tls.crt", manifestutils.TempoInternalTLSCertDir))
	assert.NotContains(t, args, fmt.Sprintf("--traces.tls.ca-file=%s/service-ca.crt", manifestutils.TempoInternalTLSCADir))
	assert.Contains(t, args, fmt.Sprintf("--traces.read.endpoint=http://%s:16686",
		naming.ServiceFqdn(tempo.Namespace, tempo.Name, manifestutils.QueryFrontendComponentName)))
	assert.Equal(t, corev1.URISchemeHTTP, dep.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme)
}

func TestIngress(t *testing.T) {
	objects, err := BuildGateway(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
					Ingress: v1alpha1.IngressSpec{
						Type: v1alpha1.IngressTypeIngress,
						Host: "jaeger.example.com",
						Annotations: map[string]string{
							"traefik.ingress.kubernetes.io/router.tls": "true",
						},
					},
				},
			},
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
	}})

	require.NoError(t, err)
	require.Equal(t, 5, len(objects))
	pathType := networkingv1.PathTypePrefix
	assert.Equal(t, &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, "test"),
			Namespace: "project1",
			Labels:    manifestutils.ComponentLabels("gateway", "test"),
			Annotations: map[string]string{
				"traefik.ingress.kubernetes.io/router.tls": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "jaeger.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: naming.Name(manifestutils.GatewayComponentName, "test"),
											Port: networkingv1.ServiceBackendPort{
												Name: "public",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, objects[3].(*networkingv1.Ingress))
}

func TestRoute(t *testing.T) {
	objects, err := BuildGateway(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "project1",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
					Ingress: v1alpha1.IngressSpec{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.RouteSpec{
							Termination: v1alpha1.TLSRouteTerminationTypeEdge,
						},
					},
				},
			},
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
	}})

	require.NoError(t, err)
	require.Equal(t, 5, len(objects))
	assert.Equal(t, &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.Name(manifestutils.GatewayComponentName, "test"),
			Namespace: "project1",
			Labels:    manifestutils.ComponentLabels("gateway", "test"),
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: naming.Name(manifestutils.GatewayComponentName, "test"),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("public"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}, objects[3].(*routev1.Route))
}

func TestOverrideResources(t *testing.T) {
	overrideResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}

	tempo := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simplest",
			Namespace: "observability",
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					TempoComponentSpec: v1alpha1.TempoComponentSpec{
						Resources: &overrideResources,
					},
					Enabled: true,
					Ingress: v1alpha1.IngressSpec{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.RouteSpec{
							Termination: v1alpha1.TLSRouteTerminationTypePassthrough,
						},
					},
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeOpenShift,
				Authentication: []v1alpha1.AuthenticationSpec{
					{
						TenantName: "dev",
						TenantID:   "abcd1",
					},
				},
			},
			Resources: v1alpha1.Resources{
				Total: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}

	objects, err := BuildGateway(manifestutils.Params{
		Tempo: tempo,
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					ServingCertsService: true,
					OpenShiftRoute:      true,
					BaseDomain:          "domain",
				},
			},
		},
	})

	require.NoError(t, err)
	obj := getObjectByTypeAndName(objects, "tempo-simplest-gateway", reflect.TypeOf(&appsv1.Deployment{}))
	require.NotNil(t, obj)
	dep, ok := obj.(*appsv1.Deployment)
	require.True(t, ok)
	assert.Equal(t, dep.Spec.Template.Spec.Containers[0].Resources, overrideResources)

}
