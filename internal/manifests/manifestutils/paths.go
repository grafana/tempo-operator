package manifestutils

const (
	// TLSDir is the path that is mounted from the secret for TLS.
	TLSDir = "/var/run/tls"
	// CABundleDir is the path that is mounted from the configmap for TLS.
	CABundleDir = "/var/run/ca"
	// CAReceiver is the path that is mounted from the configmap for TLS for receiver.
	CAReceiver = "/var/run/ca-receiver"

	// TempoInternalTLSCertDir returns the mount path of the HTTP service certificates for communication between Tempo components.
	TempoInternalTLSCertDir = TLSDir + "/server"

	// ReceiverTLSCertDir returns the mount path of the receivers certificates (for ingesting traces).
	ReceiverTLSCertDir = TLSDir + "/receiver"

	// StorageTLSCADir contains the CA file for accessing object storage.
	StorageTLSCADir = TLSDir + "/storage/ca"
	// StorageTLSCertDir contains the certificate and key file for accessing object storage.
	StorageTLSCertDir = TLSDir + "/storage/cert"
)
