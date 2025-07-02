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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/status"
	"github.com/grafana/tempo-operator/internal/version"
)

func createSecret(t *testing.T, nsn types.NamespacedName) *corev1.Secret {
	storageSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		StringData: map[string]string{
			"endpoint":          "http://minio:9000",
			"bucket":            "tempo",
			"access_key_id":     "tempo-user",
			"access_key_secret": "abcd1234",
		},
	}
	err := k8sClient.Create(context.Background(), storageSecret)
	require.NoError(t, err)
	return storageSecret
}

func createTenantSecret(t *testing.T, nsn types.NamespacedName) *corev1.Secret {
	tenantSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		StringData: map[string]string{
			"clientID":     "da252601-01c1-44f2-8feb-edc38582c3a9",
			"clientSecret": "super-secret",
		},
	}
	err := k8sClient.Create(context.Background(), tenantSecret)
	require.NoError(t, err)
	return tenantSecret
}

func createTempoCR(t *testing.T, nsn types.NamespacedName, storageSecret *corev1.Secret) {
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{},
			},
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)
}

func TestReconcile(t *testing.T) {
	nsn := types.NamespacedName{Name: "reconcile-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
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
		list := &appsv1.DeploymentList{}
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

	// test status
	updatedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo)
	require.NoError(t, err)
	assert.Equal(t, "0.0.0", updatedTempo.Status.TempoVersion)

	// test status condition
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionReady),
		Status:             "True",
		LastTransitionTime: updatedTempo.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonReady),
		Message:            "All components are operational",
	}}, updatedTempo.Status.Conditions)
	// make sure LastTransitionTime is recent
	assert.InDelta(t, metav1.NewTime(time.Now()).Unix(), updatedTempo.Status.Conditions[0].LastTransitionTime.Unix(), 60)
}

func TestReadyToConfigurationError(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "ready-to-configerr-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcileResult, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Ready=true
	updatedTempo1 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo1)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionReady),
		Status:             "True",
		LastTransitionTime: updatedTempo1.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonReady),
		Message:            "All components are operational",
	}}, updatedTempo1.Status.Conditions)

	// Update the storage secret to an invalid endpoint
	storageSecret.Data["endpoint"] = []byte("invalid")
	err = k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// LastTransitionTime gets stored in seconds, therefore we need to wait a bit to verify that the time got updated
	time.Sleep(1 * time.Second)

	// Reconcile
	reconcileResult, err = reconciler.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "terminal error")

	// Verify status conditions: Ready=false, ConfigurationError=true
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionReady),
			Status:             "False",
			LastTransitionTime: updatedTempo2.Status.Conditions[0].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonReady),
			Message:            "All components are operational",
		},
		{
			Type:               string(v1alpha1.ConditionConfigurationError),
			Status:             "True",
			LastTransitionTime: updatedTempo2.Status.Conditions[1].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "\"endpoint\" field of storage secret must be a valid URL",
		},
	}, updatedTempo2.Status.Conditions)
	assert.Greater(t, updatedTempo2.Status.Conditions[0].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
	assert.Greater(t, updatedTempo2.Status.Conditions[1].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
}

func TestConfigurationErrorToConfigurationError(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "configerr-to-configerr-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Update the storage secret to an invalid endpoint
	storageSecret.Data["endpoint"] = []byte("invalid")
	err := k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "terminal error")

	// Verify status conditions: ConfigurationError=true
	updatedTempo1 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo1)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionConfigurationError),
		Status:             "True",
		LastTransitionTime: updatedTempo1.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "\"endpoint\" field of storage secret must be a valid URL",
	}}, updatedTempo1.Status.Conditions)

	// Remove access_key from the storage secret
	delete(storageSecret.Data, "access_key_id")
	err = k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	_, err = reconciler.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "terminal error")

	// Verify status conditions: ConfigurationError=true
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)

	// We don't want to compare LastTransitionTime because it could change
	lastTransitionTime := updatedTempo2.Status.Conditions[0].LastTransitionTime
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionConfigurationError),
			Status:             "True",
			LastTransitionTime: lastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "storage secret must contain \"access_key_id\" field, \"endpoint\" field of storage secret must be a valid URL",
		},
	}, updatedTempo2.Status.Conditions)
}

func TestConfigurationErrorToReady(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "configerr-to-ready-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Update the storage secret to an invalid endpoint
	storageSecret.Data["endpoint"] = []byte("invalid")
	err := k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcileResult, err := reconciler.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "terminal error")
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: ConfigurationError=true
	updatedTempo1 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo1)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionConfigurationError),
		Status:             "True",
		LastTransitionTime: updatedTempo1.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "\"endpoint\" field of storage secret must be a valid URL",
	}}, updatedTempo1.Status.Conditions)

	// Update the storage secret to a valid endpoint
	storageSecret.Data["endpoint"] = []byte("http://minio:9000")
	err = k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// LastTransitionTime gets stored in seconds, therefore we need to wait a bit to verify that the time got updated
	time.Sleep(1 * time.Second)

	// Reconcile
	reconcileResult, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Ready=true, ConfigurationError=false
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionConfigurationError),
			Status:             "False",
			LastTransitionTime: updatedTempo2.Status.Conditions[1].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "\"endpoint\" field of storage secret must be a valid URL",
		},
		{
			Type:               string(v1alpha1.ConditionReady),
			Status:             "True",
			LastTransitionTime: updatedTempo2.Status.Conditions[0].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonReady),
			Message:            "All components are operational",
		},
	}, updatedTempo2.Status.Conditions)
	assert.Greater(t, updatedTempo2.Status.Conditions[0].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
	assert.Greater(t, updatedTempo2.Status.Conditions[1].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
}

func TestReconcileGenericError(t *testing.T) {
	nsn := types.NamespacedName{Name: "reconcile-errors", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
				OpenShift: configv1alpha1.OpenShiftFeatureGates{
					OpenShiftRoute: true, // this will throw an error, as the CRD is not installed
				},
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	_, err := reconciler.Reconcile(context.Background(), req)
	require.Error(t, err)

	updatedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionFailed),
		Status:             "True",
		LastTransitionTime: updatedTempo.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonFailedReconciliation),
		Message:            updatedTempo.Status.Conditions[0].Message,
	}}, updatedTempo.Status.Conditions)
	assert.Contains(t, updatedTempo.Status.Conditions[0].Message, "error listing routes: no kind is registered for the type v1.RouteList")
}

func TestStorageCustomCA(t *testing.T) {
	nsn := types.NamespacedName{Name: "custom-ca", Namespace: "default"}
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}

	storageSecret := createSecret(t, nsn)
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{},
			},
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
				TLS: v1alpha1.TLSSpec{
					Enabled: true,
					CA:      "custom-ca",
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	_, err = reconciler.Reconcile(context.Background(), req)
	require.Error(t, err)
	updatedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionConfigurationError),
		Status:             "True",
		LastTransitionTime: updatedTempo.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "could not fetch ConfigMap: configmaps \"custom-ca\" not found",
	}}, updatedTempo.Status.Conditions)

	caConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ca",
			Namespace: nsn.Namespace,
		},
	}
	err = k8sClient.Create(context.Background(), caConfigMap)
	require.NoError(t, err)

	_, err = reconciler.Reconcile(context.Background(), req)
	require.Error(t, err)
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionConfigurationError),
		Status:             "True",
		LastTransitionTime: updatedTempo2.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "CA ConfigMap must contain a 'service-ca.crt' key",
	}}, updatedTempo2.Status.Conditions)

	caConfigMap.Data = map[string]string{
		"ca.crt": "test",
	}
	err = k8sClient.Update(context.Background(), caConfigMap)
	require.NoError(t, err)

	_, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	updatedTempo3 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo3)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionConfigurationError),
			Status:             "False",
			LastTransitionTime: updatedTempo3.Status.Conditions[0].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "CA ConfigMap must contain a 'service-ca.crt' key",
		},
		{
			Type:               string(v1alpha1.ConditionReady),
			Status:             "True",
			LastTransitionTime: updatedTempo3.Status.Conditions[0].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonReady),
			Message:            "All components are operational",
		},
	}, updatedTempo3.Status.Conditions)
}

func TestTLSEnable(t *testing.T) {
	nsn := types.NamespacedName{Name: "tls-enabled-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
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
		NamespacedName: nsn,
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   nsn.Name,
			"app.kubernetes.io/managed-by": "tempo-operator",
		}),
	}
	{
		list := &corev1.SecretList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
		names := []string{}
		for _, cm := range list.Items {
			names = append(names, cm.Name)
		}

		expectedNames := []string{
			"compactor-mtls",
			"distributor-mtls",
			"ingester-mtls",
			"querier-mtls",
			"query-frontend-mtls",
			"signing-ca",
		}
		for _, expected := range expectedNames {
			assert.Contains(t, names, fmt.Sprintf("tempo-%s-%s", nsn.Name, expected))

		}
	}
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
		names := []string{}
		for _, cm := range list.Items {
			names = append(names, cm.Name)
		}
		assert.Contains(t, names, fmt.Sprintf("tempo-%s-ca-bundle", nsn.Name))
	}
}

func TestPruneIngress(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "prune-ingress-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo:       "docker.io/grafana/tempo:1.5.0",
				TempoQuery:  "docker.io/grafana/tempo-query:1.5.0",
				JaegerQuery: "docker.io/jaegertracing/jaeger-query:1.60",
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
			},
			Template: v1alpha1.TempoTemplateSpec{
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						ServicesQueryDuration: &metav1.Duration{Duration: time.Hour},
						Enabled:               true,
						Ingress: v1alpha1.IngressSpec{
							Type: v1alpha1.IngressTypeIngress,
						},
					},
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcileResult, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify Ingress is created
	ingressNsn := types.NamespacedName{Name: "tempo-prune-ingress-test-query-frontend", Namespace: "default"}
	ingress := networkingv1.Ingress{}
	err = k8sClient.Get(context.Background(), ingressNsn, &ingress)
	require.NoError(t, err)

	// Disable Ingress in CR
	err = k8sClient.Get(context.Background(), nsn, tempo) // operator modified CR, fetch latest version
	require.NoError(t, err)
	tempo.Spec.Template.QueryFrontend.JaegerQuery.Ingress.Type = v1alpha1.IngressTypeNone
	err = k8sClient.Update(context.Background(), tempo)
	require.NoError(t, err)

	// Reconcile
	reconcileResult, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify Ingress got deleted
	err = k8sClient.Get(context.Background(), ingressNsn, &ingress)
	require.Error(t, err)
	require.True(t, apierrors.IsNotFound(err))
}

func TestK8SGatewaySecret(t *testing.T) {
	nsn := types.NamespacedName{Name: "ocp-mode", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	ttnsn := types.NamespacedName{Name: "test-tenant-secret", Namespace: "default"}
	tenantSecret := createTenantSecret(t, ttnsn)

	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						ServicesQueryDuration: &metav1.Duration{Duration: time.Hour},
						Enabled:               true,
					},
				},
			},
			Images: configv1alpha1.ImagesSpec{
				Tempo:        "docker.io/grafana/tempo:1.5.0",
				TempoQuery:   "docker.io/grafana/tempo-query:1.5.0",
				TempoGateway: "docker.io/observatorium/api:1.5.0",
				JaegerQuery:  "docker.io/jaegertracing/jaeger-query:1.60",
			},
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{},
			},
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeStatic,
				Authentication: []v1alpha1.AuthenticationSpec{
					{
						TenantName: "test-tenant-1",
						TenantID:   "5dfdb9c2-4ab9-448d-b7ff-8cd1c474524f",
						OIDC: &v1alpha1.OIDCSpec{
							Secret: &v1alpha1.TenantSecretSpec{
								Name: tenantSecret.Name,
							},
						},
					},
				},
				Authorization: &v1alpha1.AuthorizationSpec{
					Roles: []v1alpha1.RoleSpec{
						{
							Name: "read-write",
							Permissions: []v1alpha1.PermissionType{
								v1alpha1.Read,
								v1alpha1.Write,
							},
							Resources: []string{
								"logs",
								"metrics",
								"traces",
							},
							Tenants: []string{"test-tenant-1"},
						},
					},
					RoleBindings: []v1alpha1.RoleBindingsSpec{
						{
							Name:  "test-oidc",
							Roles: []string{"read-write"},
							Subjects: []v1alpha1.Subject{
								{
									Name: "user",
									Kind: "user",
								},
							},
						},
					},
				},
			},
		},
	}
	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			Gates: configv1alpha1.FeatureGates{
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
		NamespacedName: nsn,
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)
	gwnsn := types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway", nsn.Name), Namespace: "default"}
	got := &corev1.Secret{}
	err = k8sClient.Get(context.Background(), gwnsn, got)
	assert.NoError(t, err)

	assert.Contains(t, string(got.Data["tenants.yaml"]), string(tenantSecret.Data["clientID"]))
	assert.Contains(t, string(got.Data["tenants.yaml"]), string(tenantSecret.Data["clientSecret"]))
}

func TestOpenShiftMode_finalizer(t *testing.T) {
	namespaceName := strings.ReplaceAll(strings.ToLower(t.Name()), "_", "")
	tempoName := "tempo-gateway"
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
	err := k8sClient.Create(context.Background(), namespace)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, k8sClient.Delete(context.Background(), namespace))
	}()
	storageSecret := createSecret(t, types.NamespacedName{Name: "ocp-mode", Namespace: namespaceName})
	defer func() {
		require.NoError(t, k8sClient.Delete(context.Background(), storageSecret))
	}()

	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoName,
			Namespace: namespaceName,
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						ServicesQueryDuration: &metav1.Duration{Duration: time.Hour},
						Enabled:               true,
					},
				},
			},
			Images: configv1alpha1.ImagesSpec{
				Tempo:        "docker.io/grafana/tempo:1.5.0",
				TempoQuery:   "docker.io/grafana/tempo-query:1.5.0",
				TempoGateway: "docker.io/observatorium/api:1.5.0",
				JaegerQuery:  "docker.io/jaegertracing/jaeger-query:1.60",
			},
			LimitSpec: v1alpha1.LimitSpec{
				PerTenant: map[string]v1alpha1.RateLimitSpec{},
			},
			Retention: v1alpha1.RetentionSpec{
				PerTenant: map[string]v1alpha1.RetentionConfig{},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeOpenShift,
			},
		},
	}
	err = k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	reconciler := TempoStackReconciler{
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
	assert.True(t, apierrors.IsNotFound(err))
	err = k8sClient.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("tempo-%s-gateway-%s", tempoName, namespaceName), Namespace: "default"}, gatewayClusterRoleBinding)
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestReconcileManifestsValidateModes(t *testing.T) {
	nsn := types.NamespacedName{Name: "bar", Namespace: "default"}
	storageSecret := createSecret(t, nsn)

	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TempoStackSpec{
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
				QueryFrontend: v1alpha1.TempoQueryFrontendSpec{
					JaegerQuery: v1alpha1.JaegerQuerySpec{
						ServicesQueryDuration: &metav1.Duration{Duration: time.Hour},
						Enabled:               true,
					},
				},
			},
			Storage: v1alpha1.ObjectStorageSpec{
				Secret: v1alpha1.ObjectStorageSecretSpec{
					Name: storageSecret.Name,
					Type: "s3",
				},
			},
			Tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeStatic,
			},
		},
	}

	tt := []struct {
		name     string
		tenants  *v1alpha1.TenantsSpec
		validate func(t *testing.T, err error)
	}{
		{
			name: "static mode not configured the right way",
			tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeStatic,
			},
			validate: func(t *testing.T, err error) {
				require.Error(t, err)
				v, ok := err.(*status.ConfigurationError)
				if !ok {
					t.Fatal("invalid error type")
				}
				assert.Equal(t, v1alpha1.ReasonInvalidTenantsConfiguration, v.Reason)
			},
		},
		{
			name: "fail get tenant secrets",
			tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.ModeStatic,
			},
			validate: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t,
					"invalid configuration: Invalid tenants configuration: spec.tenants.authentication is required in static mode",
					err.Error(),
				)
			},
		},
	}

	err := k8sClient.Create(context.Background(), tempo)
	require.NoError(t, err)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := k8sClient.Update(context.Background(), tempo)
			require.NoError(t, err)
			reconciler := TempoStackReconciler{Client: k8sClient, Scheme: testScheme}
			err = reconciler.createOrUpdate(context.Background(), *tempo)
			tc.validate(t, err)
		})
	}
}

func TestUpgrade(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "upgrade-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(1),
		CtrlConfig: configv1alpha1.ProjectConfig{
			DefaultImages: configv1alpha1.ImagesSpec{
				Tempo: "docker.io/grafana/tempo:1.5.0",
			},
			Gates: configv1alpha1.FeatureGates{
				TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
			},
		},
		Version: version.Get(),
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcile, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	// Upgrade process of first reconcile detected an empty operator version in the status field of the CR
	// and updated the version to the current operator version (0.0.0)
	updatedTempo := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo)
	require.NoError(t, err)
	assert.Equal(t, "0.0.0", reconciler.Version.OperatorVersion)
	assert.Equal(t, reconciler.Version.OperatorVersion, updatedTempo.Status.OperatorVersion)

	// Bump operator version
	reconciler.Version.OperatorVersion = "100.0.0"

	// Reconcile should perform all upgrade steps until latest version
	reconcile, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcile.Requeue)

	// Verify CR is at latest version
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo)
	require.NoError(t, err)
	assert.Equal(t, "100.0.0", updatedTempo.Status.OperatorVersion)
}
