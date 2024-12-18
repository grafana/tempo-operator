package tlsprofile

import (
	"context"

	"github.com/ViaQ/logerr/v2/kverrors"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/crypto"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

// APIServerName is the apiserver resource name used to fetch it.
const APIServerName = "cluster"

// GetTLSProfileFromCluster get the TLS profile from the cluster, if it's defined.
func getTLSProfileFromCluster(ctx context.Context, c k8getter) (openshiftconfigv1.TLSSecurityProfile, error) {
	var apiServer openshiftconfigv1.APIServer
	if err := c.Get(ctx, client.ObjectKey{Name: APIServerName}, &apiServer); err != nil {
		return openshiftconfigv1.TLSSecurityProfile{}, kverrors.Wrap(err, "failed to lookup openshift apiServer")
	}
	return *apiServer.Spec.TLSSecurityProfile, nil
}

// GetTLSSecurityProfile gets the tls profile info to apply.
func getTLSSecurityProfile(tlsProfileType configv1alpha1.TLSProfileType) (openshiftconfigv1.TLSSecurityProfile, error) {
	switch tlsProfileType {
	case configv1alpha1.TLSProfileOldType:
		return openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileOldType,
		}, nil
	case configv1alpha1.TLSProfileIntermediateType:
		return openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileIntermediateType,
		}, nil
	case configv1alpha1.TLSProfileModernType:
		return openshiftconfigv1.TLSSecurityProfile{
			Type: openshiftconfigv1.TLSProfileModernType,
		}, nil
	default:
		return openshiftconfigv1.TLSSecurityProfile{}, kverrors.New("unable to determine tls profile settings %s", tlsProfileType)
	}
}

// GetDefaultTLSSecurityProfile get the default tls profile settings if none is specified.
func GetDefaultTLSSecurityProfile() openshiftconfigv1.TLSSecurityProfile {
	return openshiftconfigv1.TLSSecurityProfile{
		Type: openshiftconfigv1.TLSProfileIntermediateType,
	}
}

// GetTLSSettings get the tls settings that belongs to the TLS profile specifications.
func GetTLSSettings(profile openshiftconfigv1.TLSSecurityProfile) (TLSProfileOptions, error) {
	var (
		minTLSVersion openshiftconfigv1.TLSProtocolVersion
		ciphers       []string
	)

	switch profile.Type {
	case openshiftconfigv1.TLSProfileCustomType:
		if profile.Custom == nil {
			return TLSProfileOptions{}, kverrors.New("missing TLS custom profile spec")
		}
		minTLSVersion = profile.Custom.MinTLSVersion
		ciphers = profile.Custom.Ciphers
	case openshiftconfigv1.TLSProfileOldType, openshiftconfigv1.TLSProfileIntermediateType, openshiftconfigv1.TLSProfileModernType:
		spec := openshiftconfigv1.TLSProfiles[profile.Type]
		minTLSVersion = spec.MinTLSVersion
		ciphers = spec.Ciphers
	default:
		return TLSProfileOptions{}, kverrors.New("unable to determine tls profile settings %s", profile.Type)
	}

	// need to remap all ciphers to their respective IANA names used by Go
	return TLSProfileOptions{
		MinTLSVersion: string(minTLSVersion),
		Ciphers:       crypto.OpenSSLToIANACipherSuites(ciphers),
	}, nil
}
