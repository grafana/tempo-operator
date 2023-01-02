package certrotation

import (
	"crypto/x509"
	"time"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"

	"github.com/ViaQ/logerr/v2/kverrors"
	"github.com/openshift/library-go/pkg/crypto"
	corev1 "k8s.io/api/core/v1"
)

// ComponentCertificates is a map of Microservices component names to TLS certificates.
type ComponentCertificates map[string]SelfSignedCertKey

// Options is a set of configuration values to use when
// building manifests for Microservices certificates.
type Options struct {
	Certificates   ComponentCertificates
	CABundle       *corev1.ConfigMap
	Signer         SigningCA
	StackName      string
	StackNamespace string
	RawCACerts     []*x509.Certificate
	Rotation       Rotation
}

// SigningCA rotates a self-signed signing CA stored in a secret. It creates a new one when
// - refresh duration is over
// - or 80% of validity is over
// - or the CA is expired.
type SigningCA struct {
	RawCA    *crypto.CA
	Secret   *corev1.Secret
	Rotation signerRotation
}

// SelfSignedCertKey rotates a key and cert signed by a signing CA and stores it in a secret.
//
// It creates a new one when
// - refresh duration is over
// - or 80% of validity is over
// - or the cert is expired.
// - or the signing CA changes.
type SelfSignedCertKey struct {
	Secret   *corev1.Secret
	Rotation certificateRotation
}

// Rotation define the validity/refresh pairs for certificates.
type Rotation struct {
	CACertValidity     time.Duration
	CACertRefresh      time.Duration
	TargetCertValidity time.Duration
	TargetCertRefresh  time.Duration
}

// ParseRotation builds a new RotationOptions struct from the feature gate string values.
func ParseRotation(cfg configv1alpha1.BuiltInCertManagement) (Rotation, error) {
	caValidity, err := time.ParseDuration(cfg.CACertValidity)
	if err != nil {
		return Rotation{}, kverrors.Wrap(err, "failed to parse CA validity duration", "value", cfg.CACertValidity)
	}

	caRefresh, err := time.ParseDuration(cfg.CACertRefresh)
	if err != nil {
		return Rotation{}, kverrors.Wrap(err, "failed to parse CA refresh duration", "value", cfg.CACertRefresh)
	}

	certValidity, err := time.ParseDuration(cfg.CertValidity)
	if err != nil {
		return Rotation{}, kverrors.Wrap(err, "failed to parse target certificate validity duration", "value", cfg.CertValidity)
	}

	certRefresh, err := time.ParseDuration(cfg.CertRefresh)
	if err != nil {
		return Rotation{}, kverrors.Wrap(err, "failed to parse target certificate refresh duration", "value", cfg.CertRefresh)
	}

	return Rotation{
		CACertValidity:     caValidity,
		CACertRefresh:      caRefresh,
		TargetCertValidity: certValidity,
		TargetCertRefresh:  certRefresh,
	}, nil
}
