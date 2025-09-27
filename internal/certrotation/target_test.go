package certrotation

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

func TestCertificatesExpired(t *testing.T) {

	var (
		CACertValidity, _   = time.ParseDuration("10m")
		CACertRefresh, _    = time.ParseDuration("5m")
		CertValidity, _     = time.ParseDuration("2m")
		CertRefresh, _      = time.ParseDuration("1m")
		stackName           = "dev"
		stackNamespce       = "ns"
		invalidNotAfter, _  = time.Parse(time.RFC3339, "")
		invalidNotBefore, _ = time.Parse(time.RFC3339, "")
		rawCA, caBytes      = newTestCABundle(t, "dev-ca")
		cfg                 = configv1alpha1.BuiltInCertManagement{
			CACertValidity: metav1.Duration{Duration: CACertValidity},
			CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
			CertValidity:   metav1.Duration{Duration: CertValidity},
			CertRefresh:    metav1.Duration{Duration: CertRefresh},
		}
	)

	certBytes, keyBytes, err := rawCA.Config.GetPEMBytes()
	require.NoError(t, err)

	opts := Options{
		StackName:      stackName,
		StackNamespace: stackNamespce,
		Signer: SigningCA{
			RawCA: rawCA,
			Secret: &corev1.Secret{
				Data: map[string][]byte{
					corev1.TLSCertKey:       certBytes,
					corev1.TLSPrivateKeyKey: keyBytes,
				},
			},
		},
		CABundle: &corev1.ConfigMap{
			Data: map[string]string{
				CAFile: string(caBytes),
			},
		},
		RawCACerts: rawCA.Config.Certs,
	}
	err = ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	for _, name := range TempoStackComponentCertSecretNames(stackName) {
		cert := opts.Certificates[name]
		cert.Secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: stackNamespce,
				Annotations: map[string]string{
					CertificateIssuer:              "dev_ns@signing-ca@10000",
					CertificateNotAfterAnnotation:  invalidNotAfter.Format(time.RFC3339),
					CertificateNotBeforeAnnotation: invalidNotBefore.Format(time.RFC3339),
				},
			},
		}
		opts.Certificates[name] = cert
	}

	var expired *CertExpiredError
	err = CertificatesExpired(opts)

	require.Error(t, err)
	require.ErrorAs(t, err, &expired)
	require.Len(t, err.(*CertExpiredError).Reasons, 6)
}

func TestBuildTargetCertKeyPairSecrets_Create(t *testing.T) {
	var (
		CACertValidity, _ = time.ParseDuration("10m")
		CACertRefresh, _  = time.ParseDuration("5m")
		CertValidity, _   = time.ParseDuration("2m")
		CertRefresh, _    = time.ParseDuration("1m")
		rawCA, _          = newTestCABundle(t, "test-ca")
		cfg               = configv1alpha1.BuiltInCertManagement{
			CACertValidity: metav1.Duration{Duration: CACertValidity},
			CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
			CertValidity:   metav1.Duration{Duration: CertValidity},
			CertRefresh:    metav1.Duration{Duration: CertRefresh},
		}
	)

	opts := Options{
		StackName:      "dev",
		StackNamespace: "ns",
		Signer: SigningCA{
			RawCA: rawCA,
		},
		RawCACerts: rawCA.Config.Certs,
	}

	err := ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	objs, err := buildTargetCertKeyPairSecrets(opts)
	require.NoError(t, err)
	require.Len(t, objs, 6)
}

func TestBuildTargetCertKeyPairSecrets_Rotate(t *testing.T) {
	var (
		CACertValidity, _   = time.ParseDuration("10m")
		CACertRefresh, _    = time.ParseDuration("5m")
		CertValidity, _     = time.ParseDuration("2m")
		CertRefresh, _      = time.ParseDuration("1m")
		rawCA, _            = newTestCABundle(t, "test-ca")
		invalidNotAfter, _  = time.Parse(time.RFC3339, "")
		invalidNotBefore, _ = time.Parse(time.RFC3339, "")
		cfg                 = configv1alpha1.BuiltInCertManagement{
			CACertValidity: metav1.Duration{Duration: CACertValidity},
			CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
			CertValidity:   metav1.Duration{Duration: CertValidity},
			CertRefresh:    metav1.Duration{Duration: CertRefresh},
		}
	)

	opts := Options{
		StackName:      "dev",
		StackNamespace: "ns",
		Signer: SigningCA{
			RawCA: rawCA,
		},
		RawCACerts: rawCA.Config.Certs,
		Certificates: map[string]SelfSignedCertKey{
			"tempo-dev-ingester-mtls": {
				Secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-dev-ingester-mtls",
						Namespace: "ns",
						Annotations: map[string]string{
							CertificateIssuer:              "dev_ns@signing-ca@10000",
							CertificateNotAfterAnnotation:  invalidNotAfter.Format(time.RFC3339),
							CertificateNotBeforeAnnotation: invalidNotBefore.Format(time.RFC3339),
						},
					},
				},
			},
		},
	}
	err := ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	objs, err := buildTargetCertKeyPairSecrets(opts)
	require.NoError(t, err)
	require.Len(t, objs, 6)

	// Check serving certificate rotation
	s := objs[2].(*corev1.Secret)
	ss := opts.Certificates["tempo-dev-ingester-mtls"]

	require.NotEqual(t, s.Annotations[CertificateIssuer], ss.Secret.Annotations[CertificateIssuer])
	require.NotEqual(t, s.Annotations[CertificateNotAfterAnnotation], ss.Secret.Annotations[CertificateNotAfterAnnotation])
	require.NotEqual(t, s.Annotations[CertificateNotBeforeAnnotation], ss.Secret.Annotations[CertificateNotBeforeAnnotation])
	require.NotEqual(t, s.Annotations[CertificateHostnames], ss.Secret.Annotations[CertificateHostnames])
	require.NotEqual(t, string(s.Data[corev1.TLSCertKey]), string(ss.Secret.Data[corev1.TLSCertKey]))
	require.NotEqual(t, string(s.Data[corev1.TLSPrivateKeyKey]), string(ss.Secret.Data[corev1.TLSPrivateKeyKey]))

	certs, err := cert.ParseCertsPEM(s.Data[corev1.TLSCertKey])
	require.NoError(t, err)
	require.Contains(t, certs[0].ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	require.Contains(t, certs[0].ExtKeyUsage, x509.ExtKeyUsageServerAuth)
}
