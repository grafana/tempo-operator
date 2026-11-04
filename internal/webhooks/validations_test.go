package webhooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

type sarRecorder struct {
	allowed bool
	reviews []authorizationv1.SubjectAccessReview
	k8sFake
}

func (r *sarRecorder) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	sar := obj.(*authorizationv1.SubjectAccessReview)
	sar.Status.Allowed = r.allowed
	r.reviews = append(r.reviews, *sar)
	return nil
}

func newAdmissionContext() context.Context {
	return admission.NewContextWithRequest(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			UserInfo: authenticationv1.UserInfo{
				Username: "alice",
				Groups:   []string{"dev"},
				Extra:    map[string]authenticationv1.ExtraValue{"scopes": {"user:full"}},
			},
		},
	})
}

// pod overrides

var (
	env = []corev1.EnvVar{{
		Name: "SECRET",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "other-secret"}, Key: "password"},
		},
	}}
	envFrom = []corev1.EnvFromSource{{
		SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "other-secret"}},
	}}
)

func assertPodCreateSAR(t *testing.T, sar authorizationv1.SubjectAccessReview) {
	assert.Equal(t, "alice", sar.Spec.User)
	assert.Equal(t, []string{"dev"}, sar.Spec.Groups)
	assert.Equal(t, map[string]authorizationv1.ExtraValue{"scopes": {"user:full"}}, sar.Spec.Extra)
	assert.Equal(t, &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Resource:  "pods",
		Verb:      "create",
	}, sar.Spec.ResourceAttributes)
}

func TestValidatePodOverrides(t *testing.T) {
	newTempo := func(images configv1alpha1.ImagesSpec, sa string) v1alpha1.TempoStack {
		return v1alpha1.TempoStack{
			ObjectMeta: metav1.ObjectMeta{Name: "simplest", Namespace: "ns1"},
			Spec: v1alpha1.TempoStackSpec{
				Images:         images,
				ServiceAccount: sa,
			},
		}
	}
	defaultSA := naming.DefaultServiceAccountName("simplest")
	customImages := configv1alpha1.ImagesSpec{Tempo: "evil/image:latest"}
	defaultImages := configv1alpha1.ImagesSpec{
		Tempo:           "docker.io/grafana/tempo:x.y.z",
		TempoQuery:      "docker.io/grafana/tempo-query:x.y.z",
		JaegerQuery:     "docker.io/jaegertracing/jaeger-query:x.y.z",
		TempoGateway:    "quay.io/observatorium/api:x.y.z",
		TempoGatewayOpa: "quay.io/observatorium/opa-openshift:x.y.z",
		OauthProxy:      "quay.io/openshift/origin-oauth-proxy:x.y.z",
	}

	withSpec := func(tempo v1alpha1.TempoStack, fn func(spec *v1alpha1.TempoStackSpec)) v1alpha1.TempoStack {
		fn(&tempo.Spec)
		return tempo
	}

	tests := []struct {
		name        string
		tempo       v1alpha1.TempoStack
		allowed     bool
		expectedSAR bool
		expected    []string
	}{
		{
			name:        "defaults",
			tempo:       newTempo(configv1alpha1.ImagesSpec{}, defaultSA),
			expectedSAR: false,
		},
		{
			name:        "custom image, allowed",
			tempo:       newTempo(customImages, defaultSA),
			allowed:     true,
			expectedSAR: true,
		},
		{
			name:        "custom image, denied",
			tempo:       newTempo(customImages, defaultSA),
			expectedSAR: true,
			expected:    []string{"spec.images.tempo"},
		},
		{
			name:        "default images set explicitly",
			tempo:       newTempo(defaultImages, defaultSA),
			expectedSAR: false,
		},
		{
			name: "default and custom images, denied",
			tempo: newTempo(configv1alpha1.ImagesSpec{
				Tempo:      defaultImages.Tempo,
				TempoQuery: "evil/tempo-query:latest",
				OauthProxy: "evil/oauth-proxy:latest",
			}, defaultSA),
			expectedSAR: true,
			expected:    []string{"spec.images.tempoQuery", "spec.images.oauthProxy"},
		},
		{
			name:        "custom service account, denied",
			tempo:       newTempo(configv1alpha1.ImagesSpec{}, "privileged"),
			expectedSAR: true,
			expected:    []string{"spec.serviceAccount"},
		},
		{
			name:        "custom image and service account, denied",
			tempo:       newTempo(customImages, "privileged"),
			expectedSAR: true,
			expected:    []string{"spec.images.tempo", "spec.serviceAccount"},
		},
		{
			name: "env and envFrom, allowed",
			tempo: withSpec(newTempo(configv1alpha1.ImagesSpec{}, defaultSA), func(spec *v1alpha1.TempoStackSpec) {
				spec.Env = env
				spec.EnvFrom = envFrom
			}),
			allowed:     true,
			expectedSAR: true,
		},
		{
			name: "env and envFrom, denied",
			tempo: withSpec(newTempo(configv1alpha1.ImagesSpec{}, defaultSA), func(spec *v1alpha1.TempoStackSpec) {
				spec.Env = env
				spec.EnvFrom = envFrom
			}),
			expectedSAR: true,
			expected:    []string{"spec.env", "spec.envFrom"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := newAdmissionContext()
			recorder := &sarRecorder{allowed: test.allowed}
			v := &validator{client: recorder, ctrlConfig: configv1alpha1.ProjectConfig{DefaultImages: defaultImages}}

			errs := v.validatePodOverrides(ctx, test.tempo)

			var paths []string
			for _, err := range errs {
				assert.Equal(t, field.ErrorTypeForbidden, err.Type)
				paths = append(paths, err.Field)
			}
			assert.Equal(t, test.expected, paths)

			if !test.expectedSAR {
				assert.Empty(t, recorder.reviews)
				return
			}
			require.Len(t, recorder.reviews, 1)
			assertPodCreateSAR(t, recorder.reviews[0])
		})
	}
}

func TestValidateMonolithicPodOverrides(t *testing.T) {
	newTempo := func(fn func(spec *v1alpha1.TempoMonolithicSpec)) v1alpha1.TempoMonolithic {
		tempo := v1alpha1.TempoMonolithic{
			ObjectMeta: metav1.ObjectMeta{Name: "simplest", Namespace: "ns1"},
		}
		fn(&tempo.Spec)
		return tempo
	}

	tests := []struct {
		name        string
		tempo       v1alpha1.TempoMonolithic
		allowed     bool
		expectedSAR bool
		expected    []string
	}{
		{
			name:        "defaults",
			tempo:       newTempo(func(spec *v1alpha1.TempoMonolithicSpec) {}),
			expectedSAR: false,
		},
		{
			name: "custom service account, env and envFrom, allowed",
			tempo: newTempo(func(spec *v1alpha1.TempoMonolithicSpec) {
				spec.ServiceAccount = "privileged"
				spec.Env = env
				spec.EnvFrom = envFrom
			}),
			allowed:     true,
			expectedSAR: true,
		},
		{
			name: "custom service account, denied",
			tempo: newTempo(func(spec *v1alpha1.TempoMonolithicSpec) {
				spec.ServiceAccount = "privileged"
			}),
			expectedSAR: true,
			expected:    []string{"spec.serviceAccount"},
		},
		{
			name: "custom service account, env and envFrom, denied",
			tempo: newTempo(func(spec *v1alpha1.TempoMonolithicSpec) {
				spec.ServiceAccount = "privileged"
				spec.Env = env
				spec.EnvFrom = envFrom
			}),
			expectedSAR: true,
			expected:    []string{"spec.serviceAccount", "spec.env", "spec.envFrom"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &sarRecorder{allowed: test.allowed}
			v := &monolithicValidator{client: recorder}

			errs := v.validatePodOverrides(newAdmissionContext(), test.tempo)

			var paths []string
			for _, err := range errs {
				assert.Equal(t, field.ErrorTypeForbidden, err.Type)
				paths = append(paths, err.Field)
			}
			assert.Equal(t, test.expected, paths)

			if !test.expectedSAR {
				assert.Empty(t, recorder.reviews)
				return
			}
			require.Len(t, recorder.reviews, 1)
			assertPodCreateSAR(t, recorder.reviews[0])
		})
	}
}

func TestValidateCreateCustomImagesDenied(t *testing.T) {
	ctx := admission.NewContextWithRequest(context.Background(), admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{UserInfo: authenticationv1.UserInfo{Username: "alice"}},
	})
	v := &validator{client: &sarRecorder{allowed: false}}
	tempo := &v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{Name: "simplest", Namespace: "ns1"},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{Tempo: "evil/image:latest"},
		},
	}
	require.NoError(t, NewDefaulter(configv1alpha1.ProjectConfig{}).Default(ctx, tempo))

	_, err := v.ValidateCreate(ctx, tempo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec.images.tempo: Forbidden")
	assert.Contains(t, err.Error(), "user alice does not have permission to create pods in namespace ns1")
}

// OpenShift tenancy mode

func assertTokenReviewSAR(t *testing.T, recorder *sarRecorder) {
	require.Len(t, recorder.reviews, 1)
	sar := recorder.reviews[0]
	assert.Equal(t, "alice", sar.Spec.User)
	assert.Equal(t, []string{"dev"}, sar.Spec.Groups)
	assert.Equal(t, map[string]authorizationv1.ExtraValue{"scopes": {"user:full"}}, sar.Spec.Extra)
	assert.Equal(t, &authorizationv1.ResourceAttributes{
		Group:    "authentication.k8s.io",
		Resource: "tokenreviews",
		Verb:     "create",
	}, sar.Spec.ResourceAttributes)
}

func TestValidateGatewayOpenShiftModeRBACDenied(t *testing.T) {
	recorder := &sarRecorder{allowed: false}
	err := validateGatewayOpenShiftModeRBAC(newAdmissionContext(), recorder)

	require.EqualError(t, err, "user alice does not have permission to create tokenreviews.authentication.k8s.io")
	assertTokenReviewSAR(t, recorder)
}

func TestValidateGatewayOpenShiftModeDenied(t *testing.T) {
	recorder := &sarRecorder{allowed: false}
	v := &validator{client: recorder}
	tempo := v1alpha1.TempoStack{
		Spec: v1alpha1.TempoStackSpec{
			Tenants: &v1alpha1.TenantsSpec{Mode: v1alpha1.ModeOpenShift},
			Template: v1alpha1.TempoTemplateSpec{
				Gateway: v1alpha1.TempoGatewaySpec{Enabled: true},
			},
		},
	}

	_, errs := v.validateGateway(newAdmissionContext(), tempo)

	assert.Equal(t, field.ErrorList{field.Forbidden(
		field.NewPath("spec", "tenants", "mode"),
		"Cannot enable OpenShift tenancy mode: user alice does not have permission to create tokenreviews.authentication.k8s.io",
	)}, errs)
	assertTokenReviewSAR(t, recorder)
}

func TestValidateMonolithicMultitenancyOpenShiftModeDenied(t *testing.T) {
	recorder := &sarRecorder{allowed: false}
	v := &monolithicValidator{client: recorder}
	tempo := v1alpha1.TempoMonolithic{
		Spec: v1alpha1.TempoMonolithicSpec{
			Multitenancy: &v1alpha1.MonolithicMultitenancySpec{
				Enabled: true,
				TenantsSpec: v1alpha1.TenantsSpec{
					Mode:           v1alpha1.ModeOpenShift,
					Authentication: []v1alpha1.AuthenticationSpec{{TenantName: "dev", TenantID: "1610b0c3-c509-4592-a256-a1871353dbfa"}},
				},
			},
		},
	}

	_, errs := v.validateMultitenancy(newAdmissionContext(), tempo)

	assert.Equal(t, field.ErrorList{field.Forbidden(
		field.NewPath("spec", "multitenancy", "mode"),
		"Cannot enable OpenShift tenancy mode: user alice does not have permission to create tokenreviews.authentication.k8s.io",
	)}, errs)
	assertTokenReviewSAR(t, recorder)
}
