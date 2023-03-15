package tlsprofile

import (
	"context"
	"testing"

	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	options, err := Get(ctx, fg, cl, l)
	assert.Equal(t, err, ErrGetInvalidProfile)
	assert.Equal(t, TLSProfileOptions{}, options)
}

func TestGetSpecificProfile(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
	}
	l := log.FromContext(ctx)

	cl := &clientStub{}

	oldSettings, err := GetTLSSettings(openshiftconfigv1.TLSSecurityProfile{
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
	cl := &clientStub{}

	returnErr := apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(returnErr)

	options, err := Get(ctx, fg, cl, l)
	cl.AssertExpectations(t)
	cl.AssertNumberOfCalls(t, "Get", 1)

	assert.Equal(t, ErrGetProfileFromCluster, err)
	assert.Equal(t, TLSProfileOptions{}, options)
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
	cl := &clientStub{}
	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		v := args.Get(2).(*openshiftconfigv1.APIServer)
		v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileModernType,
		}
	})

	modernSettings, err := GetTLSSettings(openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileModernType,
	})
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl, l)

	cl.AssertExpectations(t)
	cl.AssertNumberOfCalls(t, "Get", 1)

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
	cl := &clientStub{}
	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		v := args.Get(2).(*openshiftconfigv1.APIServer)
		v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileType("wedontknowwhattodowiththis"),
		}
	})

	_, err := Get(ctx, fg, cl, l)

	cl.AssertExpectations(t)
	cl.AssertNumberOfCalls(t, "Get", 1)

	assert.Error(t, err)
}
