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

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

func TestGetInvalidOrEmptyTLSProfile(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: "",
	}

	cl := &clientStub{}

	options, err := Get(ctx, fg, cl)
	assert.Equal(t, err, ErrGetInvalidProfile)
	assert.Equal(t, TLSProfileOptions{}, options)
}

func TestGetSpecificProfile(t *testing.T) {
	ctx := context.Background()
	fg := configv1alpha1.FeatureGates{
		TLSProfile: string(configv1alpha1.TLSProfileOldType),
	}

	cl := &clientStub{}

	oldSettings, err := GetTLSSettings(openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileOldType,
	})
	require.NoError(t, err)

	options, err := Get(ctx, fg, cl)
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
	cl := &clientStub{}

	returnErr := apierrors.NewNotFound(schema.GroupResource{}, "something wasn't found")
	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(returnErr)

	options, err := Get(ctx, fg, cl)
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

	options, err := Get(ctx, fg, cl)

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
	cl := &clientStub{}
	cl.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		v := args.Get(2).(*openshiftconfigv1.APIServer)
		v.Spec.TLSSecurityProfile = &openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileType("wedontknowwhattodowiththis"),
		}
	})

	_, err := Get(ctx, fg, cl)

	cl.AssertExpectations(t)
	cl.AssertNumberOfCalls(t, "Get", 1)

	assert.Error(t, err)
}
