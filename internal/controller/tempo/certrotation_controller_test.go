package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestCertRotationReconciler_SkipsDeletingTempoStack(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-stack-deleting", Namespace: "default"}
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:       nsn.Name,
			Namespace:  nsn.Namespace,
			Finalizers: []string{"test.finalizer/block-deletion"},
		},
		Spec: v1alpha1.TempoStackSpec{
			ManagementState: v1alpha1.ManagementStateManaged,
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "dummy",
					Type: v1alpha1.ObjectStorageSecretS3,
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	// Delete sets DeletionTimestamp but the finalizer prevents actual removal.
	err = k8sClient.Delete(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := CertRotationReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
				Enabled:        true,
				CACertValidity: metav1.Duration{Duration: time.Hour * 43830},
				CACertRefresh:  metav1.Duration{Duration: time.Hour * 35064},
				CertValidity:   metav1.Duration{Duration: time.Hour * 2160},
				CertRefresh:    metav1.Duration{Duration: time.Hour * 1728},
			},
		},
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify the certRotationRequiredAt annotation was NOT set
	updated := &v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, updated)
	require.NoError(t, err)
	assert.NotNil(t, updated.DeletionTimestamp, "resource should still have DeletionTimestamp set")
	_, exists := updated.Annotations["tempo.grafana.com/certRotationRequiredAt"]
	assert.False(t, exists, "certRotationRequiredAt annotation should not be set on a deleting resource")

	// Clean up: remove the finalizer so the resource can be garbage collected.
	updated.Finalizers = nil
	require.NoError(t, k8sClient.Update(context.Background(), updated))
}

func TestCertRotationMonolithicReconciler_SkipsDeletingTempoMonolithic(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-mono-deleting", Namespace: "default"}
	tempo := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:       nsn.Name,
			Namespace:  nsn.Namespace,
			Finalizers: []string{"test.finalizer/block-deletion"},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	// Delete sets DeletionTimestamp but the finalizer prevents actual removal.
	err = k8sClient.Delete(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := CertRotationMonolithicReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
				Enabled:        true,
				CACertValidity: metav1.Duration{Duration: time.Hour * 43830},
				CACertRefresh:  metav1.Duration{Duration: time.Hour * 35064},
				CertValidity:   metav1.Duration{Duration: time.Hour * 2160},
				CertRefresh:    metav1.Duration{Duration: time.Hour * 1728},
			},
		},
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify the certRotationRequiredAt annotation was NOT set
	updated := &v1alpha1.TempoMonolithic{}
	err = k8sClient.Get(context.Background(), nsn, updated)
	require.NoError(t, err)
	assert.NotNil(t, updated.DeletionTimestamp, "resource should still have DeletionTimestamp set")
	_, exists := updated.Annotations["tempo.grafana.com/certRotationRequiredAt"]
	assert.False(t, exists, "certRotationRequiredAt annotation should not be set on a deleting resource")

	// Clean up: remove the finalizer so the resource can be garbage collected.
	updated.Finalizers = nil
	require.NoError(t, k8sClient.Update(context.Background(), updated))
}

func TestCertRotationReconciler_NotFoundTempoStack(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-stack-nonexistent", Namespace: "default"}
	reconciler := CertRotationReconciler{
		Client: k8sClient,
		Scheme: testScheme,
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestCertRotationMonolithicReconciler_NotFoundTempoMonolithic(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-mono-nonexistent", Namespace: "default"}
	reconciler := CertRotationMonolithicReconciler{
		Client: k8sClient,
		Scheme: testScheme,
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestCertRotationReconciler_SkipsUnmanagedTempoStack(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-stack-unmanaged", Namespace: "default"}
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			ManagementState: v1alpha1.ManagementStateUnmanaged,
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: "dummy",
					Type: v1alpha1.ObjectStorageSecretS3,
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, k8sClient.Delete(context.Background(), tempo))
	}()

	reconciler := CertRotationReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
				Enabled:        true,
				CACertValidity: metav1.Duration{Duration: time.Hour * 43830},
				CACertRefresh:  metav1.Duration{Duration: time.Hour * 35064},
				CertValidity:   metav1.Duration{Duration: time.Hour * 2160},
				CertRefresh:    metav1.Duration{Duration: time.Hour * 1728},
			},
		},
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestCertRotationMonolithicReconciler_SkipsUnmanagedTempoMonolithic(t *testing.T) {
	nsn := types.NamespacedName{Name: "certrot-mono-unmanaged", Namespace: "default"}
	tempo := &v1alpha1.TempoMonolithic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoMonolithicSpec{
			Management: v1alpha1.ManagementStateUnmanaged,
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, k8sClient.Delete(context.Background(), tempo))
	}()

	reconciler := CertRotationMonolithicReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			BuiltInCertManagement: configv1alpha1.BuiltInCertManagement{
				Enabled:        true,
				CACertValidity: metav1.Duration{Duration: time.Hour * 43830},
				CACertRefresh:  metav1.Duration{Duration: time.Hour * 35064},
				CertValidity:   metav1.Duration{Duration: time.Hour * 2160},
				CertRefresh:    metav1.Duration{Duration: time.Hour * 1728},
			},
		},
	}
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: nsn})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}
