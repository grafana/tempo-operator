package manifestutils

const (
	// httpTLSDir is the path that is mounted from the secret for TLS.
	httpTLSDir = "/var/run/tls/http"
	// grpcTLSDir is the path that is mounted from the secret for TLS.
	grpcTLSDir = "/var/run/tls/grpc"
	// tempo Microservices CABundleDir is the path that is mounted from the configmap for TLS.
	CABundleDir = "/var/run/ca"
)
