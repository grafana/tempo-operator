package tlsprofile

import (
	"context"
	"errors"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

// ErrGetProfileFromCluster happens when failed to get the cluster security policy in openshift.
var ErrGetProfileFromCluster = errors.New("failed to get profile from cluster, using default TLS profile")

// ErrGetInvalidProfile happens when the profile is invalid or unknow.
var ErrGetInvalidProfile = errors.New("got invalid TLS profile from cluster, using default TLS profile")

// Get the profile according to the features configuration, if the policy is invalid or is not specified (empty string) this
// should return an error, if openshift.ClusterTLSPolicy is enabled, it should get the profile
// from the cluster, if the cluster return a unknow profile this should return an error.
func Get(ctx context.Context, fg configv1alpha1.FeatureGates, c k8getter) (TLSProfileOptions, error) {
	var tlsProfileType openshiftconfigv1.TLSSecurityProfile
	var err error
	var returnedErr error
	// If ClusterTLSPolicy is enabled get the policy from the cluster
	if fg.OpenShift.ClusterTLSPolicy {
		tlsProfileType, err = getTLSProfileFromCluster(ctx, c)
		if err != nil {
			return TLSProfileOptions{}, ErrGetProfileFromCluster
		}
	} else {
		tlsProfileType, err = getTLSSecurityProfile(configv1alpha1.TLSProfileType(fg.TLSProfile))
		if err != nil {
			return TLSProfileOptions{}, ErrGetInvalidProfile
		}
	}

	// Transform the policy type to concrete settings (ciphers and minVersion).
	tlsProfile, err := GetTLSSettings(tlsProfileType)
	if err != nil {
		return TLSProfileOptions{}, err
	}

	return tlsProfile, returnedErr
}
