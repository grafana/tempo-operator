package certrotation

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

func TestBuildAll(t *testing.T) {
	CACertValidity, _ := time.ParseDuration("10m")
	CACertRefresh, _ := time.ParseDuration("5m")
	CertValidity, _ := time.ParseDuration("2m")
	CertRefresh, _ := time.ParseDuration("1m")

	cfg := configv1alpha1.BuiltInCertManagement{
		CACertValidity: metav1.Duration{Duration: CACertValidity},
		CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
		CertValidity:   metav1.Duration{Duration: CertValidity},
		CertRefresh:    metav1.Duration{Duration: CertRefresh},
	}
	opts := Options{
		StackName:      "dev",
		StackNamespace: "ns",
	}
	err := ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	objs, err := BuildAll(opts)
	require.NoError(t, err)
	require.Len(t, objs, 8)

	for _, obj := range objs {
		objectName := obj.GetName()
		require.True(t, strings.HasPrefix(objectName, fmt.Sprintf("tempo-%s", opts.StackName)))
		require.Equal(t, obj.GetNamespace(), opts.StackNamespace)

		switch o := obj.(type) {
		case *corev1.Secret:
			require.Contains(t, o.Annotations, CertificateIssuer)
			require.Contains(t, o.Annotations, CertificateNotAfterAnnotation)
			require.Contains(t, o.Annotations, CertificateNotBeforeAnnotation)
		}
	}
}

func TestApplyDefaultSettings_EmptySecrets(t *testing.T) {

	CACertValidity, _ := time.ParseDuration("10m")
	CACertRefresh, _ := time.ParseDuration("5m")
	CertValidity, _ := time.ParseDuration("2m")
	CertRefresh, _ := time.ParseDuration("1m")

	cfg := configv1alpha1.BuiltInCertManagement{
		CACertValidity: metav1.Duration{Duration: CACertValidity},
		CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
		CertValidity:   metav1.Duration{Duration: CertValidity},
		CertRefresh:    metav1.Duration{Duration: CertRefresh},
	}

	opts := Options{
		StackName:      "tempostacks-dev",
		StackNamespace: "ns",
	}

	err := ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	cs := TempoStackComponentCertSecretNames(opts.StackName)

	for service, name := range cs {
		cert, ok := opts.Certificates[name]
		require.True(t, ok)
		require.NotEmpty(t, cert.Rotation)

		hostnames := []string{
			"localhost",
			fmt.Sprintf("%s.%s.svc.cluster.local", service, opts.StackNamespace),
			fmt.Sprintf("%s.%s.svc", service, opts.StackNamespace),
		}

		require.ElementsMatch(t, hostnames, cert.Rotation.Hostnames)
		require.Equal(t, defaultUserInfo, cert.Rotation.UserInfo)
		require.Nil(t, cert.Secret)
	}
}

func TestApplyDefaultSettings_ExistingSecrets(t *testing.T) {
	const (
		stackName      = "dev"
		stackNamespace = "ns"
	)

	CACertValidity, _ := time.ParseDuration("10m")
	CACertRefresh, _ := time.ParseDuration("5m")
	CertValidity, _ := time.ParseDuration("2m")
	CertRefresh, _ := time.ParseDuration("1m")

	cfg := configv1alpha1.BuiltInCertManagement{
		CACertValidity: metav1.Duration{Duration: CACertValidity},
		CACertRefresh:  metav1.Duration{Duration: CACertRefresh},
		CertValidity:   metav1.Duration{Duration: CertValidity},
		CertRefresh:    metav1.Duration{Duration: CertRefresh},
	}

	opts := Options{
		StackName:      stackName,
		StackNamespace: stackNamespace,
		Certificates:   ComponentCertificates{},
	}

	cs := TempoStackComponentCertSecretNames(opts.StackName)

	for _, name := range cs {
		opts.Certificates[name] = SelfSignedCertKey{
			Secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: stackNamespace,
					Annotations: map[string]string{
						CertificateNotBeforeAnnotation: "not-before",
						CertificateNotAfterAnnotation:  "not-after",
					},
				},
			},
		}
	}

	err := ApplyDefaultSettings(&opts, cfg, TempoStackComponentCertSecretNames(opts.StackName))
	require.NoError(t, err)

	for service, name := range cs {
		cert, ok := opts.Certificates[name]
		require.True(t, ok)
		require.NotEmpty(t, cert.Rotation)

		hostnames := []string{
			"localhost",
			fmt.Sprintf("%s.%s.svc.cluster.local", service, opts.StackNamespace),
			fmt.Sprintf("%s.%s.svc", service, opts.StackNamespace),
		}

		require.ElementsMatch(t, hostnames, cert.Rotation.Hostnames)
		require.Equal(t, defaultUserInfo, cert.Rotation.UserInfo)

		require.NotNil(t, cert.Secret)
	}
}
