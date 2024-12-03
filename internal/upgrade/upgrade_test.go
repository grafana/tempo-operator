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

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
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
			Images:          configv1alpha1.ImagesSpec{},
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

func TestUpgradeTempoStackToLatest(t *testing.T) {
	nsn := types.NamespacedName{Name: "upgrade-to-latest-test", Namespace: "default"}
	original := createTempoCR(t, nsn, "0.0.1", v1alpha1.ManagementStateManaged)

	currentV := version.Get()
	currentV.OperatorVersion = "100.0.0"
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
	_, err := upgrade.Upgrade(context.Background(), original)
	require.NoError(t, err)

	upgradedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &upgradedTempo)
	assert.NoError(t, err)

	// assert versions were updated
	assert.Equal(t, currentV.OperatorVersion, upgradedTempo.Status.OperatorVersion)
	assert.Equal(t, currentV.TempoVersion, upgradedTempo.Status.TempoVersion)
}

func TestUpgradeTempoMonolithicToLatest(t *testing.T) {
	nsn := types.NamespacedName{Name: "upgrade-to-latest-test", Namespace: "default"}
	ctrlConfig := configv1alpha1.ProjectConfig{
		DefaultImages: configv1alpha1.ImagesSpec{
			Tempo: "docker.io/grafana/tempo:x.y.z",
		},
	}
	original := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoMonolithicSpec{},
	}
	err := k8sClient.Create(context.Background(), original)
	require.NoError(t, err)

	original.Status = v1alpha1.TempoMonolithicStatus{
		OperatorVersion: "0.0.0",
	}
	err = k8sClient.Status().Update(context.Background(), original)
	require.NoError(t, err)

	currentV := version.Get()
	currentV.OperatorVersion = "100.0.0"
	currentV.TempoVersion = "1.2.3"

	upgrade := &Upgrade{
		Client:     k8sClient,
		Recorder:   record.NewFakeRecorder(1),
		CtrlConfig: ctrlConfig,
		Version:    currentV,
		Log:        logger,
	}
	_, err = upgrade.Upgrade(context.Background(), original)
	require.NoError(t, err)

	upgradedTempo := v1alpha1.TempoMonolithic{}
	err = k8sClient.Get(context.Background(), nsn, &upgradedTempo)
	assert.NoError(t, err)

	// assert versions were updated
	assert.Equal(t, currentV.OperatorVersion, upgradedTempo.Status.OperatorVersion)
	assert.Equal(t, currentV.TempoVersion, upgradedTempo.Status.TempoVersion)
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
			_, _ = upgrade.Upgrade(context.Background(), originalTempo)

			upgradedTempo := v1alpha1.TempoStack{}
			err := k8sClient.Get(context.Background(), nsn, &upgradedTempo)
			assert.NoError(t, err)
			assert.Equal(t, test.version, upgradedTempo.Status.OperatorVersion)
		})
	}
}
