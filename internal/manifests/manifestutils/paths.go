package manifestutils

const (
	// TLSDir is the path that is mounted from the secret for TLS.
	TLSDir = "/var/run/tls"

	// TempoInternalTLSCADir is the path that is mounted from the configmap for TLS.
	TempoInternalTLSCADir = "/var/run/ca"
	// TempoInternalTLSCertDir returns the mount path of the HTTP service certificates for communication between Tempo components.
	TempoInternalTLSCertDir = TLSDir + "/server"

	// ReceiverTLSCADir is the path that is mounted from the configmap for TLS for receiver.
	ReceiverTLSCADir = "/var/run/ca-receiver"
	// ReceiverTLSCertDir returns the mount path of the receivers certificates (for ingesting traces).
	ReceiverTLSCertDir = TLSDir + "/receiver"

	// ReceiverGRPCTLSCADir is the path that is mounted from the configmap for TLS for receiver.
	ReceiverGRPCTLSCADir = "/var/run/ca-receiver/grpc"
	// ReceiverGRPCTLSCertDir returns the mount path of the receivers certificates (for ingesting traces).
	ReceiverGRPCTLSCertDir = TLSDir + "/receiver/grpc"

	// ReceiverHTTPTLSCADir is the path that is mounted from the configmap for TLS for receiver.
	ReceiverHTTPTLSCADir = "/var/run/ca-receiver/http"
	// ReceiverHTTPTLSCertDir returns the mount path of the receivers certificates (for ingesting traces).
	ReceiverHTTPTLSCertDir = TLSDir + "/receiver/http"

	// StorageTLSCADir contains the CA file for accessing object storage.
	StorageTLSCADir = TLSDir + "/storage/ca"
	// StorageTLSCertDir contains the certificate and key file for accessing object storage.
	StorageTLSCertDir = TLSDir + "/storage/cert"
)
