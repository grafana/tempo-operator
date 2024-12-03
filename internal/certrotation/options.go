package certrotation

import (
	"crypto/x509"
	"time"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"

	"github.com/openshift/library-go/pkg/crypto"
	corev1 "k8s.io/api/core/v1"
)

// ComponentCertificates is a map of TempoStack component names to TLS certificates.
type ComponentCertificates map[string]SelfSignedCertKey

// Options is a set of configuration values to use when
// building manifests for TempoStack certificates.
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
	return Rotation{
		CACertValidity:     cfg.CACertValidity.Duration,
		CACertRefresh:      cfg.CACertRefresh.Duration,
		TargetCertValidity: cfg.CertValidity.Duration,
		TargetCertRefresh:  cfg.CertRefresh.Duration,
	}, nil
}
