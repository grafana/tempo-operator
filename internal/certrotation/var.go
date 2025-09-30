package certrotation

import (
	"fmt"

	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
	"github.com/grafana/tempo-operator/internal/manifests/naming"
)

const (
	// CertificateNotBeforeAnnotation contains the certificate expiration date in RFC3339 format.
	CertificateNotBeforeAnnotation = "tempo.grafana.com/certificate-not-before"
	// CertificateNotAfterAnnotation contains the certificate expiration date in RFC3339 format.
	CertificateNotAfterAnnotation = "tempo.grafana.com/certificate-not-after"
	// CertificateIssuer contains the common name of the certificate that signed another certificate.
	CertificateIssuer = "tempo.grafana.com/certificate-issuer"
	// CertificateHostnames contains the hostnames used by a signer.
	CertificateHostnames = "tempo.grafana.com/certificate-hostnames"
)

const (
	// CAFile is the file name of the certificate authority file.
	CAFile = "service-ca.crt"
)

// SigningCASecretName returns the tempostacks signing CA secret name.
func SigningCASecretName(stackName string) string {
	return fmt.Sprintf("tempo-%s-signing-ca", stackName)
}

// CABundleName returns the tempostacks ca bundle configmap name.
func CABundleName(stackName string) string {
	return fmt.Sprintf("tempo-%s-ca-bundle", stackName)
}

// TempoStackComponentCertSecretNames returns a map, with the key as the service name, and the value the secret name.
func TempoStackComponentCertSecretNames(stackName string) map[string]string {
	return map[string]string{
		naming.Name(manifestutils.DistributorComponentName, stackName):   naming.TLSSecretName(manifestutils.DistributorComponentName, stackName),
		naming.Name(manifestutils.IngesterComponentName, stackName):      naming.TLSSecretName(manifestutils.IngesterComponentName, stackName),
		naming.Name(manifestutils.QuerierComponentName, stackName):       naming.TLSSecretName(manifestutils.QuerierComponentName, stackName),
		naming.Name(manifestutils.QueryFrontendComponentName, stackName): naming.TLSSecretName(manifestutils.QueryFrontendComponentName, stackName),
		naming.Name(manifestutils.CompactorComponentName, stackName):     naming.TLSSecretName(manifestutils.CompactorComponentName, stackName),
		naming.Name(manifestutils.GatewayComponentName, stackName):       naming.TLSSecretName(manifestutils.GatewayComponentName, stackName),
	}
}

// MonolithicComponentCertSecretNames returns a map of component names to their respective TLS secret names based on input name.
func MonolithicComponentCertSecretNames(name string) map[string]string {
	return map[string]string{
		naming.Name(manifestutils.TempoMonolithComponentName, name): naming.TLSSecretName(manifestutils.TempoMonolithComponentName, name),
	}
}
