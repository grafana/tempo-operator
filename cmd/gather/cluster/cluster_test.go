package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/grafana/tempo-operator/cmd/gather/config"
)

func TestGetOperatorDeploymentUsesConfiguredNameAndNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	deploymentA := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-a",
			Namespace: "ns-a",
			Labels: map[string]string{
				"app.kubernetes.io/name": "tempo-operator-a",
			},
		},
	}
	deploymentB := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-b",
			Namespace: "ns-b",
			Labels: map[string]string{
				"app.kubernetes.io/name": "tempo-operator-b",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deploymentA, deploymentB).
		Build()

	c := NewCluster(&config.Config{
		OperatorName:      "tempo-operator-b",
		OperatorNamespace: "ns-b",
		KubernetesClient:  k8sClient,
	})

	deployment, err := c.getOperatorDeployment()
	require.NoError(t, err)
	require.Equal(t, "tempo-operator-b", deployment.Name)
	require.Equal(t, "ns-b", deployment.Namespace)
}

func TestGetOperatorDeploymentReturnsErrorForMultipleMatches(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	deploymentA := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-a",
			Namespace: "ns-a",
			Labels: map[string]string{
				"app.kubernetes.io/name": "tempo-operator",
			},
		},
	}
	deploymentB := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator-b",
			Namespace: "ns-b",
			Labels: map[string]string{
				"app.kubernetes.io/name": "tempo-operator",
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deploymentA, deploymentB).
		Build()

	c := NewCluster(&config.Config{
		OperatorName:     "tempo-operator",
		KubernetesClient: k8sClient,
	})

	_, err := c.getOperatorDeployment()
	require.EqualError(t, err, `found multiple operator deployments for app.kubernetes.io/name="tempo-operator": ns-a/tempo-operator-a, ns-b/tempo-operator-b`)
}

func TestGetOperatorLogsReturnsErrorWhenNoPodsFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tempo-operator",
			Namespace: "operators",
			Labels: map[string]string{
				"app.kubernetes.io/name": "tempo-operator",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "tempo-operator",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		Build()

	c := NewCluster(&config.Config{
		OperatorName:     "tempo-operator",
		KubernetesClient: k8sClient,
	})

	err := c.GetOperatorLogs()
	require.EqualError(t, err, "no pods found for operator deployment operators/tempo-operator")
}
