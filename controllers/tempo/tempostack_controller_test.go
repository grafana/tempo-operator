package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/status"
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
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
		},
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

func TestReadyToDegraded(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "ready-to-degraded-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
		},
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
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Ready=false, Degraded=true
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
			Type:               string(v1alpha1.ConditionDegraded),
			Status:             "True",
			LastTransitionTime: updatedTempo2.Status.Conditions[1].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "invalid storage secret: \"endpoint\" field of storage secret must be a valid URL",
		},
	}, updatedTempo2.Status.Conditions)
	assert.Greater(t, updatedTempo2.Status.Conditions[0].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
	assert.Greater(t, updatedTempo2.Status.Conditions[1].LastTransitionTime.UnixNano(), updatedTempo1.Status.Conditions[0].LastTransitionTime.UnixNano())
}

func TestDegradedToDegraded(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "degraded-to-degraded-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Update the storage secret to an invalid endpoint
	storageSecret.Data["endpoint"] = []byte("invalid")
	err := k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
		},
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcileResult, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Degraded=true
	updatedTempo1 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo1)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionDegraded),
		Status:             "True",
		LastTransitionTime: updatedTempo1.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "invalid storage secret: \"endpoint\" field of storage secret must be a valid URL",
	}}, updatedTempo1.Status.Conditions)

	// Remove access_key from the storage secret
	delete(storageSecret.Data, "access_key_id")
	err = k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	reconcileResult, err = reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Degraded=true
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)

	// We don't want to compare LastTransitionTime because it could change
	lastTransitionTime := updatedTempo2.Status.Conditions[0].LastTransitionTime
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionDegraded),
			Status:             "True",
			LastTransitionTime: lastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "invalid storage secret: storage secret must contain \"access_key_id\" field, \"endpoint\" field of storage secret must be a valid URL",
		},
	}, updatedTempo2.Status.Conditions)
}

func TestDegradedToReady(t *testing.T) {
	// Create object storage secret and Tempo CR
	nsn := types.NamespacedName{Name: "degraded-to-ready-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	// Update the storage secret to an invalid endpoint
	storageSecret.Data["endpoint"] = []byte("invalid")
	err := k8sClient.Update(context.Background(), storageSecret)
	require.NoError(t, err)

	// Reconcile
	reconciler := TempoStackReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
		},
	}
	req := ctrl.Request{
		NamespacedName: nsn,
	}
	reconcileResult, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, false, reconcileResult.Requeue)

	// Verify status conditions: Degraded=true
	updatedTempo1 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo1)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{{
		Type:               string(v1alpha1.ConditionDegraded),
		Status:             "True",
		LastTransitionTime: updatedTempo1.Status.Conditions[0].LastTransitionTime,
		Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
		Message:            "invalid storage secret: \"endpoint\" field of storage secret must be a valid URL",
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

	// Verify status conditions: Ready=true, Degraded=false
	updatedTempo2 := v1alpha1.TempoStack{}
	err = k8sClient.Get(context.Background(), nsn, &updatedTempo2)
	require.NoError(t, err)
	assert.Equal(t, []metav1.Condition{
		{
			Type:               string(v1alpha1.ConditionDegraded),
			Status:             "False",
			LastTransitionTime: updatedTempo2.Status.Conditions[1].LastTransitionTime,
			Reason:             string(v1alpha1.ReasonInvalidStorageConfig),
			Message:            "invalid storage secret: \"endpoint\" field of storage secret must be a valid URL",
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

func TestTLSEnable(t *testing.T) {
	nsn := types.NamespacedName{Name: "tls-enabled-test", Namespace: "default"}
	storageSecret := createSecret(t, nsn)
	createTempoCR(t, nsn, storageSecret)

	reconciler := TempoStackReconciler{
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
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
				Tempo:      "docker.io/grafana/tempo:1.5.0",
				TempoQuery: "docker.io/grafana/tempo-query:1.5.0",
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
						Enabled: true,
						Ingress: v1alpha1.JaegerQueryIngressSpec{
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
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
			TLSProfile: string(configv1alpha1.TLSProfileIntermediateType),
		},
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
	nsn := types.NamespacedName{Name: "foo", Namespace: "default"}
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
						Enabled: true,
					},
				},
			},
			Images: configv1alpha1.ImagesSpec{
				Tempo:        "docker.io/grafana/tempo:1.5.0",
				TempoQuery:   "docker.io/grafana/tempo-query:1.5.0",
				TempoGateway: "docker.io/observatorium/api:1.5.0",
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
				Mode: v1alpha1.Static,
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
		Client: k8sClient,
		Scheme: testScheme,
		FeatureGates: configv1alpha1.FeatureGates{
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
						Enabled: true,
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
				Mode: v1alpha1.Static,
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
				Mode: v1alpha1.Static,
			},
			validate: func(t *testing.T, err error) {
				require.Error(t, err)
				v, ok := err.(*status.DegradedError)
				if !ok {
					t.Fatal("invalid error type")
				}
				assert.Equal(t, v1alpha1.ReasonInvalidTenantsConfiguration, v.Reason)
			},
		},
		{
			name: "fail get tenant secrets",
			tenants: &v1alpha1.TenantsSpec{
				Mode: v1alpha1.Static,
			},
			validate: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Equal(t,
					"cluster degraded: Invalid tenants configuration: spec.tenants.authentication is required in static mode",
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
			req := ctrl.Request{NamespacedName: nsn}
			err = reconciler.reconcileManifests(context.Background(), logr.Discard(), req, *tempo)
			tc.validate(t, err)
		})
	}
}
