package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

var logger = log.Log.WithName("unit-tests")

func createTempoCR(t *testing.T, nsn types.NamespacedName, version string, managementState v1alpha1.ManagementStateType) *v1alpha1.TempoStack {
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "tempo-operator",
			},
		},
		Spec: v1alpha1.TempoStackSpec{
			ManagementState: managementState,
			Images: configv1alpha1.ImagesSpec{
				Tempo:           "docker.io/grafana/tempo:0.0.0",
				TempoQuery:      "docker.io/grafana/tempo-query:0.0.0",
				TempoGateway:    "quay.io/observatorium/api:0.0.0",
				TempoGatewayOpa: "quay.io/observatorium/opa-openshift:0.0.0",
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Type: v1alpha1.ObjectStorageSecretS3,
					Name: "storage-secret",
				},
			},
		},
	}

	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	tempo.Status.OperatorVersion = version
	err = k8sClient.Status().Update(context.Background(), tempo)
	require.NoError(t, err)

	return tempo
}

func TestUpgradeToLatest(t *testing.T) {
	nsn := types.NamespacedName{Name: "upgrade-to-latest-test", Namespace: "default"}
	createTempoCR(t, nsn, "0.0.1", v1alpha1.ManagementStateManaged)

	currentV := version.Get()
	currentV.OperatorVersion = "0.1.0"
	currentV.TempoVersion = "1.2.3"

	upgrade := &Upgrade{
		Client:   k8sClient,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo:           "docker.io/grafana/tempo:latest",
				TempoQuery:      "docker.io/grafana/tempo-query:latest",
				TempoGateway:    "quay.io/observatorium/api:latest",
				TempoGatewayOpa: "quay.io/observatorium/opa-openshift:latest",
			},
		},
		Version: currentV,
		Log:     logger,
	}
	err := upgrade.TempoStacks(context.Background())
	require.NoError(t, err)

	upgradedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &upgradedTempo)
	assert.NoError(t, err)
	assert.Equal(t, currentV.OperatorVersion, upgradedTempo.Status.OperatorVersion)
	assert.Equal(t, currentV.TempoVersion, upgradedTempo.Status.TempoVersion)

	// assert images were updated
	assert.Equal(t, "docker.io/grafana/tempo:latest", upgradedTempo.Spec.Images.Tempo)
	assert.Equal(t, "docker.io/grafana/tempo-query:latest", upgradedTempo.Spec.Images.TempoQuery)
	assert.Equal(t, "quay.io/observatorium/api:latest", upgradedTempo.Spec.Images.TempoGateway)
	assert.Equal(t, "quay.io/observatorium/opa-openshift:latest", upgradedTempo.Spec.Images.TempoGatewayOpa)
}

func TestSkipUpgrade(t *testing.T) {
	currentOperatorVersion := "5.0.0"
	tests := []struct {
		name            string
		version         string
		managementState v1alpha1.ManagementStateType
	}{
		// Skip upgrade if the in-cluster version of the CR is more recent than the operator version
		// For example, in case an old operator version got deployed by mistake
		{"newer-than-ours", "10.0.0", v1alpha1.ManagementStateManaged},

		// Do not perform upgrade and do not update any images
		{"up-to-date", "5.0.0", v1alpha1.ManagementStateManaged},

		// Ignore unparseable versions
		{"unparseable", "abc", v1alpha1.ManagementStateManaged},

		// Ignore unmanaged instances
		{"unmanaged", "1.0.0", v1alpha1.ManagementStateUnmanaged},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nsn := types.NamespacedName{Name: "upgrade-test-" + test.name, Namespace: "default"}
			originalTempo := createTempoCR(t, nsn, test.version, test.managementState)

			currentV := version.Get()
			currentV.OperatorVersion = currentOperatorVersion

			upgrade := &Upgrade{
				Client:   k8sClient,
				Recorder: record.NewFakeRecorder(1),
				CtrlConfig: configv1alpha1.ProjectConfig{
					DefaultImages: configv1alpha1.ImagesSpec{
						Tempo:           "docker.io/grafana/tempo:latest",
						TempoQuery:      "docker.io/grafana/tempo-query:latest",
						TempoGateway:    "quay.io/observatorium/api:latest",
						TempoGatewayOpa: "quay.io/observatorium/opa-openshift:latest",
					},
				},
				Version: currentV,
				Log:     logger,
			}
			err := upgrade.TempoStacks(context.Background())
			require.NoError(t, err)

			upgradedTempo := v1alpha1.TempoStack{}
			err = k8sClient.Get(context.Background(), nsn, &upgradedTempo)
			assert.NoError(t, err)
			assert.Equal(t, test.version, upgradedTempo.Status.OperatorVersion)

			// assert images were not updated
			assert.Equal(t, originalTempo.Spec.Images.Tempo, upgradedTempo.Spec.Images.Tempo)
			assert.Equal(t, originalTempo.Spec.Images.TempoQuery, upgradedTempo.Spec.Images.TempoQuery)
			assert.Equal(t, originalTempo.Spec.Images.TempoGateway, upgradedTempo.Spec.Images.TempoGateway)
			assert.Equal(t, originalTempo.Spec.Images.TempoGatewayOpa, upgradedTempo.Spec.Images.TempoGatewayOpa)
		})
	}
}
