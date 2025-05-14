package controllers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/version"
)

func TestReconcileMonolithic(t *testing.T) {
	nsn := types.NamespacedName{Name: "sample", Namespace: "default"}
	tempo := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := TempoMonolithicReconciler{
		Client:     k8sClient,
		Scheme:     testScheme,
		CtrlConfig: configv1alpha1.DefaultProjectConfig(),
	}
	reconcile, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	// Check if objects of specific types were created and are managed by the operator
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   nsn.Name,
			"app.kubernetes.io/managed-by": "tempo-operator",
		}),
	}
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.StatefulSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
}

func TestOpenShiftModeMonolithic_finalizer(t *testing.T) {
	namespaceName := strings.ReplaceAll(strings.ToLower(t.Name()), "_", "")
	tempoName := "tempo-gateway"
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
	err := k8sClient.Create(context.Background(), namespace)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, k8sClient.Delete(context.Background(), namespace))
	}()

	tempo := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoName,
			Namespace: namespaceName,
		},
		Spec: v1alpha1.TempoMonolithicSpec{
			Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
				Enabled: true,
				TenantsSpec: v1alpha1.TenantsSpec{
					Mode: v1alpha1.ModeOpenShift,
					Authentication: []v1alpha1.AuthenticationSpec{
						{
							TenantName: "test",
							TenantID:   "test",
						},
					},
				},
			},

			Storage: &v1alpha1.MonolithicStorageSpec{
				Traces: v1alpha1.MonolithicTracesStorageSpec{
					Backend: v1alpha1.MonolithicTracesStorageBackendPV,
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := TempoMonolithicReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				TempoGatewayOpa: "opa:latest",
			},
			Gates: configv1alpha1.FeatureGates{
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					BaseDomain: "localhost",
				},
				BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
					Enabled: true,
					CACertValidity: metav1.Duration{
						Duration: time.Hour * 43830,
					},
					CACertRefresh: metav1.Duration{
						Duration: time.Hour * 35064,
					},
					CertValidity: metav1.Duration{
						Duration: time.Hour * 2160,
					},
					CertRefresh: metav1.Duration{
						Duration: time.Hour * 1728,
					},
				},
				HTTPEncryption: true,
				GRPCEncryption: true,
				TLSProfile:     string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Name: tempoName, Namespace: namespaceName},
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	gatewayClusterRole := &rbacv1.ClusterRole{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway-%s", tempoName, namespaceName), Namespace: "default"}, gatewayClusterRole)
	require.NoError(t, err)
	gatewayClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway-%s", tempoName, namespaceName), Namespace: "default"}, gatewayClusterRoleBinding)
	require.NoError(t, err)

	err = k8sClient.Delete(context.Background(), tempo)
	require.NoError(t, err)
	reconcile, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	// the cluster role should be deleted
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway-%s", tempoName, namespaceName), Namespace: "default"}, gatewayClusterRole)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway-%s", tempoName, namespaceName), Namespace: "default"}, gatewayClusterRoleBinding)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
}
