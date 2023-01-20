package certrotation

import (
	"fmt"
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

// SigningCASecretName returns the microservices signing CA secret name.
func SigningCASecretName(stackName string) string {
	return fmt.Sprintf("tempo-%s-signing-ca", stackName)
}

// CABundleName returns the microservices ca bundle configmap name.
func CABundleName(stackName string) string {
	return fmt.Sprintf("tempo-%s-ca-bundle", stackName)
}

// ComponentCertSecretNames returns a list of all tempo component certificate secret names.
func ComponentCertSecretNames(stackName string) []string {
	return []string{
		fmt.Sprintf("tempo-%s-distributor-http", stackName),
		fmt.Sprintf("tempo-%s-distributor-grpc", stackName),
		fmt.Sprintf("tempo-%s-ingester-http", stackName),
		fmt.Sprintf("tempo-%s-ingester-grpc", stackName),
		fmt.Sprintf("tempo-%s-querier-http", stackName),
		fmt.Sprintf("tempo-%s-querier-grpc", stackName),
		fmt.Sprintf("tempo-%s-query-frontend-http", stackName),
		fmt.Sprintf("tempo-%s-query-frontend-grpc", stackName),
		fmt.Sprintf("tempo-%s-compactor-http", stackName),
		fmt.Sprintf("tempo-%s-compactor-grpc", stackName),
	}
}
