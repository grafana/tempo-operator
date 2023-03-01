package tlsprofile

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	openshiftconfigv1 "github.com/openshift/api/config/v1"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
)

// ErrGetProfileFromCluster return when failed to get the cluster security policy in openshift.
var ErrGetProfileFromCluster = errors.New("failed to get profile from cluster, using default TLS profile")

// Get the profile according to the features configuration, if the policy is invalid or is not specified (empty string) this
// should return the default profile (Intermediate), if openshift.ClusterTLSPolicy is enabled, it should get the profile
// from the cluster, if the cluster return a unknow profile this should return an error.
func Get(ctx context.Context, fg configv1alpha1.FeatureGates, c k8getter, log logr.Logger) (TLSProfileOptions, error) {
	var tlsProfileType openshiftconfigv1.TLSSecurityProfile
	var err error
	var returnedErr error
	// If ClusterTLSPolicy is enabled get the policy from the cluster
	if fg.OpenShift.ClusterTLSPolicy {
		tlsProfileType, err = getTLSProfileFromCluster(ctx, c)
		if err != nil {
			returnedErr = ErrGetProfileFromCluster
			tlsProfileType = getDefaultTLSSecurityProfile()
		}
	} else {
		tlsProfileType, err = getTLSSecurityProfile(configv1alpha1.TLSProfileType(fg.TLSProfile))
		if err != nil {
			log.Error(err, "failed to get security profile. will use default tls profile.")
			tlsProfileType = getDefaultTLSSecurityProfile()
		}
	}

	// Transform the policy type to concrete settings (ciphers and minVersion).
	tlsProfile, err := getTLSSettings(tlsProfileType)
	if err != nil {
		return TLSProfileOptions{}, err
	}

	return tlsProfile, returnedErr
}
