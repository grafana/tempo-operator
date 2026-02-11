package monolithic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestBuildAll(t *testing.T) {
	opts := Options{
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

	objects, err := BuildAll(opts)
	require.NoError(t, err)
	require.Len(t, objects, 4)
}

func TestIngestionServingCertName(t *testing.T) {
	tests := []struct {
		name             string
		multitenancy     *v1alpha1.MonolithicMultitenancySpec
		expectedCertName string
	}{
		{
			name:             "gateway disabled: use tempo serving cert",
			multitenancy:     nil,
			expectedCertName: "tempo-sample-serving-cert",
		},
		{
			name: "gateway enabled: use gateway serving cert",
			multitenancy: &v1alpha1.MonolithicMultitenancySpec{
				Enabled: true,
				TenantsSpec: v1alpha1.TenantsSpec{
					Authentication: []v1alpha1.AuthenticationSpec{
						{TenantName: "test"},
					},
				},
			},
			expectedCertName: "tempo-sample-gateway-serving-cert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				CtrlConfig: configv1alpha1.ProjectConfig{
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
									},
								},
							},
						},
						Multitenancy: tt.multitenancy,
					},
				},
			}

			objects, err := BuildAll(opts)
			require.NoError(t, err)

			sts := findStatefulSet(objects)
			require.NotNil(t, sts, "statefulset not found in BuildAll output")

			volume := findVolume(sts, tt.expectedCertName)
			assert.Equal(t, tt.expectedCertName, volume.Secret.SecretName)
		})
	}
}

func findStatefulSet(objects []client.Object) *appsv1.StatefulSet {
	for _, obj := range objects {
		if sts, ok := obj.(*appsv1.StatefulSet); ok {
			return sts
		}
	}
	return nil
}

func findVolume(sts *appsv1.StatefulSet, volumeName string) corev1.Volume {
	for _, v := range sts.Spec.Template.Spec.Volumes {
		if v.Name == volumeName {
			return v
		}
	}
	return corev1.Volume{}
}
