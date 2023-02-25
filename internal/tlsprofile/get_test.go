package tlsprofile

import (
	"context"
	"testing"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
)

func TestGetInvalidOrEmptyTLSProfile(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: "",
	}
	l := log.FromContext(ctx)

	cl := &clientStub{}

	defaultSettings, err := getTLSSettings(getDefaultTLSSecurityProfile())
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl, l)
	assert.NoError(t, err)
	assert.Equal(t, defaultSettings, options)
}

func TestGetSpecificProfile(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
	}
	l := log.FromContext(ctx)

	cl := &clientStub{}

	oldSettings, err := getTLSSettings(openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileOldType,
	})
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl, l)
	assert.NoError(t, err)
	assert.Equal(t, oldSettings, options)
}

func TestGetWithClusterError(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
		OpenShift: configv1alpha1.OpenShiftFeatureGates{
			ClusterTLSPolicy: true,
		},
	}
	l := log.FromContext(ctx)

	cl := &clientStub{
		GetStub: func(_ context.Context, name types.NamespacedName, object client.Object, _ ...client.GetOption) error {
			return apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
		},
	}

	defaultSettings, err := getTLSSettings(getDefaultTLSSecurityProfile())
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl, l)
	assert.NoError(t, err)
	assert.Equal(t, defaultSettings, options)
}

func TestGetWithClusterPolicy(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
		OpenShift: configv1alpha1.OpenShiftFeatureGates{
			ClusterTLSPolicy: true,
		},
	}
	l := log.FromContext(ctx)

	cl := &clientStub{
		GetStub: func(_ context.Context, name types.NamespacedName, object client.Object, _ ...client.GetOption) error {
			switch v := object.(type) {
			case *openshiftconfigv1.APIServer:
				v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileModernType,
				}
			}
			return nil
		},
	}

	modernSettings, err := getTLSSettings(openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileModernType,
	})
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl, l)
	assert.NoError(t, err)
	assert.Equal(t, modernSettings, options)
}

func TestGetWithInvalidClusterPolicy(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
		OpenShift: configv1alpha1.OpenShiftFeatureGates{
			ClusterTLSPolicy: true,
		},
	}
	l := log.FromContext(ctx)

	cl := &clientStub{
		GetStub: func(_ context.Context, name types.NamespacedName, object client.Object, _ ...client.GetOption) error {
			switch v := object.(type) {
			case *openshiftconfigv1.APIServer:
				v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileType("wedontknowwhattodowiththis"),
				}
			}
			return nil
		},
	}
	_, err := Get(ctx, fg, cl, l)
	assert.Error(t, err)
}
