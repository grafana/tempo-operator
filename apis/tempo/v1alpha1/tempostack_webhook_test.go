package v1alpha1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func TestDefault(t *testing.T) {
	defaulter := &Defaulter{
		ctrlConfig: v1alpha1.ProjectConfig{
			DefaultImages: v1alpha1.ImagesSpec{
				Tempo:           "docker.io/grafana/tempo:x.y.z",
				TempoQuery:      "docker.io/grafana/tempo-query:x.y.z",
				TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
				TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.3",
			},
			Distribution: "upstream",
		},
	}
	defaultMaxSearch := 0
	defaultDefaultResultLimit := 20

	tests := []struct {
		input    *TempoStack
		expected *TempoStack
		name     string
	}{
		{
			name: "no action default values are provided",
			input: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TempoStackSpec{
					ReplicationFactor: 2,
					Images: v1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:1.2.3",
						TempoQuery:      "docker.io/grafana/tempo-query:1.2.3",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.4",
					},
					ServiceAccount: "tempo-test",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
				},
			},
			expected: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: TempoStackSpec{
					ReplicationFactor: 2,
					Images: v1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:1.2.3",
						TempoQuery:      "docker.io/grafana/tempo-query:1.2.3",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.4",
					},
					ServiceAccount: "tempo-test",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
					SearchSpec: SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: TempoTemplateSpec{
						Distributor: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
		},
		{
			name: "default values are set in the webhook",
			input: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expected: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: TempoStackSpec{
					ReplicationFactor: 1,
					Images: v1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:x.y.z",
						TempoQuery:      "docker.io/grafana/tempo-query:x.y.z",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.3",
					},
					ServiceAccount: "tempo-test",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					StorageSize: resource.MustParse("10Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
					SearchSpec: SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: TempoTemplateSpec{
						Distributor: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
		},
		{
			name: "use Edge TLS termination if unset",
			input: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: TempoStackSpec{
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: IngressTypeRoute,
								},
							},
						},
					},
				},
			},
			expected: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: TempoStackSpec{
					ReplicationFactor: 1,
					Images: v1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:x.y.z",
						TempoQuery:      "docker.io/grafana/tempo-query:x.y.z",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.3",
					},
					ServiceAccount: "tempo-test",
					Retention: RetentionSpec{
						Global: RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					StorageSize: resource.MustParse("10Gi"),
					LimitSpec: LimitSpec{
						Global: RateLimitSpec{
							Query: QueryLimit{
								MaxSearchBytesPerTrace: &defaultMaxSearch,
							},
						},
					},
					SearchSpec: SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: TempoTemplateSpec{
						Distributor: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "route",
									Route: JaegerQueryRouteSpec{
										Termination: "edge",
									},
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
			err := defaulter.Default(context.Background(), test.input)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.input)
		})
	}
}

func TestValidateStorageSecret(t *testing.T) {
	tempoAzure := TempoStack{
		Spec: TempoStackSpec{
			Storage: ObjectStorageSpec{
				Secret: ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "azure",
				},
			},
		},
	}
	tempoS3 := TempoStack{
		Spec: TempoStackSpec{
			Storage: ObjectStorageSpec{
				Secret: ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "s3",
				},
			},
		},
	}

	tempoUnknown := TempoStack{
		Spec: TempoStackSpec{
			Storage: ObjectStorageSpec{
				Secret: ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "unknown",
				},
			},
		},
	}

	tempoEmtpyType := TempoStack{
		Spec: TempoStackSpec{
			Storage: ObjectStorageSpec{
				Secret: ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "",
				},
			},
		},
	}

	type Test struct {
		name     string
		tempo    TempoStack
		input    corev1.Secret
		expected field.ErrorList
	}

	path := field.NewPath("spec").Child("storage").Child("secret")

	tests := []Test{
		{
			name:  "unknown secret type",
			tempo: tempoUnknown,
			input: corev1.Secret{
				Data: map[string][]byte{
					"container": []byte("container-test"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoUnknown.Spec.Storage.Secret, "unknown is not an allowed storage secret type"),
			},
		},
		{
			name:  "empty secret type",
			tempo: tempoEmtpyType,
			input: corev1.Secret{
				Data: map[string][]byte{
					"container": []byte("container-test"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoEmtpyType.Spec.Storage.Secret, "storage secret must specify the type"),
			},
		},
		{
			name:  "empty Azure Storage secret",
			tempo: tempoAzure,
			input: corev1.Secret{},
			expected: field.ErrorList{
				field.Invalid(path, tempoAzure.Spec.Storage.Secret, "storage secret is empty"),
			},
		},
		{
			name:  "missing or empty fields in Azure secret",
			tempo: tempoAzure,
			input: corev1.Secret{
				Data: map[string][]byte{
					"container":    []byte("container-test"),
					"account_name": []byte(""),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoAzure.Spec.Storage.Secret, "storage secret must contain \"account_name\" field"),
				field.Invalid(path, tempoAzure.Spec.Storage.Secret, "storage secret must contain \"account_key\" field"),
			},
		},
		{
			name:  "empty S3 secret",
			tempo: tempoS3,
			input: corev1.Secret{},
			expected: field.ErrorList{
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "storage secret is empty"),
			},
		},
		{
			name:  "missing or empty fields in S3 secret",
			tempo: tempoS3,
			input: corev1.Secret{
				Data: map[string][]byte{
					"bucket": []byte(""),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "storage secret must contain \"endpoint\" field"),
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "storage secret must contain \"bucket\" field"),
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "storage secret must contain \"access_key_id\" field"),
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "storage secret must contain \"access_key_secret\" field"),
			},
		},
		{
			name:  "invalid endpoint 'invalid'",
			tempo: tempoS3,
			input: corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "\"endpoint\" field of storage secret must be a valid URL"),
			},
		},
		{
			name:  "invalid endpoint '/invalid'",
			tempo: tempoS3,
			input: corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("/invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, tempoS3.Spec.Storage.Secret, "\"endpoint\" field of storage secret must be a valid URL"),
			},
		},
		{
			name:  "valid storage secret",
			tempo: tempoS3,
			input: corev1.Secret{
				Data: map[string][]byte{
					"endpoint":          []byte("http://minio.minio.svc:9000"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateStorageSecret(test.tempo, test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateReplicationFactor(t *testing.T) {
	validator := &validator{}
	path := field.NewPath("spec").Child("ReplicationFactor")

	tests := []struct {
		name     string
		expected field.ErrorList
		input    TempoStack
	}{
		{
			name: "no error replicas equal to floor(replication_factor/2) + 1",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(2),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "no error replicas greater than floor(replication_factor/2) + 1",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(3),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "error replicas less than floor(replication_factor/2) + 1",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						Ingester: TempoComponentSpec{
							Replicas: pointer.Int32(1),
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, 3,
					fmt.Sprintf("replica factor of %d requires at least %d ingester replicas", 3, 2),
				)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validator.validateReplicationFactor(test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateIngressAndRoute(t *testing.T) {
	path := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("ingress").Child("type")

	tests := []struct {
		name       string
		input      TempoStack
		ctrlConfig v1alpha1.ProjectConfig
		expected   field.ErrorList
	}{
		{
			name: "valid ingress configuration",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "ingress",
								},
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid route configuration",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "route",
								},
							},
						},
					},
				},
			},
			ctrlConfig: v1alpha1.ProjectConfig{
				Gates: v1alpha1.FeatureGates{
					OpenShift: v1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
			expected: nil,
		},
		{
			name: "ingress enabled but queryfrontend disabled",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: false,
								Ingress: JaegerQueryIngressSpec{
									Type: "ingress",
								},
							},
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					path,
					IngressTypeIngress,
					"Ingress cannot be enabled if jaegerQuery is disabled",
				),
			},
		},
		{
			name: "route enabled but route feature gate disabled",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "route",
								},
							},
						},
					},
				},
			},
			ctrlConfig: v1alpha1.ProjectConfig{
				Gates: v1alpha1.FeatureGates{
					OpenShift: v1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: false,
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					path,
					IngressTypeRoute,
					"Please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
				),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &validator{ctrlConfig: test.ctrlConfig}
			errs := validator.validateQueryFrontend(test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateGatewayAndJaegerQuery(t *testing.T) {
	path := field.NewPath("spec").Child("template").Child("gateway").Child("enabled")

	tests := []struct {
		name     string
		input    TempoStack
		expected field.ErrorList
	}{
		{
			name: "valid configuration enabled both",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: true,
						},
					},
					Tenants: &TenantsSpec{
						Mode: Static,
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid config disable gateway and enable jaegerQuery",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "route",
								},
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: false,
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid config disable both",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: false,
								Ingress: JaegerQueryIngressSpec{
									Type: "route",
								},
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: false,
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid config disable jaegerQuery",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: false,
								Ingress: JaegerQueryIngressSpec{
									Type: "ingress",
								},
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, true,
					"to use the gateway, please enable jaegerQuery",
				),
			},
		},
		{
			name: "invalid configuration, ingress and gateway enabled",
			input: TempoStack{
				Spec: TempoStackSpec{
					ReplicationFactor: 3,
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
								Ingress: JaegerQueryIngressSpec{
									Type: "ingress",
								},
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: true,
						},
					},
					Tenants: &TenantsSpec{
						Mode: Static,
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, true,
					"cannot enable gateway and jaeger query ingress at the same time, please use the Jaeger UI from the gateway",
				),
			},
		},
		{
			name: "invalid configuration, gateway enabled but no tenant configured",
			input: TempoStack{
				Spec: TempoStackSpec{
					Template: TempoTemplateSpec{
						QueryFrontend: TempoQueryFrontendSpec{
							JaegerQuery: JaegerQuerySpec{
								Enabled: true,
							},
						},
						Gateway: TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, true,
					"to enable the gateway, please configure tenants",
				),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &validator{ctrlConfig: v1alpha1.ProjectConfig{}}
			errs := validator.validateGateway(test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateTenantConfigs(t *testing.T) {
	tt := []struct {
		name    string
		input   TempoStack
		wantErr error
	}{
		{
			name: "missing tenants",
			input: TempoStack{
				Spec: TempoStackSpec{},
			},
		},
		{
			name: "another mode",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{},
				},
			},
		},
		{
			name: "static missing authentication",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode: Static,
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authentication is required in static mode"),
		},
		{
			name: "static missing authorization",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode:           Static,
						Authentication: []AuthenticationSpec{},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization is required in static mode"),
		},
		{
			name: "static missing roles",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode:           Static,
						Authorization:  &AuthorizationSpec{},
						Authentication: []AuthenticationSpec{},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization.roles is required in static mode"),
		},
		{
			name: "static missing role bindings",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode: Static,
						Authorization: &AuthorizationSpec{
							Roles: []RoleSpec{},
						},
						Authentication: []AuthenticationSpec{},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization.roleBindings is required in static mode"),
		},
		{
			name: "openshift: RBAC should not be defined",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode: OpenShift,
						Authorization: &AuthorizationSpec{
							Roles: []RoleSpec{},
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization should not be defined in openshift mode"),
		},
		{
			name: "openshift: OIDC should not be defined",
			input: TempoStack{
				Spec: TempoStackSpec{
					Tenants: &TenantsSpec{
						Mode: OpenShift,
						Authentication: []AuthenticationSpec{
							{
								OIDC: &OIDCSpec{},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authentication.oidc should not be defined in openshift mode"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantConfigs(tc.input)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestValidatorObservabilityTracingConfig(t *testing.T) {
	observabilityBase := field.NewPath("spec").Child("observability")
	metricsBase := observabilityBase.Child("metrics")
	tracingBase := observabilityBase.Child("tracing")

	tt := []struct {
		name       string
		input      TempoStack
		ctrlConfig v1alpha1.ProjectConfig
		expected   field.ErrorList
	}{
		{
			name: "not set",
			input: TempoStack{
				Spec: TempoStackSpec{},
			},
		},
		{
			name: "createServiceMonitors enabled and serviceMonitor feature gate set",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Metrics: MetricsConfigSpec{
							CreateServiceMonitors: true,
						},
					},
				},
			},
			ctrlConfig: v1alpha1.ProjectConfig{
				Gates: v1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			expected: nil,
		},
		{
			name: "createServiceMonitors enabled but serviceMonitor feature gate not set",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Metrics: MetricsConfigSpec{
							CreateServiceMonitors: true,
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					metricsBase.Child("createServiceMonitors"),
					true,
					"the prometheusOperator feature gate must be enabled to create ServiceMonitors for Tempo components",
				),
			},
		},
		{
			name: "enableAlerts enabled but enableAlerts feature gate not set",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Metrics: MetricsConfigSpec{
							CreatePrometheusRules: true,
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					metricsBase.Child("createPrometheusRules"),
					true,
					"the prometheusOperator feature gate must be enabled to create PrometheusRules for Tempo components",
				),
			},
		},
		{
			name: "enableAlerts enabled but enableAlerts feature gate set",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Metrics: MetricsConfigSpec{
							CreatePrometheusRules: true,
						},
					},
				},
			},
			ctrlConfig: v1alpha1.ProjectConfig{
				Gates: v1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			expected: nil,
		},
		{
			name: "sampling fraction not a float",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Tracing: TracingConfigSpec{
							SamplingFraction: "a",
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					tracingBase.Child("sampling_fraction"),
					"a",
					"strconv.ParseFloat: parsing \"a\": invalid syntax",
				),
			},
		},
		{
			name: "invalid jaeger agent address",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Tracing: TracingConfigSpec{
							SamplingFraction:    "0.5",
							JaegerAgentEndpoint: "--invalid--",
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					tracingBase.Child("jaeger_agent_endpoint"),
					"--invalid--",
					"address --invalid--: missing port in address",
				),
			},
		},
		{
			name: "valid configuration",
			input: TempoStack{
				Spec: TempoStackSpec{
					Observability: ObservabilitySpec{
						Tracing: TracingConfigSpec{
							SamplingFraction:    "0.5",
							JaegerAgentEndpoint: "agent:1234",
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{ctrlConfig: tc.ctrlConfig}
			assert.Equal(t, tc.expected, v.validateObservability(tc.input))
		})
	}
}

func TestValidatorValidate(t *testing.T) {

	gvType := metav1.TypeMeta{
		APIVersion: "testv1",
		Kind:       "something",
	}
	tt := []struct {
		name     string
		input    runtime.Object
		expected error
	}{
		{
			name:     "not a tempo object",
			input:    new(corev1.Pod),
			expected: apierrors.NewBadRequest("expected a TempoStack object but got *v1.Pod"),
		},
		{
			name: "pass all validators",
			input: &TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
				TypeMeta: gvType,
				Spec: TempoStackSpec{
					ServiceAccount: naming.DefaultServiceAccountName("test-obj"),
					Storage: ObjectStorageSpec{
						Secret: ObjectStorageSecretSpec{
							Name: "not-found",
						},
					},
					Template: TempoTemplateSpec{
						Ingester: TempoComponentSpec{
							Replicas: func(i int32) *int32 { return &i }(1),
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{ctrlConfig: v1alpha1.ProjectConfig{}, client: &k8sFake{}}
			_, err := v.validate(context.Background(), tc.input)
			assert.Equal(t, tc.expected, err)
		})
	}
}

func TestValidateName(t *testing.T) {

	longName := "tgqwkjwqkehkqjwhekjwqhekjhwkjehwkqjehkjqwhekjqwhekjqhwkjehkqwj" +
		"554678789021123234554678789021123234554678789021123234554678" +
		"tgqwkjwqkehkqjwhekjwqhekjhwkjehwkqjehkjqwhekjqwhekjqhwkjehkqwj"

	tt := []struct {
		name     string
		input    TempoStack
		expected error
	}{
		{
			name: "all good",
			input: TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
			},
		},
		{
			name: "too long",
			input: TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      longName,
					Namespace: "abc",
				},
			},
			expected: apierrors.NewInvalid((&TempoStack{}).GroupVersionKind().GroupKind(),
				longName, field.ErrorList{
					field.Invalid(
						field.NewPath("metadata").Child("name"),
						longName,
						fmt.Sprintf("must be no more than %d characters", maxLabelLength),
					)}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{ctrlConfig: v1alpha1.ProjectConfig{}, client: &k8sFake{}}
			v.validateStackName(tc.input)
		})
	}
}

type k8sFake struct {
	client.Client
}

func (*k8sFake) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return fmt.Errorf("mock: fails always")
}
