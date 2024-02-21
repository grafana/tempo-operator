package webhooks

import (
	"context"
	"fmt"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/handlers/storage"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

func TestDefault(t *testing.T) {
	defaulter := &Defaulter{
		ctrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo:           "docker.io/grafana/tempo:x.y.z",
				TempoQuery:      "docker.io/grafana/tempo-query:x.y.z",
				TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
				TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.3",
			},
			Distribution: "upstream",
		},
	}
	defaultDefaultResultLimit := 20

	tests := []struct {
		input    *v1alpha1.TempoStack
		expected *v1alpha1.TempoStack
		name     string
	}{
		{
			name: "no action default values are provided",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 2,
					Images: configv1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:1.2.3",
						TempoQuery:      "docker.io/grafana/tempo-query:1.2.3",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.4",
					},
					ServiceAccount: "tempo-test",
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: v1alpha1.LimitSpec{
						Global: v1alpha1.RateLimitSpec{
							Query: v1alpha1.QueryLimit{
								MaxSearchDuration: metav1.Duration{Duration: 1 * time.Hour},
							},
						},
					},
				},
			},
			expected: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 2,
					Images: configv1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:1.2.3",
						TempoQuery:      "docker.io/grafana/tempo-query:1.2.3",
						TempoGateway:    "docker.io/observatorium/gateway:1.2.3",
						TempoGatewayOpa: "docker.io/observatorium/opa-openshift:1.2.4",
					},
					ServiceAccount: "tempo-test",
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: time.Hour},
						},
					},
					StorageSize: resource.MustParse("1Gi"),
					LimitSpec: v1alpha1.LimitSpec{
						Global: v1alpha1.RateLimitSpec{
							Query: v1alpha1.QueryLimit{
								MaxSearchDuration: metav1.Duration{Duration: 1 * time.Hour},
							},
						},
					},
					SearchSpec: v1alpha1.SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: v1alpha1.TempoTemplateSpec{
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
							TLS: v1alpha1.TLSSpec{},
						},
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Querier: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
						},
					},
				},
			},
		},
		{
			name: "default values are set in the webhook",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expected: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 1,
					Images:            configv1alpha1.ImagesSpec{},
					ServiceAccount:    "tempo-test",
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					StorageSize: resource.MustParse("10Gi"),
					LimitSpec: v1alpha1.LimitSpec{
						Global: v1alpha1.RateLimitSpec{
							Query: v1alpha1.QueryLimit{
								MaxSearchDuration: metav1.Duration{Duration: 0},
							},
						},
					},
					SearchSpec: v1alpha1.SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: v1alpha1.TempoTemplateSpec{
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
							TLS: v1alpha1.TLSSpec{},
						},
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Querier: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
						},
					},
				},
			},
		},
		{
			name: "use Edge TLS termination if unset",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1alpha1.TempoStackSpec{
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: v1alpha1.IngressTypeRoute,
								},
							},
						},
					},
				},
			},
			expected: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by":   "tempo-operator",
						"tempo.grafana.com/distribution": "upstream",
					},
				},
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 1,
					Images:            configv1alpha1.ImagesSpec{},
					ServiceAccount:    "tempo-test",
					Retention: v1alpha1.RetentionSpec{
						Global: v1alpha1.RetentionConfig{
							Traces: metav1.Duration{Duration: 48 * time.Hour},
						},
					},
					StorageSize: resource.MustParse("10Gi"),
					SearchSpec: v1alpha1.SearchSpec{
						MaxDuration:        metav1.Duration{Duration: 0},
						DefaultResultLimit: &defaultDefaultResultLimit,
					},
					Template: v1alpha1.TempoTemplateSpec{
						Compactor: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
							TLS: v1alpha1.TLSSpec{},
						},
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						Querier: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							TempoComponentSpec: v1alpha1.TempoComponentSpec{
								Replicas: ptr.To(int32(1)),
							},
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "route",
									Route: v1alpha1.RouteSpec{
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
	tempoAzure := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "azure",
				},
			},
		},
	}
	tempoS3 := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "s3",
				},
			},
		},
	}

	tempoUnknown := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "unknown",
				},
			},
		},
	}

	tempoEmtpyType := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "",
				},
			},
		},
	}

	type Test struct {
		name     string
		tempo    v1alpha1.TempoStack
		input    corev1.Secret
		expected field.ErrorList
	}

	path := field.NewPath("spec").Child("storage").Child("secret")
	secretNamePath := path.Child("name")
	secretTypePath := path.Child("type")

	tests := []Test{
		{
			name:  "unknown secret type",
			tempo: tempoUnknown,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoUnknown.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"container": []byte("container-test"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretTypePath, tempoUnknown.Spec.Storage.Secret.Type, "unknown is not an allowed storage secret type"),
			},
		},
		{
			name:  "empty secret type",
			tempo: tempoEmtpyType,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoEmtpyType.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"container": []byte("container-test"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretTypePath, tempoEmtpyType.Spec.Storage.Secret.Type, "storage secret type is required"),
			},
		},
		{
			name:  "empty Azure Storage secret",
			tempo: tempoAzure,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoAzure.Spec.Storage.Secret.Name,
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoAzure.Spec.Storage.Secret.Name, "storage secret must contain \"container\" field"),
				field.Invalid(secretNamePath, tempoAzure.Spec.Storage.Secret.Name, "storage secret must contain \"account_name\" field"),
				field.Invalid(secretNamePath, tempoAzure.Spec.Storage.Secret.Name, "storage secret must contain \"account_key\" field"),
			},
		},
		{
			name:  "missing or empty fields in Azure secret",
			tempo: tempoAzure,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoAzure.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"container":    []byte("container-test"),
					"account_name": []byte(""),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoAzure.Spec.Storage.Secret.Name, "storage secret must contain \"account_name\" field"),
				field.Invalid(secretNamePath, tempoAzure.Spec.Storage.Secret.Name, "storage secret must contain \"account_key\" field"),
			},
		},
		{
			name:  "empty S3 secret",
			tempo: tempoS3,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoS3.Spec.Storage.Secret.Name,
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"endpoint\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"bucket\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"access_key_id\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"access_key_secret\" field"),
			},
		},
		{
			name:  "missing or empty fields in S3 secret",
			tempo: tempoS3,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoS3.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"bucket": []byte(""),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"endpoint\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"bucket\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"access_key_id\" field"),
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "storage secret must contain \"access_key_secret\" field"),
			},
		},
		{
			name:  "invalid endpoint 'invalid'",
			tempo: tempoS3,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoS3.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"endpoint":          []byte("invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "\"endpoint\" field of storage secret must be a valid URL"),
			},
		},
		{
			name:  "invalid endpoint '/invalid'",
			tempo: tempoS3,
			input: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: tempoS3.Spec.Storage.Secret.Name,
				},
				Data: map[string][]byte{
					"endpoint":          []byte("/invalid"),
					"bucket":            []byte("bucket"),
					"access_key_id":     []byte("id"),
					"access_key_secret": []byte("secret"),
				},
			},
			expected: field.ErrorList{
				field.Invalid(secretNamePath, tempoS3.Spec.Storage.Secret.Name, "\"endpoint\" field of storage secret must be a valid URL"),
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
			client := &k8sFake{secret: &test.input}
			_, errs := storage.GetStorageParamsForTempoStack(context.Background(), client, test.tempo)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateStorageCAConfigMap(t *testing.T) {
	path := field.NewPath("spec").Child("storage").Child("tls").Child("caName")
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "testsecret",
					Type: "s3",
				},
				TLS: v1alpha1.TLSSpec{
					Enabled: true,
					CA:      "custom-ca",
				},
			},
		},
	}
	storageSecret := corev1.Secret{
		Data: map[string][]byte{
			"endpoint":          []byte("http://minio.minio.svc:9000"),
			"bucket":            []byte("bucket"),
			"access_key_id":     []byte("id"),
			"access_key_secret": []byte("secret"),
		},
	}

	tests := []struct {
		name     string
		input    corev1.ConfigMap
		expected field.ErrorList
	}{
		{
			name: "missing cert key",
			input: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expected: field.ErrorList{
				field.Invalid(path, "test", "CA ConfigMap must contain a 'service-ca.crt' key"),
			},
		},
		{
			name: "valid configmap",
			input: corev1.ConfigMap{
				Data: map[string]string{
					"service-ca.crt": "test",
				},
			},
			expected: nil,
		},
		{
			name: "valid legacy configmap for backwards compatibility",
			input: corev1.ConfigMap{
				Data: map[string]string{
					"ca.crt": "test",
				},
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &k8sFake{secret: &storageSecret, configmap: &test.input}
			_, errs := storage.GetStorageParamsForTempoStack(context.Background(), client, tempo)
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
		input    v1alpha1.TempoStack
	}{
		{
			name: "no error replicas equal to floor(replication_factor/2) + 1",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(2)),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "no error replicas greater than floor(replication_factor/2) + 1",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(3)),
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "error replicas less than floor(replication_factor/2) + 1",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: ptr.To(int32(1)),
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

func TestValidateQueryFrontend(t *testing.T) {
	ingressTypePath := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("ingress").Child("type")
	prometheusEndpointPath := field.NewPath("spec").Child("template").Child("queryFrontend").Child("jaegerQuery").Child("monitorTab").Child("prometheusEndpoint")

	tests := []struct {
		name       string
		input      v1alpha1.TempoStack
		ctrlConfig configv1alpha1.ProjectConfig
		expected   field.ErrorList
	}{
		{
			name: "valid ingress configuration",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
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
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "route",
								},
							},
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: true,
					},
				},
			},
			expected: nil,
		},
		{
			name: "ingress enabled but queryfrontend disabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: false,
								Ingress: v1alpha1.IngressSpec{
									Type: "ingress",
								},
							},
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					ingressTypePath,
					v1alpha1.IngressTypeIngress,
					"Ingress cannot be enabled if jaegerQuery is disabled",
				),
			},
		},
		{
			name: "route enabled but route feature gate disabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "route",
								},
							},
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					OpenShift: configv1alpha1.OpenShiftFeatureGates{
						OpenShiftRoute: false,
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					ingressTypePath,
					v1alpha1.IngressTypeRoute,
					"Please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
				),
			},
		},
		{
			name: "monitor tab enabled, missing prometheus endpoint",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								MonitorTab: v1alpha1.JaegerQueryMonitor{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{},
			},
			expected: field.ErrorList{
				field.Invalid(
					prometheusEndpointPath,
					"",
					"Prometheus endpoint must be set when monitoring is enabled",
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
		input    v1alpha1.TempoStack
		expected field.ErrorList
	}{
		{
			name: "valid configuration enabled both",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
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
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid config disable gateway and enable jaegerQuery",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "route",
								},
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: false,
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid config disable both",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: false,
								Ingress: v1alpha1.IngressSpec{
									Type: "route",
								},
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: false,
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid configuration, ingress and gateway enabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "ingress",
								},
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
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
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
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
			},
			expected: field.ErrorList{
				field.Invalid(path, true,
					"to enable the gateway, please configure tenants",
				),
			},
		},
		{
			name: "valid ingress configuration",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
							Ingress: v1alpha1.IngressSpec{
								Type: "ingress",
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid route, feature gateway disabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
							Ingress: v1alpha1.IngressSpec{
								Type: "route",
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("template").Child("gateway").Child("ingress").Child("type"),
					v1alpha1.IngressType("route"),
					"please enable the featureGates.openshift.openshiftRoute feature gate to use Routes",
				),
			},
		},
		{
			name: "invalid configuration, enable two ingesss",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
								Ingress: v1alpha1.IngressSpec{
									Type: "ingress",
								},
							},
						},
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
							Ingress: v1alpha1.IngressSpec{
								Type: "ingress",
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					path,
					true,
					"cannot enable gateway and jaeger query ingress at the same time, please use the Jaeger UI from the gateway",
				),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &validator{ctrlConfig: configv1alpha1.ProjectConfig{}}
			errs := validator.validateGateway(test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestValidateTenantConfigs(t *testing.T) {
	tt := []struct {
		name    string
		input   v1alpha1.TempoStack
		wantErr error
	}{
		{
			name: "missing tenants",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{},
			},
		},
		{
			name: "another mode",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{},
				},
			},
		},
		{
			name: "static missing authentication",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authentication is required in static mode"),
		},
		{
			name: "static missing authorization",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode:           v1alpha1.ModeStatic,
						Authentication: []v1alpha1.AuthenticationSpec{},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization is required in static mode"),
		},
		{
			name: "static missing roles",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode:           v1alpha1.ModeStatic,
						Authorization:  &v1alpha1.AuthorizationSpec{},
						Authentication: []v1alpha1.AuthenticationSpec{},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization.roles is required in static mode"),
		},
		{
			name: "static missing role bindings",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
						Authorization: &v1alpha1.AuthorizationSpec{
							Roles: []v1alpha1.RoleSpec{},
						},
						Authentication: []v1alpha1.AuthenticationSpec{},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization.roleBindings is required in static mode"),
		},
		{
			name: "openshift: RBAC should not be defined",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeOpenShift,
						Authorization: &v1alpha1.AuthorizationSpec{
							Roles: []v1alpha1.RoleSpec{},
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization should not be defined in openshift mode"),
		},
		{
			name: "openshift: OIDC should not be defined",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeOpenShift,
						Authentication: []v1alpha1.AuthenticationSpec{
							{
								OIDC: &v1alpha1.OIDCSpec{},
							},
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authentication.oidc should not be defined in openshift mode"),
		},
		{
			name: "static: OIDC should be defined",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
						Authorization: &v1alpha1.AuthorizationSpec{
							Roles:        []v1alpha1.RoleSpec{},
							RoleBindings: []v1alpha1.RoleBindingsSpec{},
						},
						Authentication: []v1alpha1.AuthenticationSpec{
							{},
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
					},
				},
			},
			wantErr: fmt.Errorf("spec.tenants.authorization.oidc is required for each tenant in static mode"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantConfigs(tc.input.Spec.Tenants, tc.input.Spec.Template.Gateway.Enabled)
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
		input      v1alpha1.TempoStack
		ctrlConfig configv1alpha1.ProjectConfig
		expected   field.ErrorList
	}{
		{
			name: "not set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{},
			},
		},
		{
			name: "createServiceMonitors enabled and prometheusOperator feature gate set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Metrics: v1alpha1.MetricsConfigSpec{
							CreateServiceMonitors: true,
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			expected: nil,
		},
		{
			name: "createServiceMonitors enabled but prometheusOperator feature gate not set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Metrics: v1alpha1.MetricsConfigSpec{
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
			name: "createPrometheusRules enabled but prometheusOperator feature gate not set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Metrics: v1alpha1.MetricsConfigSpec{
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
			name: "createPrometheusRules and createServiceMonitors enabled and prometheusOperator feature gate set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Metrics: v1alpha1.MetricsConfigSpec{
							CreateServiceMonitors: true,
							CreatePrometheusRules: true,
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			expected: nil,
		},
		{
			name: "createPrometheusRules enabled but createServiceMonitors not enabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Metrics: v1alpha1.MetricsConfigSpec{
							CreatePrometheusRules: true,
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					PrometheusOperator: true,
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					metricsBase.Child("createPrometheusRules"),
					true,
					"the Prometheus rules alert based on collected metrics, therefore the createServiceMonitors feature must be enabled when enabling the createPrometheusRules feature",
				),
			},
		},
		{
			name: "sampling fraction not a float",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
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
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
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
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Tracing: v1alpha1.TracingConfigSpec{
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

func TestValidatorObservabilityGrafana(t *testing.T) {
	tt := []struct {
		name       string
		input      v1alpha1.TempoStack
		ctrlConfig configv1alpha1.ProjectConfig
		expected   field.ErrorList
	}{
		{
			name: "datasource not enabled",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{},
			},
			expected: nil,
		},
		{
			name: "datasource enabled, feature gate not set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Grafana: v1alpha1.GrafanaConfigSpec{
							CreateDatasource: true,
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					GrafanaOperator: false,
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("spec").Child("observability").Child("grafana").Child("createDatasource"),
					true,
					"the grafanaOperator feature gate must be enabled to create a Datasource for Tempo",
				),
			},
		},
		{
			name: "datasource enabled, feature gate set",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					Observability: v1alpha1.ObservabilitySpec{
						Grafana: v1alpha1.GrafanaConfigSpec{
							CreateDatasource: true,
						},
					},
				},
			},
			ctrlConfig: configv1alpha1.ProjectConfig{
				Gates: configv1alpha1.FeatureGates{
					GrafanaOperator: true,
				},
			},
			expected: nil,
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
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
				TypeMeta: gvType,
				Spec: v1alpha1.TempoStackSpec{
					ServiceAccount: naming.DefaultServiceAccountName("test-obj"),
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name: "not-found",
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: func(i int32) *int32 { return &i }(1),
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{ctrlConfig: configv1alpha1.ProjectConfig{}, client: &k8sFake{}}
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
		input    v1alpha1.TempoStack
		expected field.ErrorList
	}{
		{
			name: "all good",
			input: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
			},
		},
		{
			name: "too long",
			input: v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      longName,
					Namespace: "abc",
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("metadata").Child("name"),
					longName,
					fmt.Sprintf("must be no more than %d characters", maxLabelLength),
				)},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, validateName(tc.input.Name))
		})
	}
}

func TestValidateDeprecatedFields(t *testing.T) {
	zero := 0
	one := 1

	tt := []struct {
		name     string
		input    v1alpha1.TempoStack
		expected field.ErrorList
	}{
		{
			name:  "no deprecated fields",
			input: v1alpha1.TempoStack{},
		},
		{
			name: "deprecated global maxSearchBytesPerTrace set to 0",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					LimitSpec: v1alpha1.LimitSpec{
						Global: v1alpha1.RateLimitSpec{
							Query: v1alpha1.QueryLimit{
								MaxSearchBytesPerTrace: &zero,
							},
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "limits", "global", "query", "maxSearchBytesPerTrace"),
					&zero,
					"this field is deprecated and must be unset",
				),
			},
		},
		{
			name: "deprecated global maxSearchBytesPerTrace set to 1",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					LimitSpec: v1alpha1.LimitSpec{
						Global: v1alpha1.RateLimitSpec{
							Query: v1alpha1.QueryLimit{
								MaxSearchBytesPerTrace: &one,
							},
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "limits", "global", "query", "maxSearchBytesPerTrace"),
					&one,
					"this field is deprecated and must be unset",
				),
			},
		},
		{
			name: "deprecated per-tenant maxSearchBytesPerTrace set to 0",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					LimitSpec: v1alpha1.LimitSpec{
						PerTenant: map[string]v1alpha1.RateLimitSpec{
							"tenant1": {
								Query: v1alpha1.QueryLimit{
									MaxSearchBytesPerTrace: &zero,
								},
							},
						},
					},
				},
			},
			expected: field.ErrorList{
				field.Invalid(
					field.NewPath("spec.limits.perTenant[tenant1].query.maxSearchBytesPerTrace"),
					&zero,
					"this field is deprecated and must be unset",
				),
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{ctrlConfig: configv1alpha1.ProjectConfig{}}
			assert.Equal(t, tc.expected, v.validateDeprecatedFields(tc.input))
		})
	}
}

func TestValidateReceiverTLSAndGateway(t *testing.T) {
	tests := []struct {
		name     string
		input    v1alpha1.TempoStack
		expected field.ErrorList
	}{
		{
			name: "valid configuration disable both",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: false,
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TLS: v1alpha1.TLSSpec{
								Enabled: false,
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid configuration enable only gateway",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
							},
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TLS: v1alpha1.TLSSpec{
								Enabled: false,
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid configuration enable only receiver TLS",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: false,
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TLS: v1alpha1.TLSSpec{
								Enabled: true,
								Cert:    "my-cert",
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid configuration enable both",
			input: v1alpha1.TempoStack{
				Spec: v1alpha1.TempoStackSpec{
					ReplicationFactor: 3,
					Template: v1alpha1.TempoTemplateSpec{
						Gateway: v1alpha1.TempoGatewaySpec{
							Enabled: true,
						},
						QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
							JaegerQuery: v1alpha1.JaegerQuerySpec{
								Enabled: true,
							},
						},
						Distributor: v1alpha1.TempoDistributorSpec{
							TLS: v1alpha1.TLSSpec{
								Enabled: true,
							},
						},
					},
					Tenants: &v1alpha1.TenantsSpec{
						Mode: v1alpha1.ModeStatic,
					},
				},
			},
			expected: field.ErrorList{field.Invalid(
				field.NewPath("spec").Child("template").Child("gateway").Child("enabled"),
				true,
				"Cannot enable gateway and distributor TLS at the same time",
			)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &validator{ctrlConfig: configv1alpha1.ProjectConfig{}}
			errs := validator.validateGateway(test.input)
			assert.Equal(t, test.expected, errs)
		})
	}
}

func TestWarning(t *testing.T) {
	gvType := metav1.TypeMeta{
		APIVersion: "testv1",
		Kind:       "something",
	}

	tests := []struct {
		name     string
		input    runtime.Object
		expected admission.Warnings
		client   client.Client
	}{
		{
			name: "no secret exists",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
				TypeMeta: gvType,
				Spec: v1alpha1.TempoStackSpec{
					ServiceAccount: naming.DefaultServiceAccountName("test-obj"),
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name: "not-found",
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: func(i int32) *int32 { return &i }(1),
						},
					},
				},
			},
			client:   &k8sFake{},
			expected: admission.Warnings{"could not fetch Secret: mock: fails always"},
		},
		{
			name: "warning for use extra config",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
				TypeMeta: gvType,
				Spec: v1alpha1.TempoStackSpec{
					ServiceAccount: naming.DefaultServiceAccountName("test-obj"),
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name: "not-found",
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: func(i int32) *int32 { return &i }(1),
						},
					},
					ExtraConfig: &v1alpha1.ExtraConfigSpec{
						Tempo: v1.JSON{Raw: []byte("{}")},
					},
				},
			},
			client: &k8sFake{
				secret: &corev1.Secret{},
			},
			expected: admission.Warnings{
				"override tempo configuration could potentially break the stack, use it carefully",
			},
		},
		{
			name: "no extra config used",
			input: &v1alpha1.TempoStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-obj",
					Namespace: "abc",
				},
				TypeMeta: gvType,
				Spec: v1alpha1.TempoStackSpec{
					ServiceAccount: naming.DefaultServiceAccountName("test-obj"),
					Storage: v1alpha1.ObjectStorageSpec{
						Secret: v1alpha1.ObjectStorageSecretSpec{
							Name: "not-found",
						},
					},
					Template: v1alpha1.TempoTemplateSpec{
						Ingester: v1alpha1.TempoComponentSpec{
							Replicas: func(i int32) *int32 { return &i }(1),
						},
					},
				},
			},
			client: &k8sFake{
				secret: &corev1.Secret{},
			},
			expected: admission.Warnings{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := &validator{ctrlConfig: configv1alpha1.ProjectConfig{}, client: test.client}
			wrgs, _ := v.validate(context.Background(), test.input)
			assert.Equal(t, test.expected, wrgs)
		})
	}
}

type k8sFake struct {
	secret    *corev1.Secret
	configmap *corev1.ConfigMap
	client.Client
}

func (k *k8sFake) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	switch typed := obj.(type) {
	case *corev1.Secret:
		if k.secret != nil {
			k.secret.DeepCopyInto(typed)
			return nil
		}
	case *corev1.ConfigMap:
		if k.configmap != nil {
			k.configmap.DeepCopyInto(typed)
			return nil
		}
	}
	return fmt.Errorf("mock: fails always")
}
