package manifestutils

const (
	// TLSDir is the path that is mounted from the secret for TLS.
	TLSDir = "/var/run/tls"
	// CABundleDir is the path that is mounted from the configmap for TLS.
	CABundleDir = "/var/run/ca"
)
